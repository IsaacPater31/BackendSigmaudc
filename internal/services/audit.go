// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package services

import (
	"log"

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

// GetAuditLogs retorna los últimos registros de auditoría.
// Si limit está vacío se usa el valor por defecto de constants.DefaultAuditLimit.
func (s *AuditService) GetAuditLogs(limit string) ([]models.AuditLog, error) {
	if limit == "" {
		limit = constants.DefaultAuditLimit
	}
	return s.repo.GetAuditLogs(limit)
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
