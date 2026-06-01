package repositories

import (
	"database/sql"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/lib/pq"
)

type MatriculaRepository struct {
	db *sql.DB
}

func NewMatriculaRepository(db *sql.DB) *MatriculaRepository {
	return &MatriculaRepository{db: db}
}

func (r *MatriculaRepository) GetEstudianteBase(usuarioID int) (int, int, string, error) {
	var estudianteID, semestre int
	var estado string
	err := r.db.QueryRow(`SELECT id, semestre, estado FROM estudiante WHERE usuario_id = $1`, usuarioID).Scan(&estudianteID, &semestre, &estado)
	return estudianteID, semestre, estado, err
}

func (r *MatriculaRepository) GetEstudianteBaseByID(estudianteID int) (int, string, error) {
	var semestre int
	var estado string
	err := r.db.QueryRow(`SELECT semestre, estado FROM estudiante WHERE id = $1`, estudianteID).Scan(&semestre, &estado)
	return semestre, estado, err
}

func (r *MatriculaRepository) GetProgramaIDByEstudianteID(estudianteID int) (int, error) {
	var programaID int
	query := `
		SELECT u.programa_id
		FROM estudiante e
		JOIN usuario u ON u.id = e.usuario_id
		WHERE e.id = $1
	`
	err := r.db.QueryRow(query, estudianteID).Scan(&programaID)
	return programaID, err
}

func (r *MatriculaRepository) GetPeriodoActivo() (*models.PeriodoAcademico, error) {
	var periodo models.PeriodoAcademico
	query := `SELECT id, year, semestre, activo, archivado
	          FROM periodo_academico
	          WHERE activo = true AND archivado = false
	          ORDER BY year DESC, semestre DESC LIMIT 1`
	err := r.db.QueryRow(query).Scan(&periodo.ID, &periodo.Year, &periodo.Semestre, &periodo.Activo, &periodo.Archivado)
	if err != nil {
		return nil, err
	}
	return &periodo, nil
}

func (r *MatriculaRepository) GetPlazos(periodoID, programaID int) (*models.Plazos, error) {
	var plazos models.Plazos
	query := `SELECT id, periodo_id, programa_id, documentos, inscripcion, modificaciones
	          FROM plazos WHERE periodo_id = $1 AND programa_id = $2`
	err := r.db.QueryRow(query, periodoID, programaID).Scan(
		&plazos.ID, &plazos.PeriodoID, &plazos.ProgramaID,
		&plazos.Documentos, &plazos.Inscripcion, &plazos.Modificaciones,
	)
	if err != nil {
		return nil, err
	}
	return &plazos, nil
}

func (r *MatriculaRepository) CountApprovedRequiredDocs(estudianteID, periodoID int) (int, error) {
	var count int
	query := `SELECT COUNT(*)
	          FROM documentos_estudiante
	          WHERE estudiante_id = $1
	            AND periodo_id = $2
	            AND tipo_documento IN ('certificado_eps', 'comprobante_matricula')
	            AND estado = 'aprobado'`
	err := r.db.QueryRow(query, estudianteID, periodoID).Scan(&count)
	return count, err
}

func (r *MatriculaRepository) GetEstudianteIDByUsuario(usuarioID int) (int, error) {
	var id int
	err := r.db.QueryRow(`SELECT id FROM estudiante WHERE usuario_id = $1`, usuarioID).Scan(&id)
	return id, err
}

func (r *MatriculaRepository) GetHorarioActualClases(estudianteID, periodoID int) ([]models.HorarioClase, error) {
	query := `
		SELECT 
			ha.id_asignatura,
			a.codigo,
			a.nombre,
			g.id as grupo_id,
			g.codigo as grupo_codigo,
			COALESCE(g.docente, '') as docente,
			COALESCE(hg.dia, '') as dia,
			COALESCE(hg.hora_inicio::text, '') as hora_inicio,
			COALESCE(hg.hora_fin::text, '') as hora_fin,
			COALESCE(hg.salon, '') as salon
		FROM historial_academico ha
		JOIN grupo g ON ha.grupo_id = g.id
		JOIN asignatura a ON ha.id_asignatura = a.id
		LEFT JOIN horario_grupo hg ON hg.grupo_id = g.id
		JOIN periodo_academico p ON ha.id_periodo = p.id
		WHERE ha.id_estudiante = $1
			AND ha.estado = 'matriculada'
			AND p.activo = true
			AND p.archivado = false
			AND p.id = $2
		ORDER BY 
			CASE hg.dia
				WHEN 'LUNES' THEN 1
				WHEN 'MARTES' THEN 2
				WHEN 'MIERCOLES' THEN 3
				WHEN 'JUEVES' THEN 4
				WHEN 'VIERNES' THEN 5
				WHEN 'SABADO' THEN 6
				ELSE 7
			END,
			hg.hora_inicio
	`
	rows, err := r.db.Query(query, estudianteID, periodoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clases := []models.HorarioClase{}
	for rows.Next() {
		var clase models.HorarioClase
		if err := rows.Scan(
			&clase.AsignaturaID,
			&clase.AsignaturaCodigo,
			&clase.AsignaturaNombre,
			&clase.GrupoID,
			&clase.GrupoCodigo,
			&clase.Docente,
			&clase.Dia,
			&clase.HoraInicio,
			&clase.HoraFin,
			&clase.Salon,
		); err != nil {
			continue
		}
		clases = append(clases, clase)
	}
	return clases, rows.Err()
}

func (r *MatriculaRepository) GetEstudianteIDByCodigo(codigo string) (int, error) {
	var estudianteID int
	query := `
		SELECT e.id
		FROM estudiante e
		JOIN usuario u ON e.usuario_id = u.id
		WHERE u.codigo = $1
	`
	err := r.db.QueryRow(query, codigo).Scan(&estudianteID)
	return estudianteID, err
}

type EstudianteUsuarioInfo struct {
	UsuarioID         int
	UsuarioCodigo     sql.NullString
	UsuarioEmail      sql.NullString
	UsuarioRol        sql.NullString
	UsuarioProgramaID sql.NullInt64
	EstudianteNombre  sql.NullString
}

type MateriaMatriculadaBase struct {
	HistorialID  int
	AsignaturaID int
	Codigo       string
	Nombre       string
	Creditos     int
	GrupoID      int
	GrupoCodigo  string
	Docente      string
}

type HistorialPrevioAsignatura struct {
	Estado   string
	Year     int
	Semestre int
}

type ProgramaInfo struct {
	ID     int
	Nombre string
}

func (r *MatriculaRepository) GetEstudianteUsuarioInfo(estudianteID int) (*EstudianteUsuarioInfo, error) {
	info := &EstudianteUsuarioInfo{}
	query := `
		SELECT u.id, u.codigo, u.email, u.rol, u.programa_id, e.nombre
		FROM estudiante e
		JOIN usuario u ON e.usuario_id = u.id
		WHERE e.id = $1
	`
	err := r.db.QueryRow(query, estudianteID).Scan(
		&info.UsuarioID,
		&info.UsuarioCodigo,
		&info.UsuarioEmail,
		&info.UsuarioRol,
		&info.UsuarioProgramaID,
		&info.EstudianteNombre,
	)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (r *MatriculaRepository) GetMateriasMatriculadasBase(estudianteID, periodoID int) ([]MateriaMatriculadaBase, error) {
	query := `
		SELECT
			ha.id,
			ha.id_asignatura,
			a.codigo,
			a.nombre,
			a.creditos,
			ha.grupo_id,
			g.codigo,
			COALESCE(g.docente, '')
		FROM historial_academico ha
		JOIN asignatura a ON a.id = ha.id_asignatura
		JOIN grupo g ON g.id = ha.grupo_id
		WHERE ha.id_estudiante = $1
			AND ha.id_periodo = $2
			AND ha.estado = 'matriculada'
		ORDER BY a.codigo
	`
	rows, err := r.db.Query(query, estudianteID, periodoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	materias := make([]MateriaMatriculadaBase, 0)
	for rows.Next() {
		var mat MateriaMatriculadaBase
		if err := rows.Scan(
			&mat.HistorialID,
			&mat.AsignaturaID,
			&mat.Codigo,
			&mat.Nombre,
			&mat.Creditos,
			&mat.GrupoID,
			&mat.GrupoCodigo,
			&mat.Docente,
		); err != nil {
			return nil, err
		}
		materias = append(materias, mat)
	}

	return materias, rows.Err()
}

func (r *MatriculaRepository) GetHorariosForGroups(groupIDs []int) (map[int][]models.HorarioDisponible, error) {
	horarios := make(map[int][]models.HorarioDisponible)
	if len(groupIDs) == 0 {
		return horarios, nil
	}
	query := `
		SELECT grupo_id, dia, hora_inicio::text, hora_fin::text, salon
		FROM horario_grupo
		WHERE grupo_id = ANY($1)
	`
	rows, err := r.db.Query(query, pq.Array(groupIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var grupoID int
		var horario models.HorarioDisponible
		if err := rows.Scan(&grupoID, &horario.Dia, &horario.HoraInicio, &horario.HoraFin, &horario.Salon); err != nil {
			return nil, err
		}
		horarios[grupoID] = append(horarios[grupoID], horario)
	}
	return horarios, rows.Err()
}

func (r *MatriculaRepository) GetPeriodoYearSemestreByID(periodoID int) (int, int, error) {
	var year, semestre int
	err := r.db.QueryRow(`SELECT year, semestre FROM periodo_academico WHERE id = $1`, periodoID).Scan(&year, &semestre)
	return year, semestre, err
}

func (r *MatriculaRepository) GetHistorialPrevioAsignatura(estudianteID, asignaturaID, periodoActualID int) ([]HistorialPrevioAsignatura, error) {
	query := `
		SELECT
			ha.estado,
			p.year,
			p.semestre
		FROM historial_academico ha
		JOIN periodo_academico p ON p.id = ha.id_periodo
		WHERE ha.id_estudiante = $1
			AND ha.id_asignatura = $2
			AND ha.id_periodo != $3
		ORDER BY p.year DESC, p.semestre DESC
	`
	rows, err := r.db.Query(query, estudianteID, asignaturaID, periodoActualID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	historial := make([]HistorialPrevioAsignatura, 0)
	for rows.Next() {
		var h HistorialPrevioAsignatura
		if err := rows.Scan(&h.Estado, &h.Year, &h.Semestre); err != nil {
			return nil, err
		}
		historial = append(historial, h)
	}
	return historial, rows.Err()
}

func (r *MatriculaRepository) GetNucleoComunOtrasCarreras(programaID int) ([]models.AsignaturaCompleta, error) {
	query := `
		SELECT DISTINCT
			pa.semestre,
			a.id,
			a.codigo,
			a.nombre,
			a.creditos,
			COALESCE(at.nombre, '') as tipo_nombre,
			a.tiene_laboratorio,
			pa.categoria
		FROM pensum_asignatura pa
		JOIN asignatura a ON pa.asignatura_id = a.id
		LEFT JOIN asignatura_tipo at ON a.tipo_id = at.id
		JOIN pensum p ON pa.pensum_id = p.id
		WHERE pa.categoria = 'nucleo_comun'
			AND p.programa_id != $1
			AND p.activo = true
		ORDER BY a.codigo
	`
	rows, err := r.db.Query(query, programaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	asignaturas := make([]models.AsignaturaCompleta, 0)
	for rows.Next() {
		var asig models.AsignaturaCompleta
		if err := rows.Scan(
			&asig.Semestre,
			&asig.ID,
			&asig.Codigo,
			&asig.Nombre,
			&asig.Creditos,
			&asig.TipoNombre,
			&asig.TieneLaboratorio,
			&asig.Categoria,
		); err != nil {
			return nil, err
		}
		asignaturas = append(asignaturas, asig)
	}
	return asignaturas, rows.Err()
}

func (r *MatriculaRepository) GetProgramasNucleoComun(asignaturaID int) ([]ProgramaInfo, error) {
	query := `
		SELECT DISTINCT p.programa_id, pr.nombre
		FROM pensum_asignatura pa
		JOIN pensum p ON pa.pensum_id = p.id
		JOIN programa pr ON p.programa_id = pr.id
		WHERE pa.asignatura_id = $1
			AND pa.categoria = 'nucleo_comun'
			AND p.activo = true
		ORDER BY pr.nombre
	`
	rows, err := r.db.Query(query, asignaturaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	programas := make([]ProgramaInfo, 0)
	for rows.Next() {
		var p ProgramaInfo
		if err := rows.Scan(&p.ID, &p.Nombre); err != nil {
			return nil, err
		}
		programas = append(programas, p)
	}
	return programas, rows.Err()
}

func (r *MatriculaRepository) GetProgramaPorGrupo(asignaturaID int) (*ProgramaInfo, error) {
	query := `
		SELECT DISTINCT p.programa_id, pr.nombre
		FROM pensum_asignatura pa
		JOIN pensum p ON pa.pensum_id = p.id
		JOIN programa pr ON p.programa_id = pr.id
		WHERE pa.asignatura_id = $1
			AND pa.categoria = 'nucleo_comun'
			AND p.activo = true
		LIMIT 1
	`
	var p ProgramaInfo
	err := r.db.QueryRow(query, asignaturaID).Scan(&p.ID, &p.Nombre)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *MatriculaRepository) CountMateriasMatriculadas(estudianteID, periodoID int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM historial_academico WHERE id_estudiante = $1 AND id_periodo = $2 AND estado = 'matriculada'`
	err := r.db.QueryRow(query, estudianteID, periodoID).Scan(&count)
	return count, err
}

func (r *MatriculaRepository) GetInscritosCredits(estudianteID, periodoID int) (int, error) {
	var creditos sql.NullInt64
	query := `
		SELECT COALESCE(SUM(a.creditos), 0)
		FROM historial_academico ha
		JOIN asignatura a ON a.id = ha.id_asignatura
		WHERE ha.id_estudiante = $1
		  AND ha.id_periodo = $2
		  AND ha.estado = 'matriculada'
	`
	err := r.db.QueryRow(query, estudianteID, periodoID).Scan(&creditos)
	if err != nil {
		return 0, err
	}
	return int(creditos.Int64), nil
}

func (r *MatriculaRepository) GetCreditLimit(pensumID, semestre int) (int, error) {
	var limite sql.NullInt64
	query := `
		SELECT creditos_semestre
		FROM creditos_acumulados_pensum
		WHERE pensum_id = $1 AND semestre = $2
	`
	err := r.db.QueryRow(query, pensumID, semestre).Scan(&limite)
	if err != nil {
		return 0, err
	}
	if limite.Valid {
		return int(limite.Int64), nil
	}
	return 0, sql.ErrNoRows
}

func (r *MatriculaRepository) GetCreditLimitFallback(pensumID, semestre int) (int, error) {
	var total sql.NullInt64
	query := `
		SELECT COALESCE(SUM(a.creditos), 0)
		FROM pensum_asignatura pa
		JOIN asignatura a ON a.id = pa.asignatura_id
		WHERE pa.pensum_id = $1 AND pa.semestre = $2
	`
	err := r.db.QueryRow(query, pensumID, semestre).Scan(&total)
	if err != nil {
		return 0, err
	}
	return int(total.Int64), nil
}
