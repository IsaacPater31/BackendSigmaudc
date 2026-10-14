// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/httpx"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/andrxsq/SIGMAUDC/internal/utils"
	"github.com/gorilla/mux"
)

type DocumentosHandler struct {
	service *services.DocumentosService
}

func NewDocumentosHandler(service *services.DocumentosService) *DocumentosHandler {
	return &DocumentosHandler{service: service}
}

func (h *DocumentosHandler) GetDocumentosEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	resp, err := h.service.GetDocumentosEstudiante(claims.Sub, claims.ProgramaID)
	if errors.Is(err, services.ErrEstudianteNoEncontradoDoc) {
		httpx.Error(w, "Estudiante no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *DocumentosHandler) SubirDocumento(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseMultipartForm(constants.MaxDocumentoBytes)
	tipoDocumento := r.FormValue("tipo_documento")
	file, header, err := r.FormFile("archivo")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "archivo requerido"})
		return
	}
	defer file.Close()

	resp, err := h.service.SubirDocumento(claims.Sub, claims.ProgramaID, tipoDocumento, file, header, utils.GetIPAddress(r), r.UserAgent())
	switch {
	case errors.Is(err, services.ErrDocumentoPlazo):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "el plazo de documentos no está disponible"})
		return
	case errors.Is(err, services.ErrEstudianteNoEncontradoDoc):
		httpx.Error(w, "Estudiante no encontrado", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrDocumentoTipoInvalido):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "tipo_documento inválido. Debe ser 'certificado_eps' o 'comprobante_matricula'"})
		return
	case errors.Is(err, services.ErrDocumentoArchivoInvalido):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "formato o tamaño de archivo inválido"})
		return
	case errors.Is(err, services.ErrDocumentoReviewInvalida):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Documento ya subido o aprobado"})
		return
	case err != nil:
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *DocumentosHandler) GetDocumentosPorPrograma(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	page, pageSize := parsePagination(r, "page", "page_size")
	estado := strings.TrimSpace(r.URL.Query().Get("estado"))
	if estado != "" && estado != constants.EstadoDocPendiente && estado != constants.EstadoDocAprobado && estado != constants.EstadoDocRechazado {
		httpx.Error(w, "Invalid estado filter", http.StatusBadRequest)
		return
	}

	documentos, err := h.service.GetDocumentosPorProgramaPaginated(claims.ProgramaID, estado, page, pageSize)
	if err != nil {
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(documentos)
}

func (h *DocumentosHandler) RevisarDocumento(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	docID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		httpx.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}
	var req models.RevisarDocumentoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.RevisarDocumento(claims.Sub, claims.ProgramaID, docID, req, utils.GetIPAddress(r), r.UserAgent())
	switch {
	case errors.Is(err, services.ErrJefeNoEncontradoDoc):
		httpx.Error(w, "Jefe departamental no encontrado", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrDocumentoNoEncontrado):
		httpx.Error(w, "Documento no encontrado", http.StatusNotFound)
		return
	case errors.Is(err, services.ErrDocumentoForbidden):
		httpx.Error(w, "Forbidden: documento no pertenece a tu programa", http.StatusForbidden)
		return
	case errors.Is(err, services.ErrDocumentoReviewInvalida):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "estado/observación inválidos"})
		return
	case err != nil:
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
