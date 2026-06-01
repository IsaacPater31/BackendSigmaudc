package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

type AuditRepository struct {
	db *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

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
