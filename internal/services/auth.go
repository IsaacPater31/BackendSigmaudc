package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
	"github.com/andrxsq/SIGMAUDC/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAuthUserNotFound       = errors.New("user not found")
	ErrAuthWrongPassword      = errors.New("wrong password")
	ErrAuthNeedsPasswordSetup = errors.New("password setup required")
	ErrAuthEmailMismatch      = errors.New("email mismatch")
	ErrAuthCodigoMismatch     = errors.New("codigo mismatch")
	ErrAuthPasswordExists     = errors.New("password already exists")
)

type AuthService struct {
	repo      *repositories.AuthRepository
	auditoria *AuditoriaService
	jwtSecret string
}

func NewAuthService(repo *repositories.AuthRepository, auditoria *AuditoriaService, jwtSecret string) *AuthService {
	return &AuthService{repo: repo, auditoria: auditoria, jwtSecret: jwtSecret}
}

func (s *AuthService) Login(req models.LoginRequest, ip, userAgent string) (models.LoginResponse, error) {
	usuario, err := s.repo.GetUsuarioByCodigo(req.Codigo)
	if errors.Is(err, sql.ErrNoRows) {
		s.auditoria.Registrar(0, "login_fallido", "Usuario no encontrado: "+req.Codigo, ip, userAgent)
		return models.LoginResponse{
			Message:   "El código de usuario no existe en el sistema",
			ErrorType: "user_not_found",
		}, ErrAuthUserNotFound
	}
	if err != nil {
		s.auditoria.Registrar(0, "login_fallido", "Error de base de datos: "+err.Error(), ip, userAgent)
		return models.LoginResponse{
			Message:   "Error de conexión con el servidor. Por favor intenta más tarde",
			ErrorType: "connection_error",
		}, err
	}

	if !usuario.PasswordHash.Valid || usuario.PasswordHash.String == "" {
		s.auditoria.Registrar(usuario.ID, "login_fallido", "Intento de login sin contraseña configurada", ip, userAgent)
		return models.LoginResponse{
			RequiresPasswordSetup: true,
			UserID:                usuario.ID,
		}, ErrAuthNeedsPasswordSetup
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.PasswordHash.String), []byte(req.Password)); err != nil {
		s.auditoria.Registrar(usuario.ID, "login_fallido", "Contraseña incorrecta", ip, userAgent)
		return models.LoginResponse{
			Message:   "La contraseña ingresada es incorrecta",
			ErrorType: "wrong_password",
		}, ErrAuthWrongPassword
	}

	token, err := s.generateJWT(*usuario)
	if err != nil {
		return models.LoginResponse{}, err
	}
	s.auditoria.Registrar(usuario.ID, "login_exitoso", "Inicio de sesión exitoso", ip, userAgent)
	return models.LoginResponse{Token: token}, nil
}

func (s *AuthService) SetPassword(req models.SetPasswordRequest, ip, userAgent string) (models.SetPasswordResponse, error) {
	usuario, err := s.repo.GetUsuarioByID(req.UserID)
	if errors.Is(err, sql.ErrNoRows) {
		return models.SetPasswordResponse{Success: false, Message: "Usuario no encontrado"}, ErrAuthUserNotFound
	}
	if err != nil {
		return models.SetPasswordResponse{}, err
	}

	if strings.TrimSpace(req.Codigo) != strings.TrimSpace(usuario.Codigo) {
		s.auditoria.Registrar(req.UserID, "verificacion_codigo_fallida", "Código no coincide al crear contraseña", ip, userAgent)
		return models.SetPasswordResponse{Success: false, Message: "El código ingresado no coincide con el código del usuario"}, ErrAuthCodigoMismatch
	}

	if usuario.PasswordHash.Valid && usuario.PasswordHash.String != "" {
		s.auditoria.Registrar(req.UserID, "intento_crear_contraseña_existente", "Intento de crear contraseña cuando ya existe", ip, userAgent)
		return models.SetPasswordResponse{Success: false, Message: "Este usuario ya tiene una contraseña configurada"}, ErrAuthPasswordExists
	}

	emailIngresado := strings.ToLower(strings.TrimSpace(req.Email))
	emailBD := strings.ToLower(strings.TrimSpace(usuario.Email))
	if emailIngresado == "" || emailBD == "" || emailIngresado != emailBD {
		descripcion := fmt.Sprintf("Correo no coincide: ingresado='%s', esperado='%s', usuario_id=%d", emailIngresado, emailBD, req.UserID)
		s.auditoria.Registrar(req.UserID, "verificacion_correo_fallida", descripcion, ip, userAgent)
		return models.SetPasswordResponse{Success: false, Message: "El correo electrónico no coincide con el registrado para este código"}, ErrAuthEmailMismatch
	}

	valid, message := utils.ValidatePassword(req.NewPassword)
	if !valid {
		return models.SetPasswordResponse{Success: false, Message: message}, errors.New("password invalid")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return models.SetPasswordResponse{}, err
	}
	if err := s.repo.UpdatePassword(req.UserID, string(hashedPassword)); err != nil {
		return models.SetPasswordResponse{}, err
	}

	token, err := s.generateJWT(*usuario)
	if err != nil {
		return models.SetPasswordResponse{}, err
	}
	s.auditoria.Registrar(usuario.ID, "cambio_contraseña", "Creación de contraseña inicial", ip, userAgent)
	return models.SetPasswordResponse{Success: true, Token: token}, nil
}

func (s *AuthService) GetCurrentUser(userID int) (*models.Usuario, error) {
	return s.repo.GetCurrentUser(userID)
}

func (s *AuthService) generateJWT(usuario models.Usuario) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &models.JWTClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.Itoa(usuario.ID),
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Sub:        usuario.ID,
		Codigo:     usuario.Codigo,
		Rol:        usuario.Rol,
		ProgramaID: usuario.ProgramaID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
