package repositories

import (
	"database/sql"
	"errors"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/lib/pq"
)

// PlazosRepository encapsula todas las consultas SQL del modulo de periodos/plazos.
type PlazosRepository struct {
	db *sql.DB
}

func NewPlazosRepository(db *sql.DB) *PlazosRepository {
	return &PlazosRepository{db: db}
}

func (r *PlazosRepository) GetPeriodos() ([]models.PeriodoAcademico, error) {
	query := `SELECT id, year, semestre, activo, archivado
	          FROM periodo_academico
	          ORDER BY archivado ASC, activo DESC, year DESC, semestre DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	periodos := make([]models.PeriodoAcademico, 0)
	for rows.Next() {
		var p models.PeriodoAcademico
		if err := rows.Scan(&p.ID, &p.Year, &p.Semestre, &p.Activo, &p.Archivado); err != nil {
			return nil, err
		}
		periodos = append(periodos, p)
	}
	return periodos, rows.Err()
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

func (r *PlazosRepository) ExistsPeriodoByYearAndSemestre(year, semestre int) (bool, error) {
	var exists int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM periodo_academico WHERE year = $1 AND semestre = $2`,
		year, semestre,
	).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func (r *PlazosRepository) CreatePeriodo(year, semestre int) (*models.PeriodoAcademico, error) {
	var periodo models.PeriodoAcademico
	query := `INSERT INTO periodo_academico (year, semestre, activo, archivado)
	          VALUES ($1, $2, false, false)
	          RETURNING id, year, semestre, activo, archivado`
	if err := r.db.QueryRow(query, year, semestre).Scan(
		&periodo.ID, &periodo.Year, &periodo.Semestre, &periodo.Activo, &periodo.Archivado,
	); err != nil {
		return nil, err
	}
	return &periodo, nil
}

func (r *PlazosRepository) GetProgramaIDs() ([]int, error) {
	rows, err := r.db.Query(`SELECT id FROM programa`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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

func (r *PlazosRepository) EnsureDefaultPlazos(periodoID, programaID int) error {
	query := `INSERT INTO plazos (periodo_id, programa_id, documentos, inscripcion, modificaciones)
	          VALUES ($1, $2, false, false, false)
	          ON CONFLICT (periodo_id, programa_id) DO NOTHING`
	_, err := r.db.Exec(query, periodoID, programaID)
	return err
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

func (r *PlazosRepository) DeactivateOtherPeriodos(periodoID int) error {
	_, err := r.db.Exec(`UPDATE periodo_academico SET activo = false WHERE id <> $1`, periodoID)
	return err
}

func (r *PlazosRepository) UpdatePeriodo(periodoID int, activo, archivado bool) (*models.PeriodoAcademico, error) {
	var updated models.PeriodoAcademico
	query := `UPDATE periodo_academico SET activo = $1, archivado = $2
	          WHERE id = $3
	          RETURNING id, year, semestre, activo, archivado`
	err := r.db.QueryRow(query, activo, archivado, periodoID).Scan(
		&updated.ID, &updated.Year, &updated.Semestre, &updated.Activo, &updated.Archivado,
	)
	if err != nil {
		return nil, err
	}
	return &updated, nil
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

func (r *PlazosRepository) GetPeriodosConPlazos(programaID int) ([]models.PeriodoConPlazos, error) {
	query := `
		SELECT
			p.id, p.year, p.semestre, p.activo, p.archivado,
			pl.id, pl.periodo_id, pl.programa_id, pl.documentos, pl.inscripcion, pl.modificaciones
		FROM periodo_academico p
		LEFT JOIN plazos pl ON p.id = pl.periodo_id AND pl.programa_id = $1
		ORDER BY p.archivado ASC, p.activo DESC, p.year DESC, p.semestre DESC
	`
	rows, err := r.db.Query(query, programaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	periodos := make([]models.PeriodoConPlazos, 0)
	for rows.Next() {
		var p models.PeriodoConPlazos
		var plazosID, plazosPeriodoID, plazosProgramaID sql.NullInt64
		var documentos, inscripcion, modificaciones sql.NullBool

		if err := rows.Scan(
			&p.ID, &p.Year, &p.Semestre, &p.Activo, &p.Archivado,
			&plazosID, &plazosPeriodoID, &plazosProgramaID,
			&documentos, &inscripcion, &modificaciones,
		); err != nil {
			return nil, err
		}

		if plazosID.Valid {
			p.Plazos = &models.Plazos{
				ID:             int(plazosID.Int64),
				PeriodoID:      int(plazosPeriodoID.Int64),
				ProgramaID:     int(plazosProgramaID.Int64),
				Documentos:     documentos.Valid && documentos.Bool,
				Inscripcion:    inscripcion.Valid && inscripcion.Bool,
				Modificaciones: modificaciones.Valid && modificaciones.Bool,
			}
		}

		periodos = append(periodos, p)
	}
	return periodos, rows.Err()
}
