// Package services contiene la lógica de negocio compartida entre handlers.
// Separa la responsabilidad de los handlers (orquestar peticiones HTTP)
// de las operaciones que involucran reglas de negocio transversales.
package services

import (
	"database/sql"
	"log"
)

// AuditoriaService encapsula toda la lógica de registro de eventos de auditoría.
//
// Principios aplicados:
//   - SRP: su única responsabilidad es registrar eventos en la tabla auditoria.
//   - GRASP Information Expert: es el único que sabe cómo persistir una auditoría.
//   - DIP: los handlers dependen de este servicio, no reimplementan la lógica.
type AuditoriaService struct {
	db *sql.DB
}

// NewAuditoriaService crea una nueva instancia del servicio de auditoría.
func NewAuditoriaService(db *sql.DB) *AuditoriaService {
	return &AuditoriaService{db: db}
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
// Esta función nunca falla la operación principal: ante un error de BD,
// solo lo registra en el log del servidor.
func (s *AuditoriaService) Registrar(usuarioID int, accion, descripcion, ip, userAgent string) {
	var userID sql.NullInt64
	if usuarioID > 0 {
		userID = sql.NullInt64{Int64: int64(usuarioID), Valid: true}
	}

	query := `INSERT INTO auditoria (usuario_id, accion, descripcion, ip, user_agent)
	          VALUES ($1, $2, $3, $4, $5)`

	if _, err := s.db.Exec(query, userID, accion, descripcion, ip, userAgent); err != nil {
		// El error de auditoría NO debe interrumpir la operación principal.
		log.Printf("[AuditoriaService] Error registrando evento '%s': %v", accion, err)
	}
}
