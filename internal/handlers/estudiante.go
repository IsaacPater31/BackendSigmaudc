package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
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
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	datos, err := h.service.GetDatosEstudiante(claims.Sub)
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		http.Error(w, "Estudiante no encontrado", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(datos)
}

func (h *EstudianteHandler) UpdateDatosEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	var payload models.UpdateDatosRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Payload inválido", http.StatusBadRequest)
		return
	}
	err = h.service.UpdateDatosEstudiante(claims.Sub, payload)
	if errors.Is(err, services.ErrSexoInvalido) {
		http.Error(w, "Valor de sexo inválido", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		http.Error(w, "estudiante no encontrado", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *EstudianteHandler) SubirFotoEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseMultipartForm(constants.MaxFotoBytes); err != nil {
		http.Error(w, "No se pudo procesar el archivo", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("foto")
	if err != nil {
		http.Error(w, "Debes subir una imagen", http.StatusBadRequest)
		return
	}
	defer file.Close()

	photoURL, err := h.service.UploadEstudianteFoto(claims.Sub, file, header.Filename)
	if errors.Is(err, services.ErrFormatoImagenInvalido) {
		http.Error(w, "Formato de imagen no permitido", http.StatusBadRequest)
		return
	}
	if errors.Is(err, services.ErrEstudianteNoEncontrado) {
		http.Error(w, "estudiante no encontrado", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, "No se pudo guardar la imagen", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"foto_perfil": photoURL})
}
