// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey,
// Isaac David Paternina Noble, Harry Perez Perea, Camilo Rodríguez Garay,
// José Félix Bustamante Soto y Juan Felipe Eraso Navarro.
// Todos los derechos reservados.

package services

import (
	"log"
	"strconv"
	"strings"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

// AuditService centraliza la consulta y el registro de eventos de auditoría.
//
// Principios aplicados:
//   - SRP: única responsabilidad sobre la tabla auditoria (leer y registrar).
//   - GRASP Information Expert: es el único que sabe cómo persistir una auditoría.
//   - DIP: handlers y servicios dependen de este servicio, no acceden a la BD.
type AuditService struct {
	repo *repositories.AuditRepository
}

// NewAuditService crea una nueva instancia del servicio de auditoría.
func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// maxAuditLimit acota la cantidad de registros consultables por petición.
const maxAuditLimit = 500

// GetAuditLogs retorna los últimos registros de auditoría.
// El parámetro limit se interpreta como entero; valores vacíos, no numéricos
// o fuera de rango se reemplazan por constants.DefaultAuditLimit o se acotan
// a maxAuditLimit, evitando errores de SQL ante entradas inválidas.
func (s *AuditService) GetAuditLogs(limit string) ([]models.AuditLog, error) {
	return s.repo.GetAuditLogs(strconv.Itoa(parseAuditLimit(limit)))
}

// parseAuditLimit convierte el query param limit en un entero válido.
func parseAuditLimit(limit string) int {
	if limit == "" {
		v, err := strconv.Atoi(constants.DefaultAuditLimit)
		if err != nil {
			return 50
		}
		return v
	}
	v, err := strconv.Atoi(strings.TrimSpace(limit))
	if err != nil || v <= 0 {
		v, _ = strconv.Atoi(constants.DefaultAuditLimit)
	}
	if v > maxAuditLimit {
		v = maxAuditLimit
	}
	return v
}

// Registrar inserta un evento de auditoría en la base de datos.
//
// Parámetros:
//   - usuarioID: ID del usuario que realizó la acción. Si es 0, se registra como anónimo.
//   - accion: identificador corto de la acción (ej. "login_exitoso", "subida_documento").
//   - descripcion: texto libre con el detalle del evento.
//   - ip: dirección IP del cliente.
//   - userAgent: cabecera User-Agent del cliente.
//
// Esta función nunca interrumpe la operación principal: ante un error de BD
// solo lo registra en el log del servidor.
func (s *AuditService) Registrar(usuarioID int, accion, descripcion, ip, userAgent string) {
	if err := s.repo.RegistrarEvento(usuarioID, accion, descripcion, ip, userAgent); err != nil {
		log.Printf("[AuditService] error registrando evento '%s': %v", accion, err)
	}
}
