package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) GetEstudianteID(usuarioID int) (int, error) {
	var estudianteID int
	err := r.db.QueryRow(`SELECT id FROM estudiante WHERE usuario_id = $1`, usuarioID).Scan(&estudianteID)
	return estudianteID, err
}

func (r *ProfileRepository) GetJefeID(usuarioID int) (int, error) {
	var jefeID int
	err := r.db.QueryRow(`SELECT id FROM jefe_departamental WHERE usuario_id = $1`, usuarioID).Scan(&jefeID)
	return jefeID, err
}

func (r *ProfileRepository) GetDatosEstudiante(usuarioID int) (*models.EstudianteDatosResponse, sql.NullFloat64, error) {
	query := `
		SELECT e.id, u.codigo, COALESCE(e.nombre, ''), COALESCE(e.apellido, ''), u.email,
		       COALESCE(p.nombre, '') AS programa, e.semestre, e.promedio, e.estado,
		       COALESCE(e.sexo, 'otro'), COALESCE(e.foto_perfil, '')
		FROM usuario u
		JOIN estudiante e ON e.usuario_id = u.id
		LEFT JOIN programa p ON p.id = u.programa_id
		WHERE u.id = $1
	`
	var datos models.EstudianteDatosResponse
	var promedio sql.NullFloat64
	err := r.db.QueryRow(query, usuarioID).Scan(
		&datos.EstudianteID, &datos.Codigo, &datos.Nombre, &datos.Apellido, &datos.Email,
		&datos.Programa, &datos.Semestre, &promedio, &datos.Estado, &datos.Sexo, &datos.FotoPerfil,
	)
	if err != nil {
		return nil, promedio, err
	}
	return &datos, promedio, nil
}

func (r *ProfileRepository) UpdateEstudianteDatos(estudianteID int, req models.UpdateDatosRequest, sexo string) error {
	_, err := r.db.Exec(`UPDATE estudiante SET nombre = $1, apellido = $2, sexo = $3 WHERE id = $4`, req.Nombre, req.Apellido, sexo, estudianteID)
	return err
}

func (r *ProfileRepository) UpdateEstudianteFoto(estudianteID int, photoURL string) error {
	_, err := r.db.Exec(`UPDATE estudiante SET foto_perfil = $1 WHERE id = $2`, photoURL, estudianteID)
	return err
}

func (r *ProfileRepository) GetDatosJefe(usuarioID int) (*models.JefeDatosResponse, error) {
	query := `
		SELECT jd.id, u.codigo, COALESCE(jd.nombre, ''), COALESCE(jd.apellido, ''), u.email,
		       COALESCE(p.nombre, '') AS programa, COALESCE(jd.sexo, 'otro'), COALESCE(jd.foto_perfil, '')
		FROM usuario u
		JOIN jefe_departamental jd ON jd.usuario_id = u.id
		LEFT JOIN programa p ON p.id = u.programa_id
		WHERE u.id = $1
	`
	var datos models.JefeDatosResponse
	err := r.db.QueryRow(query, usuarioID).Scan(
		&datos.JefeID, &datos.Codigo, &datos.Nombre, &datos.Apellido, &datos.Email,
		&datos.Programa, &datos.Sexo, &datos.FotoPerfil,
	)
	if err != nil {
		return nil, err
	}
	return &datos, nil
}

func (r *ProfileRepository) UpdateJefeDatos(jefeID int, req models.UpdateDatosRequest, sexo string) error {
	_, err := r.db.Exec(`UPDATE jefe_departamental SET nombre = $1, apellido = $2, sexo = $3 WHERE id = $4`, req.Nombre, req.Apellido, sexo, jefeID)
	return err
}

func (r *ProfileRepository) UpdateJefeFoto(jefeID int, photoURL string) error {
	_, err := r.db.Exec(`UPDATE jefe_departamental SET foto_perfil = $1 WHERE id = $2`, photoURL, jefeID)
	return err
}
