// Package handlers – AuthHandler
// Gestiona la autenticación de usuarios: login y configuración inicial de contraseña.
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/middleware"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/andrxsq/SIGMAUDC/internal/utils"
)

// AuthHandler gestiona las peticiones de autenticación del sistema.
//
// Principios aplicados:
//   - SRP: solo se ocupa de autenticación; la auditoría es delegada al AuditoriaService.
//   - DIP: depende de AuditoriaService (abstracción), no implementa la lógica de auditoría.
type AuthHandler struct {
	service *services.AuthService
}

// NewAuthHandler crea una nueva instancia de AuthHandler con sus dependencias inyectadas.
func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login autentica a un usuario por código y contraseña.
//
// POST /auth/login
// Body: models.LoginRequest
//
// Flujos posibles:
//   - Usuario no existe → 401 con errorType "user_not_found".
//   - Usuario sin contraseña → 200 con requiresPasswordSetup: true.
//   - Contraseña incorrecta → 401 con errorType "wrong_password".
//   - Éxito → 200 con token JWT.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Login(req, utils.GetIPAddress(r), r.UserAgent())
	if errors.Is(err, services.ErrAuthUserNotFound) || errors.Is(err, services.ErrAuthWrongPassword) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if errors.Is(err, services.ErrAuthNeedsPasswordSetup) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// SetPassword establece la contraseña inicial de un usuario en su primer acceso.
//
// POST /auth/set-password
// Body: models.SetPasswordRequest
//
// Valida que el userId, código y email coincidan con los datos en BD,
// y que el usuario aún no tenga contraseña configurada.
func (h *AuthHandler) SetPassword(w http.ResponseWriter, r *http.Request) {
	var req models.SetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.SetPassword(req, utils.GetIPAddress(r), r.UserAgent())
	if errors.Is(err, services.ErrAuthUserNotFound) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if errors.Is(err, services.ErrAuthCodigoMismatch) || errors.Is(err, services.ErrAuthEmailMismatch) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if errors.Is(err, services.ErrAuthPasswordExists) || (!resp.Success && err != nil) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if err != nil {
		http.Error(w, "Error updating password", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// GetCurrentUser retorna los datos del usuario autenticado extraídos del JWT + BD.
//
// GET /api/me
// Requiere: cabecera Authorization con token JWT válido.
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	usuario, err := h.service.GetCurrentUser(claims.Sub)
	if err != nil {
		http.Error(w, "Error fetching user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usuario)
}
