// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

// AuditRepository encapsula el acceso a la tabla auditoria.
type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// RegistrarEvento inserta un evento de auditoría en la base de datos.
// Si usuarioID es 0, el evento se registra como anónimo (usuario_id NULL).
func (r *AuditRepository) RegistrarEvento(usuarioID int, accion, descripcion, ip, userAgent string) error {
	var userID sql.NullInt64
	if usuarioID > 0 {
		userID = sql.NullInt64{Int64: int64(usuarioID), Valid: true}
	}

	query := `INSERT INTO auditoria (usuario_id, accion, descripcion, ip, user_agent)
	          VALUES ($1, $2, $3, $4, $5)`

	_, err := r.db.Exec(query, userID, accion, descripcion, ip, userAgent)
	return err
}

// GetAuditLogs retorna los últimos registros de auditoría ordenados por fecha
// descendente, limitados al número indicado.
func (r *AuditRepository) GetAuditLogs(limit string) ([]models.AuditLog, error) {
	query := `SELECT id, usuario_id, accion, descripcion, fecha, ip, user_agent
	          FROM auditoria ORDER BY fecha DESC LIMIT $1`
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var entry models.AuditLog
		var userID sql.NullInt64
		if err := rows.Scan(&entry.ID, &userID, &entry.Accion, &entry.Descripcion, &entry.Fecha, &entry.IP, &entry.UserAgent); err != nil {
			continue
		}
		if userID.Valid {
			uid := int(userID.Int64)
			entry.UsuarioID = &uid
		}
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}
