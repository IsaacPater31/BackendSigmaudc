package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/lib/pq"
)

type PensumRepository struct {
	db *sql.DB
}

type HistorialRecord struct {
	AsignaturaID int
	Estado       string
	Nota         sql.NullFloat64
	GrupoID      sql.NullInt64
	PeriodoID    int
	Year         int
	Semestre     int
	Ordinal      int
}

func NewPensumRepository(db *sql.DB) *PensumRepository {
	return &PensumRepository{db: db}
}

func (r *PensumRepository) GetEstudianteID(usuarioID int) (int, error) {
	var id int
	err := r.db.QueryRow(`SELECT id FROM estudiante WHERE usuario_id = $1`, usuarioID).Scan(&id)
	return id, err
}

func (r *PensumRepository) GetPensumInfo(estudianteID int) (int, string, string, error) {
	var pensumID int
	var pensumNombre, programaNombre string
	query := `SELECT p.id, p.nombre, pr.nombre
	          FROM estudiante_pensum ep
	          JOIN pensum p ON ep.pensum_id = p.id
	          JOIN programa pr ON p.programa_id = pr.id
	          WHERE ep.estudiante_id = $1`
	err := r.db.QueryRow(query, estudianteID).Scan(&pensumID, &pensumNombre, &programaNombre)
	return pensumID, pensumNombre, programaNombre, err
}

func (r *PensumRepository) GetActivePeriodo() (*models.PeriodoAcademico, error) {
	var periodo models.PeriodoAcademico
	query := `SELECT id, year, semestre FROM periodo_academico WHERE activo = true AND archivado = false ORDER BY year DESC, semestre DESC LIMIT 1`
	err := r.db.QueryRow(query).Scan(&periodo.ID, &periodo.Year, &periodo.Semestre)
	if err != nil {
		return nil, err
	}
	return &periodo, nil
}

func (r *PensumRepository) GetAsignaturas(pensumID int) ([]models.AsignaturaCompleta, error) {
	query := `SELECT pa.semestre, a.id, a.codigo, a.nombre, a.creditos, COALESCE(at.nombre, ''), a.tiene_laboratorio, pa.categoria
	          FROM pensum_asignatura pa
	          JOIN asignatura a ON pa.asignatura_id = a.id
	          LEFT JOIN asignatura_tipo at ON a.tipo_id = at.id
	          WHERE pa.pensum_id = $1
	          ORDER BY pa.semestre, a.codigo`
	rows, err := r.db.Query(query, pensumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var asignaturas []models.AsignaturaCompleta
	for rows.Next() {
		var asig models.AsignaturaCompleta
		if err := rows.Scan(&asig.Semestre, &asig.ID, &asig.Codigo, &asig.Nombre, &asig.Creditos, &asig.TipoNombre, &asig.TieneLaboratorio, &asig.Categoria); err != nil {
			return nil, err
		}
		asignaturas = append(asignaturas, asig)
	}
	return asignaturas, rows.Err()
}

func (r *PensumRepository) BuildHistorialMap(estudianteID int) (map[int][]HistorialRecord, error) {
	query := `SELECT ha.id_asignatura, ha.estado, ha.nota, ha.grupo_id, ha.id_periodo, p.year, p.semestre
	          FROM historial_academico ha
	          JOIN periodo_academico p ON ha.id_periodo = p.id
	          WHERE ha.id_estudiante = $1
	          ORDER BY p.year, p.semestre, ha.id`
	rows, err := r.db.Query(query, estudianteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hist := make(map[int][]HistorialRecord)
	for rows.Next() {
		var rec HistorialRecord
		if err := rows.Scan(&rec.AsignaturaID, &rec.Estado, &rec.Nota, &rec.GrupoID, &rec.PeriodoID, &rec.Year, &rec.Semestre); err != nil {
			return nil, err
		}
		rec.Ordinal = rec.Year*2 + (rec.Semestre - 1)
		hist[rec.AsignaturaID] = append(hist[rec.AsignaturaID], rec)
	}
	return hist, rows.Err()
}

func (r *PensumRepository) BuildPrereqMap(pensumID int) (map[int][]models.Prerequisito, error) {
	query := `SELECT pr.asignatura_id, pr.prerequisito_id, a.codigo, a.nombre, COALESCE(pa.semestre, 0), pr.tipo
	          FROM pensum_prerequisito pr
	          JOIN asignatura a ON pr.prerequisito_id = a.id
	          LEFT JOIN pensum_asignatura pa ON pa.asignatura_id = a.id AND pa.pensum_id = $1
	          WHERE pr.pensum_id = $1
	          ORDER BY a.codigo`
	rows, err := r.db.Query(query, pensumID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	prereqMap := make(map[int][]models.Prerequisito)
	for rows.Next() {
		var prereq models.Prerequisito
		if err := rows.Scan(&prereq.AsignaturaID, &prereq.PrerequisitoID, &prereq.Codigo, &prereq.Nombre, &prereq.Semestre, &prereq.Tipo); err != nil {
			return nil, err
		}
		prereqMap[prereq.AsignaturaID] = append(prereqMap[prereq.AsignaturaID], prereq)
	}
	return prereqMap, rows.Err()
}

func (r *PensumRepository) ListPensums() ([]models.PensumItem, error) {
	rows, err := r.db.Query(`SELECT id, nombre FROM pensum ORDER BY nombre`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.PensumItem
	for rows.Next() {
		var p models.PensumItem
		if err := rows.Scan(&p.ID, &p.Nombre); err == nil {
			list = append(list, p)
		}
	}
	return list, rows.Err()
}

func (r *PensumRepository) GetGruposPensum(pensumID, periodoID int) ([]models.GrupoPensum, []int, error) {
	query := `SELECT g.id, g.codigo, a.id, a.codigo, a.nombre, COALESCE(pa.semestre, 0), a.creditos, COALESCE(g.docente, ''), g.cupo_disponible, g.cupo_max
	          FROM grupo g
	          JOIN asignatura a ON g.asignatura_id = a.id
	          JOIN pensum_asignatura pa ON pa.asignatura_id = a.id AND pa.pensum_id = $1
	          WHERE g.periodo_id = $2
	          ORDER BY pa.semestre, a.codigo, g.codigo`
	rows, err := r.db.Query(query, pensumID, periodoID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var grupos []models.GrupoPensum
	var ids []int
	for rows.Next() {
		var g models.GrupoPensum
		if err := rows.Scan(&g.ID, &g.Codigo, &g.AsignaturaID, &g.AsignaturaCodigo, &g.AsignaturaNombre, &g.Semestre, &g.Creditos, &g.Docente, &g.CupoDisponible, &g.CupoMax); err != nil {
			continue
		}
		grupos = append(grupos, g)
		ids = append(ids, g.ID)
	}
	return grupos, ids, rows.Err()
}

func (r *PensumRepository) FetchHorariosForGroups(groupIDs []int) (map[int][]models.HorarioDisponible, error) {
	horarios := make(map[int][]models.HorarioDisponible)
	if len(groupIDs) == 0 {
		return horarios, nil
	}
	query := `SELECT grupo_id, dia, hora_inicio::text, hora_fin::text, COALESCE(salon, '')
	          FROM horario_grupo WHERE grupo_id = ANY($1)`
	rows, err := r.db.Query(query, pq.Array(groupIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var grupoID int
		var h models.HorarioDisponible
		if err := rows.Scan(&grupoID, &h.Dia, &h.HoraInicio, &h.HoraFin, &h.Salon); err != nil {
			return nil, err
		}
		horarios[grupoID] = append(horarios[grupoID], h)
	}
	return horarios, rows.Err()
}
