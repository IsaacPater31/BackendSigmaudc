// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey,
// Isaac David Paternina Noble, Harry Perez Perea, Camilo Rodríguez Garay,
// José Félix Bustamante Soto y Juan Felipe Eraso Navarro.
// Todos los derechos reservados.

// Package handlers – AuditHandler
// Expone el endpoint de consulta de logs de auditoría del sistema.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/httpx"
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
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	logs, err := h.service.GetAuditLogs(r.URL.Query().Get("limit"))
	if err != nil {
		httpx.Error(w, "Error fetching audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
