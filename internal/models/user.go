// Package models define las estructuras de datos compartidas por la aplicación:
// entidades de negocio, DTOs de request/response y claims del token JWT.
package models

import (
	"database/sql"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ─── Entidades de usuario ─────────────────────────────────────────────────────

// Usuario representa un usuario registrado en el sistema.
type Usuario struct {
	ID             int            `json:"id"`
	Codigo         string         `json:"codigo"`
	Email          string         `json:"email"`
	PasswordHash   sql.NullString `json:"-"`
	Rol            string         `json:"rol"`
	ProgramaID     int            `json:"programa_id"`
	ProgramaNombre string         `json:"programa_nombre,omitempty"`
	Nombre         string         `json:"nombre,omitempty"`
	Apellido       string         `json:"apellido,omitempty"`
}

// ─── DTOs de autenticación ────────────────────────────────────────────────────

// LoginRequest es el body esperado en el endpoint POST /auth/login.
type LoginRequest struct {
	Codigo   string `json:"codigo"`
	Password string `json:"password"`
}

// LoginResponse es la respuesta del endpoint POST /auth/login.
// Los campos omitempty permiten retornar solo los relevantes según el flujo.
type LoginResponse struct {
	Token                string `json:"token,omitempty"`
	RequiresPasswordSetup bool   `json:"requiresPasswordSetup,omitempty"`
	UserID               int    `json:"userId,omitempty"`
	Message              string `json:"message,omitempty"`
	// ErrorType clasifica el tipo de error: "user_not_found", "wrong_password", "connection_error".
	ErrorType string `json:"errorType,omitempty"`
}

// SetPasswordRequest es el body esperado en POST /auth/set-password.
type SetPasswordRequest struct {
	UserID      int    `json:"userId"`
	Codigo      string `json:"codigo"`
	Email       string `json:"email"`
	NewPassword string `json:"newPassword"`
}

// SetPasswordResponse es la respuesta del endpoint POST /auth/set-password.
type SetPasswordResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message,omitempty"`
}

// ─── Auditoría ────────────────────────────────────────────────────────────────

// Auditoria representa un registro de evento en la tabla auditoria.
type Auditoria struct {
	ID          int           `json:"id"`
	UsuarioID   sql.NullInt64 `json:"usuario_id"`
	Accion      string        `json:"accion"`
	Descripcion string        `json:"descripcion"`
	Fecha       time.Time     `json:"fecha"`
	IP          string        `json:"ip"`
	UserAgent   string        `json:"user_agent"`
}

// AuditLog es el DTO de respuesta para el endpoint GET /api/audit.
// Usa *int para UsuarioID para serializar como null cuando no aplica.
type AuditLog struct {
	ID          int    `json:"id"`
	UsuarioID   *int   `json:"usuario_id"`
	Accion      string `json:"accion"`
	Descripcion string `json:"descripcion"`
	Fecha       string `json:"fecha"`
	IP          string `json:"ip"`
	UserAgent   string `json:"user_agent"`
}

// ─── JWT ──────────────────────────────────────────────────────────────────────

// JWTClaims define los campos personalizados que se incluyen en el token JWT.
// Embebe RegisteredClaims para los campos estándar (exp, iat, sub).
type JWTClaims struct {
	jwt.RegisteredClaims
	Sub        int    `json:"sub"`
	Codigo     string `json:"codigo"`
	Rol        string `json:"rol"`
	ProgramaID int    `json:"programa_id"`
}
