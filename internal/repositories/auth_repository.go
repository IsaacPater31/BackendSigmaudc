package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

type AuthRepository struct {
	db *sql.DB
}

func NewAuthRepository(db *sql.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) GetUsuarioByCodigo(codigo string) (*models.Usuario, error) {
	var usuario models.Usuario
	query := `SELECT id, codigo, email, password_hash, rol, programa_id
	          FROM usuario WHERE codigo = $1`
	err := r.db.QueryRow(query, codigo).Scan(
		&usuario.ID,
		&usuario.Codigo,
		&usuario.Email,
		&usuario.PasswordHash,
		&usuario.Rol,
		&usuario.ProgramaID,
	)
	if err != nil {
		return nil, err
	}
	return &usuario, nil
}

func (r *AuthRepository) GetUsuarioByID(userID int) (*models.Usuario, error) {
	var usuario models.Usuario
	query := `SELECT id, codigo, email, password_hash, rol, programa_id FROM usuario WHERE id = $1`
	err := r.db.QueryRow(query, userID).Scan(
		&usuario.ID,
		&usuario.Codigo,
		&usuario.Email,
		&usuario.PasswordHash,
		&usuario.Rol,
		&usuario.ProgramaID,
	)
	if err != nil {
		return nil, err
	}
	return &usuario, nil
}

func (r *AuthRepository) UpdatePassword(userID int, passwordHash string) error {
	_, err := r.db.Exec(`UPDATE usuario SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
	return err
}

func (r *AuthRepository) GetCurrentUser(userID int) (*models.Usuario, error) {
	var usuario models.Usuario
	var nombre, apellido sql.NullString

	query := `
		SELECT
			u.id, u.codigo, u.email, u.rol, u.programa_id,
			p.nombre as programa_nombre,
			COALESCE(jd.nombre, e.nombre) as nombre,
			COALESCE(jd.apellido, e.apellido) as apellido
		FROM usuario u
		INNER JOIN programa p ON u.programa_id = p.id
		LEFT JOIN jefe_departamental jd ON u.id = jd.usuario_id
		LEFT JOIN estudiante e ON u.id = e.usuario_id
		WHERE u.id = $1
	`

	err := r.db.QueryRow(query, userID).Scan(
		&usuario.ID, &usuario.Codigo, &usuario.Email, &usuario.Rol, &usuario.ProgramaID,
		&usuario.ProgramaNombre, &nombre, &apellido,
	)
	if err != nil {
		return nil, err
	}

	if nombre.Valid {
		usuario.Nombre = nombre.String
	}
	if apellido.Valid {
		usuario.Apellido = apellido.String
	}
	return &usuario, nil
}
