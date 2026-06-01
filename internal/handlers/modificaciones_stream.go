package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type modificacionesEventBroker struct {
	mu          sync.RWMutex
	subscribers map[int]map[chan []byte]struct{}
}

func newModificacionesEventBroker() *modificacionesEventBroker {
	return &modificacionesEventBroker{
		subscribers: make(map[int]map[chan []byte]struct{}),
	}
}

func (b *modificacionesEventBroker) subscribe(programaID int) chan []byte {
	ch := make(chan []byte, 16)
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, ok := b.subscribers[programaID]; !ok {
		b.subscribers[programaID] = make(map[chan []byte]struct{})
	}
	b.subscribers[programaID][ch] = struct{}{}
	return ch
}

func (b *modificacionesEventBroker) unsubscribe(programaID int, ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()

	programaSubs, ok := b.subscribers[programaID]
	if !ok {
		return
	}
	if _, exists := programaSubs[ch]; !exists {
		return
	}
	delete(programaSubs, ch)
	close(ch)
	if len(programaSubs) == 0 {
		delete(b.subscribers, programaID)
	}
}

func (b *modificacionesEventBroker) publish(programaID int, payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error serializando evento SSE de modificaciones: %v", err)
		return
	}

	b.mu.RLock()
	programaSubs, ok := b.subscribers[programaID]
	if !ok {
		b.mu.RUnlock()
		return
	}
	// Snapshot para evitar mantener lock durante envíos.
	targets := make([]chan []byte, 0, len(programaSubs))
	for ch := range programaSubs {
		targets = append(targets, ch)
	}
	b.mu.RUnlock()

	for _, ch := range targets {
		select {
		case ch <- data:
		default:
			// Si el consumidor está lento, descartamos para no bloquear.
		}
	}
}

var modificacionesBroker = newModificacionesEventBroker()

func (h *MatriculaHandler) emitModificacionesEvent(programaID int, eventType string, payload map[string]interface{}) {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["event_type"] = eventType
	payload["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	modificacionesBroker.publish(programaID, payload)
}

// StreamModificacionesEvents expone eventos SSE para cambios de solicitudes/cupos.
func (h *MatriculaHandler) StreamModificacionesEvents(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Usuario no autenticado", http.StatusUnauthorized)
		return
	}

	programaID := claims.ProgramaID
	if programaID <= 0 {
		http.Error(w, "Programa inválido para stream", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no soportado", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := modificacionesBroker.subscribe(programaID)
	defer modificacionesBroker.unsubscribe(programaID, ch)

	// Evento inicial para confirmar conexión.
	fmt.Fprintf(w, "event: ready\ndata: {\"event_type\":\"ready\"}\n\n")
	flusher.Flush()

	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-ch:
			fmt.Fprintf(w, "event: modificaciones\ndata: %s\n\n", data)
			flusher.Flush()
		case <-keepAlive.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}
