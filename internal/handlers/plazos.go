// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey,
// Isaac David Paternina Noble, Harry Perez Perea, Camilo Rodríguez Garay,
// José Félix Bustamante Soto y Juan Felipe Eraso Navarro.
// Todos los derechos reservados.

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/httpx"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/andrxsq/SIGMAUDC/internal/utils"
	"github.com/gorilla/mux"
)

// PlazosHandler orquesta peticiones HTTP de plazos del periodo activo.
type PlazosHandler struct {
	service *services.PlazosService
}

func NewPlazosHandler(service *services.PlazosService) *PlazosHandler {
	return &PlazosHandler{service: service}
}

func (h *PlazosHandler) GetActivePeriodoPlazos(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	resp, err := h.service.GetActivePeriodoPlazos(claims.ProgramaID)
	if err != nil {
		httpx.Error(w, "Error fetching plazos", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *PlazosHandler) UpdatePlazos(w http.ResponseWriter, r *http.Request) {
	periodoID, err := parseIntParam(r, "periodo_id")
	if err != nil {
		httpx.Error(w, "Invalid periodo ID", http.StatusBadRequest)
		return
	}

	var req models.UpdatePlazosRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		httpx.Error(w, "Solo un jefe departamental puede modificar plazos", http.StatusForbidden)
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
		httpx.Error(w, "Periodo not found", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrPeriodoArchivado):
		httpx.Error(w, "No se pueden modificar plazos de un periodo archivado", http.StatusBadRequest)
		return
	case errors.Is(err, services.ErrPeriodoInactivo):
		httpx.Error(w, "No se pueden modificar plazos de un periodo inactivo", http.StatusBadRequest)
		return
	case err != nil:
		httpx.Error(w, "Error updating plazos", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, plazos)
}

func parseIntParam(r *http.Request, key string) (int, error) {
	return strconv.Atoi(mux.Vars(r)[key])
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
