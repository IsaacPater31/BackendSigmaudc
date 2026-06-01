package services

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

var (
	ErrDocumentoPlazo            = errors.New("plazo documentos invalido")
	ErrDocumentoTipoInvalido     = errors.New("tipo documento invalido")
	ErrDocumentoArchivoInvalido  = errors.New("archivo invalido")
	ErrDocumentoNoEncontrado     = errors.New("documento no encontrado")
	ErrDocumentoForbidden        = errors.New("documento forbidden")
	ErrDocumentoReviewInvalida   = errors.New("review invalida")
	ErrEstudianteNoEncontradoDoc = errors.New("estudiante no encontrado")
	ErrJefeNoEncontradoDoc       = errors.New("jefe no encontrado")
)

type DocumentosService struct {
	repo            *repositories.DocumentosRepository
	auditoria       *AuditoriaService
	uploadDirectory string
}

func NewDocumentosService(repo *repositories.DocumentosRepository, auditoria *AuditoriaService, uploadDirectory string) *DocumentosService {
	if uploadDirectory == "" {
		uploadDirectory = "./uploads"
	}
	_ = os.MkdirAll(uploadDirectory, 0755)
	return &DocumentosService{repo: repo, auditoria: auditoria, uploadDirectory: uploadDirectory}
}

func (s *DocumentosService) verificarPlazosDocumentos(programaID int) (*models.Plazos, *models.PeriodoAcademico, error) {
	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, errors.New("no hay periodo académico activo")
	}
	if err != nil {
		return nil, nil, err
	}
	plazos, err := s.repo.GetPlazosByPeriodoPrograma(periodo.ID, programaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, errors.New("no hay plazos configurados para este programa en el periodo activo")
	}
	if err != nil {
		return nil, nil, err
	}
	if !plazos.Documentos {
		return nil, nil, errors.New("el plazo de documentos no está activo para este programa")
	}
	return plazos, periodo, nil
}

func (s *DocumentosService) GetDocumentosEstudiante(usuarioID, programaID int) (*models.DocumentosEstudianteResponse, error) {
	estudianteID, err := s.repo.GetEstudianteIDByUsuario(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEstudianteNoEncontradoDoc
	}
	if err != nil {
		return nil, err
	}

	var plazoMensaje string
	plazos, periodo, err := s.verificarPlazosDocumentos(programaID)
	if err != nil {
		plazos = nil
		periodo = nil
		plazoMensaje = err.Error()
	}

	var documentos []models.DocumentoEstudiante
	if periodo != nil {
		documentos, err = s.repo.ListDocumentosByEstudiantePeriodo(estudianteID, periodo.ID)
		if err != nil {
			return nil, err
		}
	}

	documentosAprobados := true
	if len(documentos) < constants.DocsRequeridosInscripcion {
		documentosAprobados = false
	} else {
		for _, doc := range documentos {
			if doc.Estado != constants.EstadoDocAprobado {
				documentosAprobados = false
				break
			}
		}
	}

	return &models.DocumentosEstudianteResponse{
		Documentos:          documentos,
		PeriodoActivo:       periodo,
		PlazoDocumentos:     plazos != nil && plazos.Documentos,
		PuedeSubir:          plazos != nil && plazos.Documentos && periodo != nil,
		DocumentosAprobados: documentosAprobados,
		PlazoMensaje:        plazoMensaje,
	}, nil
}

func (s *DocumentosService) SubirDocumento(usuarioID, programaID int, tipoDocumento string, file multipart.File, header *multipart.FileHeader, ip, userAgent string) (map[string]interface{}, error) {
	_, periodo, err := s.verificarPlazosDocumentos(programaID)
	if err != nil {
		return nil, ErrDocumentoPlazo
	}
	estudianteID, err := s.repo.GetEstudianteIDByUsuario(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEstudianteNoEncontradoDoc
	}
	if err != nil {
		return nil, err
	}
	if tipoDocumento != constants.TipoCertificadoEPS && tipoDocumento != constants.TipoComprobanteMatricula {
		return nil, ErrDocumentoTipoInvalido
	}
	if header.Size > constants.MaxDocumentoBytes {
		return nil, ErrDocumentoArchivoInvalido
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowedDocExt(ext, constants.ExtensionesDocumento) {
		return nil, ErrDocumentoArchivoInvalido
	}

	docExistente, err := s.repo.GetDocumentoExistente(estudianteID, periodo.ID, tipoDocumento)
	if err == nil {
		if docExistente.Estado == constants.EstadoDocPendiente || docExistente.Estado == constants.EstadoDocAprobado {
			return nil, ErrDocumentoReviewInvalida
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	programaNombre, err := s.repo.GetProgramaNombre(programaID)
	if err != nil {
		programaNombre = fmt.Sprintf("programa_%d", programaID)
	}
	estudianteCodigo, err := s.repo.GetUsuarioCodigo(usuarioID)
	if err != nil {
		estudianteCodigo = fmt.Sprintf("estudiante_%d", estudianteID)
	}

	filenameWithoutExt := strings.TrimSuffix(header.Filename, ext)
	periodoFolder := fmt.Sprintf("%d-%d", periodo.Year, periodo.Semestre)
	programaFolder := fmt.Sprintf("%d_%s", programaID, strings.ReplaceAll(strings.ToLower(programaNombre), " ", "_"))
	estudianteFolder := fmt.Sprintf("%d_%s", estudianteID, estudianteCodigo)
	uploadPath := filepath.Join(s.uploadDirectory, periodoFolder, programaFolder, estudianteFolder)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("%d_%d_%s_%s%s", estudianteID, time.Now().Unix(), tipoDocumento, filenameWithoutExt, ext)
	filePath := filepath.Join(uploadPath, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		_ = os.Remove(filePath)
		return nil, err
	}
	archivoURL := fmt.Sprintf("/uploads/%s/%s/%s/%s", periodoFolder, programaFolder, estudianteFolder, filename)

	if docExistente == nil || docExistente.ID == 0 {
		docID, fechaSubida, err := s.repo.InsertDocumento(estudianteID, programaID, periodo.ID, tipoDocumento, archivoURL)
		if err != nil {
			_ = os.Remove(filePath)
			return nil, err
		}
		s.auditoria.Registrar(usuarioID, "subida_documento", fmt.Sprintf("Documento subido: %s, Periodo: %d-%d", tipoDocumento, periodo.Year, periodo.Semestre), ip, userAgent)
		return map[string]interface{}{
			"id":             docID,
			"tipo_documento": tipoDocumento,
			"estado":         constants.EstadoDocPendiente,
			"fecha_subida":   fechaSubida,
			"message":        "Documento subido exitosamente",
		}, nil
	}

	archivoAnterior, _ := s.repo.GetArchivoURLByDocumentoID(docExistente.ID)
	if strings.HasPrefix(archivoAnterior, "/uploads/") {
		_ = os.Remove(filepath.Join(s.uploadDirectory, strings.TrimPrefix(archivoAnterior, "/uploads/")))
	}
	fechaSubida, err := s.repo.UpdateDocumentoRechazado(docExistente.ID, archivoURL)
	if err != nil {
		_ = os.Remove(filePath)
		return nil, err
	}
	s.auditoria.Registrar(usuarioID, "resubida_documento", fmt.Sprintf("Documento resubido: %s, Periodo: %d-%d (anteriormente rechazado)", tipoDocumento, periodo.Year, periodo.Semestre), ip, userAgent)
	return map[string]interface{}{
		"id":             docExistente.ID,
		"tipo_documento": tipoDocumento,
		"estado":         constants.EstadoDocPendiente,
		"fecha_subida":   fechaSubida,
		"message":        "Documento resubido exitosamente",
	}, nil
}

func (s *DocumentosService) GetDocumentosPorPrograma(programaID int) ([]models.DocumentoEstudiante, error) {
	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return []models.DocumentoEstudiante{}, nil
	}
	if err != nil {
		return nil, err
	}
	return s.repo.ListDocumentosByProgramaPeriodo(programaID, periodo.ID)
}

func (s *DocumentosService) RevisarDocumento(usuarioID, programaID, docID int, req models.RevisarDocumentoRequest, ip, userAgent string) (map[string]interface{}, error) {
	jefeID, err := s.repo.GetJefeIDByUsuario(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJefeNoEncontradoDoc
	}
	if err != nil {
		return nil, err
	}
	if req.Estado != constants.EstadoDocAprobado && req.Estado != constants.EstadoDocRechazado {
		return nil, ErrDocumentoReviewInvalida
	}
	if req.Estado == constants.EstadoDocRechazado && strings.TrimSpace(req.Observacion) == "" {
		return nil, ErrDocumentoReviewInvalida
	}
	docProgramaID, err := s.repo.GetDocumentoProgramaID(docID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrDocumentoNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if docProgramaID != programaID {
		return nil, ErrDocumentoForbidden
	}

	observacionVal := sql.NullString{Valid: false}
	if req.Estado == constants.EstadoDocRechazado && strings.TrimSpace(req.Observacion) != "" {
		observacionVal = sql.NullString{String: req.Observacion, Valid: true}
	}
	fechaRevision, err := s.repo.RevisarDocumento(docID, jefeID, req.Estado, observacionVal)
	if err != nil {
		return nil, err
	}

	info, err := s.repo.GetDocumentoAuditInfo(docID)
	if err == nil {
		accion := "revision_documento_aprobado"
		if req.Estado == constants.EstadoDocRechazado {
			accion = "revision_documento_rechazado"
		}
		descripcion := fmt.Sprintf("Documento %s: %s - Estudiante: %s, Periodo: %d-%d", req.Estado, info.TipoDocumento, info.EstudianteCodigo, info.PeriodoYear, info.PeriodoSemestre)
		if req.Estado == constants.EstadoDocRechazado && strings.TrimSpace(req.Observacion) != "" {
			descripcion += fmt.Sprintf(", Observación: %s", req.Observacion)
		}
		s.auditoria.Registrar(usuarioID, accion, descripcion, ip, userAgent)
	}

	return map[string]interface{}{
		"id":             docID,
		"estado":         req.Estado,
		"observacion":    req.Observacion,
		"fecha_revision": fechaRevision,
		"message":        "Documento revisado exitosamente",
	}, nil
}

func isAllowedDocExt(ext string, allowed []string) bool {
	for _, v := range allowed {
		if ext == v {
			return true
		}
	}
	return false
}
