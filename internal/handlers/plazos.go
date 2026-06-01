package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/andrxsq/SIGMAUDC/internal/utils"
	"github.com/gorilla/mux"
)

// PlazosHandler solo orquesta peticiones HTTP del modulo de periodos/plazos.
type PlazosHandler struct {
	service *services.PlazosService
}

func NewPlazosHandler(service *services.PlazosService) *PlazosHandler {
	return &PlazosHandler{service: service}
}

func (h *PlazosHandler) GetPeriodos(w http.ResponseWriter, r *http.Request) {
	periodos, err := h.service.GetPeriodos()
	if err != nil {
		http.Error(w, "Error fetching periodos", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, periodos)
}

func (h *PlazosHandler) GetPeriodoActivo(w http.ResponseWriter, r *http.Request) {
	periodo, err := h.service.GetPeriodoActivo()
	if err != nil {
		http.Error(w, "Error fetching periodo activo", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, periodo)
}

func (h *PlazosHandler) GetActivePeriodoPlazos(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.service.GetActivePeriodoPlazos(claims.ProgramaID)
	if err != nil {
		http.Error(w, "Error fetching plazos", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *PlazosHandler) CreatePeriodo(w http.ResponseWriter, r *http.Request) {
	var req models.CreatePeriodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	periodo, err := h.service.CreatePeriodo(req)
	switch {
	case errors.Is(err, services.ErrSemestreInvalido):
		http.Error(w, "Semestre debe ser 1 o 2", http.StatusBadRequest)
		return
	case errors.Is(err, services.ErrPeriodoDuplicado):
		http.Error(w, "Ya existe un periodo con ese año y semestre", http.StatusConflict)
		return
	case err != nil:
		http.Error(w, "Error creating periodo", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, periodo)
}

func (h *PlazosHandler) UpdatePeriodo(w http.ResponseWriter, r *http.Request) {
	periodoID, err := parseIntParam(r, "id")
	if err != nil {
		http.Error(w, "Invalid periodo ID", http.StatusBadRequest)
		return
	}

	var req models.UpdatePeriodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	periodo, err := h.service.UpdatePeriodo(periodoID, req)
	switch {
	case errors.Is(err, services.ErrPeriodoNotFound):
		http.Error(w, "Periodo not found", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrPeriodoArchivadoNoActivo):
		http.Error(w, "No se puede activar un periodo archivado", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, "Error updating periodo", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, periodo)
}

func (h *PlazosHandler) DeletePeriodo(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Los periodos no se eliminan. Utiliza el archivado para mantener el historial.", http.StatusMethodNotAllowed)
}

func (h *PlazosHandler) GetPlazos(w http.ResponseWriter, r *http.Request) {
	periodoID, err := parseIntParam(r, "periodo_id")
	if err != nil {
		http.Error(w, "Invalid periodo ID", http.StatusBadRequest)
		return
	}

	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	plazos, err := h.service.GetPlazos(periodoID, claims.ProgramaID)
	if err != nil {
		http.Error(w, "Error fetching plazos", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, plazos)
}

func (h *PlazosHandler) UpdatePlazos(w http.ResponseWriter, r *http.Request) {
	periodoID, err := parseIntParam(r, "periodo_id")
	if err != nil {
		http.Error(w, "Invalid periodo ID", http.StatusBadRequest)
		return
	}

	var req models.UpdatePlazosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		http.Error(w, "Solo un jefe departamental puede modificar plazos", http.StatusForbidden)
		return
	}

	audit := services.AuditMetadata{
		UsuarioID:  claims.Sub,
		IP:         utils.GetIPAddress(r),
		UserAgent:  r.UserAgent(),
		ProgramaID: claims.ProgramaID,
	}

	plazos, err := h.service.UpdatePlazos(periodoID, claims.ProgramaID, req, audit)
	switch {
	case errors.Is(err, services.ErrPeriodoNotFound):
		http.Error(w, "Periodo not found", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrPeriodoArchivado):
		http.Error(w, "No se pueden modificar plazos de un periodo archivado", http.StatusBadRequest)
		return
	case errors.Is(err, services.ErrPeriodoInactivo):
		http.Error(w, "No se pueden modificar plazos de un periodo inactivo", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, "Error updating plazos", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, plazos)
}

func (h *PlazosHandler) GetPeriodosConPlazos(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	periodos, err := h.service.GetPeriodosConPlazos(claims.ProgramaID)
	if err != nil {
		http.Error(w, "Error fetching periodos", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, periodos)
}

func parseIntParam(r *http.Request, key string) (int, error) {
	return strconv.Atoi(mux.Vars(r)[key])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
