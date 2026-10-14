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

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/httpx"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/services"
)

type EstudianteHandler struct {
	service *services.ProfileService
}

func NewEstudianteHandler(service *services.ProfileService) *EstudianteHandler {
	return &EstudianteHandler{service: service}
}

func (h *EstudianteHandler) GetDatosEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	datos, err := h.service.GetDatosEstudiante(claims.Sub)
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		httpx.Error(w, "Estudiante no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(datos)
}

func (h *EstudianteHandler) UpdateDatosEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var payload models.UpdateDatosRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		httpx.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}
	err = h.service.UpdateDatosEstudiante(claims.Sub, payload)
	if errors.Is(err, services.ErrSexoInvalido) {
		httpx.Error(w, "Valor de sexo inválido", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		httpx.Error(w, "estudiante no encontrado", http.StatusBadRequest)
		return
	}
	if err != nil {
		httpx.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EstudianteHandler) SubirFotoEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		httpx.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, constants.MaxFotoBytes+(64<<10))
	if err := r.ParseMultipartForm(constants.MaxFotoBytes); err != nil {
		httpx.Error(w, "la imagen excede el tamaño máximo permitido (8 MB)", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("foto")
	if err != nil {
		httpx.Error(w, "Debes subir una imagen", http.StatusBadRequest)
		return
	}
	defer file.Close()
	if header.Size > constants.MaxFotoBytes {
		httpx.Error(w, "la imagen excede el tamaño máximo permitido (8 MB)", http.StatusBadRequest)
		return
	}

	photoURL, err := h.service.UploadEstudianteFoto(claims.Sub, file, header.Filename)
	if errors.Is(err, services.ErrFormatoImagenInvalido) {
		httpx.Error(w, "Formato de imagen no permitido", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		httpx.Error(w, "estudiante no encontrado", http.StatusBadRequest)
		return
	}
	if err != nil {
		httpx.Error(w, "No se pudo guardar la imagen", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"foto_perfil": photoURL})
}
