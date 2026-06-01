// Package handlers – AuditHandler
// Expone el endpoint de consulta de logs de auditoría del sistema.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/services"
)

// AuditHandler gestiona las peticiones relacionadas con el log de auditoría.
type AuditHandler struct {
	service *services.AuditService
}

// NewAuditHandler crea una nueva instancia de AuditHandler.
func NewAuditHandler(service *services.AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

// GetAuditLogs retorna los registros de auditoría más recientes.
//
// Query param opcional:
//   - limit (string): número máximo de registros a retornar. Default: 50.
//
// Responde con un array JSON de models.AuditLog.
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.service.GetAuditLogs(r.URL.Query().Get("limit"))
	if err != nil {
		http.Error(w, "Error fetching audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
