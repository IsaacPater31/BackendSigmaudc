package repositories

import (
	"database/sql"
	"errors"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/lib/pq"
)

// PlazosRepository encapsula consultas SQL de plazos del periodo activo.
type PlazosRepository struct {
	db *sql.DB
}

func NewPlazosRepository(db *sql.DB) *PlazosRepository {
	return &PlazosRepository{db: db}
}

func (r *PlazosRepository) GetPeriodoActivo() (*models.PeriodoAcademico, error) {
	var periodo models.PeriodoAcademico
	query := `SELECT id, year, semestre, activo, archivado
	          FROM periodo_academico
	          WHERE activo = true AND archivado = false LIMIT 1`
	err := r.db.QueryRow(query).Scan(
		&periodo.ID, &periodo.Year, &periodo.Semestre, &periodo.Activo, &periodo.Archivado,
	)
	if err != nil {
		return nil, err
	}
	return &periodo, nil
}

func (r *PlazosRepository) GetPeriodoByID(periodoID int) (*models.PeriodoAcademico, error) {
	var periodo models.PeriodoAcademico
	query := `SELECT id, year, semestre, activo, archivado FROM periodo_academico WHERE id = $1`
	err := r.db.QueryRow(query, periodoID).Scan(
		&periodo.ID, &periodo.Year, &periodo.Semestre, &periodo.Activo, &periodo.Archivado,
	)
	if err != nil {
		return nil, err
	}
	return &periodo, nil
}

func (r *PlazosRepository) CreateDefaultPlazos(periodoID, programaID int) (*models.Plazos, error) {
	var created models.Plazos
	query := `INSERT INTO plazos (periodo_id, programa_id, documentos, inscripcion, modificaciones)
	          VALUES ($1, $2, false, false, false)
	          RETURNING id, periodo_id, programa_id, documentos, inscripcion, modificaciones`
	err := r.db.QueryRow(query, periodoID, programaID).Scan(
		&created.ID,
		&created.PeriodoID,
		&created.ProgramaID,
		&created.Documentos,
		&created.Inscripcion,
		&created.Modificaciones,
	)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (r *PlazosRepository) GetPlazos(periodoID, programaID int) (*models.Plazos, error) {
	var plazos models.Plazos
	query := `SELECT id, periodo_id, programa_id, documentos, inscripcion, modificaciones
	          FROM plazos WHERE periodo_id = $1 AND programa_id = $2`
	err := r.db.QueryRow(query, periodoID, programaID).Scan(
		&plazos.ID,
		&plazos.PeriodoID,
		&plazos.ProgramaID,
		&plazos.Documentos,
		&plazos.Inscripcion,
		&plazos.Modificaciones,
	)
	if err != nil {
		return nil, err
	}
	return &plazos, nil
}

func (r *PlazosRepository) GetOrCreatePlazos(periodoID, programaID int) (*models.Plazos, error) {
	plazos, err := r.GetPlazos(periodoID, programaID)
	if err == nil {
		return plazos, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	created, err := r.CreateDefaultPlazos(periodoID, programaID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r.GetOrCreatePlazos(periodoID, programaID)
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return r.GetOrCreatePlazos(periodoID, programaID)
		}
		return nil, err
	}
	return created, nil
}

func (r *PlazosRepository) UpdatePlazos(periodoID, programaID int, documentos, inscripcion, modificaciones bool) (*models.Plazos, error) {
	var updated models.Plazos
	query := `UPDATE plazos SET documentos = $1, inscripcion = $2, modificaciones = $3
	          WHERE periodo_id = $4 AND programa_id = $5
	          RETURNING id, periodo_id, programa_id, documentos, inscripcion, modificaciones`
	err := r.db.QueryRow(query, documentos, inscripcion, modificaciones, periodoID, programaID).Scan(
		&updated.ID, &updated.PeriodoID, &updated.ProgramaID,
		&updated.Documentos, &updated.Inscripcion, &updated.Modificaciones,
	)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (r *PlazosRepository) GetPeriodoProgramaInfo(periodoID, programaID int) (int, int, string, error) {
	var year, semestre int
	var programa string
	query := `SELECT p.year, p.semestre, pr.nombre
	          FROM periodo_academico p
	          CROSS JOIN programa pr
	          WHERE pr.id = $1 AND p.id = $2`
	err := r.db.QueryRow(query, programaID, periodoID).Scan(&year, &semestre, &programa)
	if err != nil {
		return 0, 0, "", err
	}
	return year, semestre, programa, nil
}

func (r *PlazosRepository) GetPeriodoYearSemestre(periodoID int) (int, int, error) {
	var year, semestre int
	err := r.db.QueryRow(`SELECT year, semestre FROM periodo_academico WHERE id = $1`, periodoID).Scan(&year, &semestre)
	if err != nil {
		return 0, 0, err
	}
	return year, semestre, nil
}
