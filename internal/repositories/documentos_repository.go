package repositories

import (
	"database/sql"
	"time"

	"github.com/andrxsq/SIGMAUDC/internal/models"
)

type DocumentosRepository struct {
	db *sql.DB
}

type DocumentoExistente struct {
	ID     int
	Estado string
}

type DocumentoAuditInfo struct {
	EstudianteCodigo string
	TipoDocumento    string
	PeriodoYear      int
	PeriodoSemestre  int
}

func NewDocumentosRepository(db *sql.DB) *DocumentosRepository {
	return &DocumentosRepository{db: db}
}

func (r *DocumentosRepository) GetPeriodoActivo() (*models.PeriodoAcademico, error) {
	var p models.PeriodoAcademico
	query := `SELECT id, year, semestre, activo, archivado
	          FROM periodo_academico WHERE activo = true AND archivado = false LIMIT 1`
	err := r.db.QueryRow(query).Scan(&p.ID, &p.Year, &p.Semestre, &p.Activo, &p.Archivado)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *DocumentosRepository) GetPlazosByPeriodoPrograma(periodoID, programaID int) (*models.Plazos, error) {
	var plazos models.Plazos
	query := `SELECT id, periodo_id, programa_id, documentos, inscripcion, modificaciones
	          FROM plazos WHERE periodo_id = $1 AND programa_id = $2`
	err := r.db.QueryRow(query, periodoID, programaID).Scan(
		&plazos.ID, &plazos.PeriodoID, &plazos.ProgramaID, &plazos.Documentos, &plazos.Inscripcion, &plazos.Modificaciones,
	)
	if err != nil {
		return nil, err
	}
	return &plazos, nil
}

func (r *DocumentosRepository) GetEstudianteIDByUsuario(usuarioID int) (int, error) {
	var estudianteID int
	err := r.db.QueryRow(`SELECT id FROM estudiante WHERE usuario_id = $1`, usuarioID).Scan(&estudianteID)
	return estudianteID, err
}

func (r *DocumentosRepository) GetJefeIDByUsuario(usuarioID int) (int, error) {
	var jefeID int
	err := r.db.QueryRow(`SELECT id FROM jefe_departamental WHERE usuario_id = $1`, usuarioID).Scan(&jefeID)
	return jefeID, err
}

func (r *DocumentosRepository) ListDocumentosByEstudiantePeriodo(estudianteID, periodoID int) ([]models.DocumentoEstudiante, error) {
	query := `SELECT id, estudiante_id, programa_id, periodo_id, tipo_documento, archivo_url,
	          estado, observacion, revisado_por, fecha_subida, fecha_revision
	          FROM documentos_estudiante
	          WHERE estudiante_id = $1 AND periodo_id = $2
	          ORDER BY fecha_subida DESC`
	rows, err := r.db.Query(query, estudianteID, periodoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var documentos []models.DocumentoEstudiante
	for rows.Next() {
		var doc models.DocumentoEstudiante
		var observacion sql.NullString
		var revisadoPor sql.NullInt64
		var fechaRevision sql.NullTime
		if err := rows.Scan(
			&doc.ID, &doc.EstudianteID, &doc.ProgramaID, &doc.PeriodoID, &doc.TipoDocumento, &doc.ArchivoURL,
			&doc.Estado, &observacion, &revisadoPor, &doc.FechaSubida, &fechaRevision,
		); err != nil {
			continue
		}
		doc.Observacion = models.NullStringJSON{NullString: observacion}
		doc.RevisadoPor = revisadoPor
		doc.FechaRevision = fechaRevision
		documentos = append(documentos, doc)
	}
	return documentos, rows.Err()
}

func (r *DocumentosRepository) GetDocumentoExistente(estudianteID, periodoID int, tipoDocumento string) (*DocumentoExistente, error) {
	var doc DocumentoExistente
	query := `SELECT id, estado FROM documentos_estudiante
	          WHERE estudiante_id = $1 AND periodo_id = $2 AND tipo_documento = $3
	          ORDER BY fecha_subida DESC LIMIT 1`
	err := r.db.QueryRow(query, estudianteID, periodoID, tipoDocumento).Scan(&doc.ID, &doc.Estado)
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *DocumentosRepository) GetProgramaNombre(programaID int) (string, error) {
	var nombre string
	err := r.db.QueryRow(`SELECT nombre FROM programa WHERE id = $1`, programaID).Scan(&nombre)
	return nombre, err
}

func (r *DocumentosRepository) GetUsuarioCodigo(usuarioID int) (string, error) {
	var codigo string
	err := r.db.QueryRow(`SELECT codigo FROM usuario WHERE id = $1`, usuarioID).Scan(&codigo)
	return codigo, err
}

func (r *DocumentosRepository) InsertDocumento(estudianteID, programaID, periodoID int, tipoDocumento, archivoURL string) (int, time.Time, error) {
	query := `INSERT INTO documentos_estudiante
	          (estudiante_id, programa_id, periodo_id, tipo_documento, archivo_url, estado)
	          VALUES ($1, $2, $3, $4, $5, 'pendiente') RETURNING id, fecha_subida`
	var id int
	var fecha time.Time
	err := r.db.QueryRow(query, estudianteID, programaID, periodoID, tipoDocumento, archivoURL).Scan(&id, &fecha)
	return id, fecha, err
}

func (r *DocumentosRepository) GetArchivoURLByDocumentoID(docID int) (string, error) {
	var archivo string
	err := r.db.QueryRow(`SELECT archivo_url FROM documentos_estudiante WHERE id = $1`, docID).Scan(&archivo)
	return archivo, err
}

func (r *DocumentosRepository) UpdateDocumentoRechazado(docID int, archivoURL string) (time.Time, error) {
	query := `UPDATE documentos_estudiante
	          SET archivo_url = $1, estado = 'pendiente', observacion = NULL,
	              revisado_por = NULL, fecha_revision = NULL, fecha_subida = CURRENT_TIMESTAMP
	          WHERE id = $2 RETURNING fecha_subida`
	var fecha time.Time
	err := r.db.QueryRow(query, archivoURL, docID).Scan(&fecha)
	return fecha, err
}

func (r *DocumentosRepository) ListDocumentosByProgramaPeriodo(programaID, periodoID int) ([]models.DocumentoEstudiante, error) {
	query := `SELECT d.id, d.estudiante_id, d.programa_id, d.periodo_id, d.tipo_documento,
	          d.archivo_url, d.estado, d.observacion, d.revisado_por, d.fecha_subida, d.fecha_revision,
	          e.nombre, e.apellido, u.codigo
	          FROM documentos_estudiante d
	          JOIN estudiante e ON d.estudiante_id = e.id
	          JOIN usuario u ON e.usuario_id = u.id
	          WHERE d.programa_id = $1 AND d.periodo_id = $2
	          ORDER BY d.fecha_subida DESC`
	rows, err := r.db.Query(query, programaID, periodoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var documentos []models.DocumentoEstudiante
	for rows.Next() {
		var doc models.DocumentoEstudiante
		var observacion sql.NullString
		var revisadoPor sql.NullInt64
		var fechaRevision sql.NullTime
		if err := rows.Scan(
			&doc.ID, &doc.EstudianteID, &doc.ProgramaID, &doc.PeriodoID, &doc.TipoDocumento,
			&doc.ArchivoURL, &doc.Estado, &observacion, &revisadoPor, &doc.FechaSubida, &fechaRevision,
			&doc.EstudianteNombre, &doc.EstudianteApellido, &doc.EstudianteCodigo,
		); err != nil {
			continue
		}
		doc.Observacion = models.NullStringJSON{NullString: observacion}
		doc.RevisadoPor = revisadoPor
		doc.FechaRevision = fechaRevision
		documentos = append(documentos, doc)
	}
	return documentos, rows.Err()
}

func (r *DocumentosRepository) GetDocumentoProgramaID(docID int) (int, error) {
	var programaID int
	err := r.db.QueryRow(`SELECT programa_id FROM documentos_estudiante WHERE id = $1`, docID).Scan(&programaID)
	return programaID, err
}

func (r *DocumentosRepository) RevisarDocumento(docID, jefeID int, estado string, observacion sql.NullString) (sql.NullTime, error) {
	var fechaRevision sql.NullTime
	query := `UPDATE documentos_estudiante
	          SET estado = $1, observacion = $2, revisado_por = $3, fecha_revision = CURRENT_TIMESTAMP
	          WHERE id = $4 RETURNING fecha_revision`
	err := r.db.QueryRow(query, estado, observacion, jefeID, docID).Scan(&fechaRevision)
	return fechaRevision, err
}

func (r *DocumentosRepository) GetDocumentoAuditInfo(docID int) (*DocumentoAuditInfo, error) {
	var info DocumentoAuditInfo
	query := `SELECT u.codigo, d.tipo_documento, p.year, p.semestre
	          FROM documentos_estudiante d
	          JOIN estudiante e ON d.estudiante_id = e.id
	          JOIN usuario u ON e.usuario_id = u.id
	          JOIN periodo_academico p ON d.periodo_id = p.id
	          WHERE d.id = $1`
	err := r.db.QueryRow(query, docID).Scan(&info.EstudianteCodigo, &info.TipoDocumento, &info.PeriodoYear, &info.PeriodoSemestre)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
