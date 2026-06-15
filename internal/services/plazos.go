package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

var (
	ErrPeriodoNotFound  = errors.New("periodo not found")
	ErrPeriodoArchivado = errors.New("periodo archivado")
	ErrPeriodoInactivo  = errors.New("periodo inactivo")
)

type AuditMetadata struct {
	UsuarioID  int
	IP         string
	UserAgent  string
	ProgramaID int
}

// PlazosService concentra reglas de negocio de plazos del periodo activo.
type PlazosService struct {
	repo      *repositories.PlazosRepository
	auditoria *AuditoriaService
}

func NewPlazosService(repo *repositories.PlazosRepository, auditoria *AuditoriaService) *PlazosService {
	return &PlazosService{
		repo:      repo,
		auditoria: auditoria,
	}
}

func (s *PlazosService) GetPeriodoActivo() (*models.PeriodoAcademico, error) {
	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return periodo, err
}

func (s *PlazosService) GetActivePeriodoPlazos(programaID int) (*models.ActivePlazosResponse, error) {
	periodo, err := s.GetPeriodoActivo()
	if err != nil {
		return nil, err
	}
	if periodo == nil {
		return &models.ActivePlazosResponse{Periodo: nil, Plazos: nil}, nil
	}

	plazos, err := s.repo.GetOrCreatePlazos(periodo.ID, programaID)
	if err != nil {
		return nil, err
	}

	return &models.ActivePlazosResponse{
		Periodo: periodo,
		Plazos:  plazos,
	}, nil
}

func (s *PlazosService) UpdatePlazos(periodoID, programaID int, req models.UpdatePlazosRequest, audit AuditMetadata) (*models.Plazos, error) {
	periodo, err := s.repo.GetPeriodoByID(periodoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPeriodoNotFound
	}
	if err != nil {
		return nil, err
	}
	if periodo.Archivado {
		return nil, ErrPeriodoArchivado
	}
	if !periodo.Activo {
		return nil, ErrPeriodoInactivo
	}

	plazos, err := s.repo.GetOrCreatePlazos(periodoID, programaID)
	if err != nil {
		return nil, err
	}

	documentos := plazos.Documentos
	inscripcion := plazos.Inscripcion
	modificaciones := plazos.Modificaciones

	if req.Documentos != nil {
		documentos = *req.Documentos
	}
	if req.Inscripcion != nil {
		inscripcion = *req.Inscripcion
	}
	if req.Modificaciones != nil {
		modificaciones = *req.Modificaciones
	}

	updated, err := s.repo.UpdatePlazos(periodoID, programaID, documentos, inscripcion, modificaciones)
	if err != nil {
		return nil, err
	}

	cambios := collectCambios(req, plazos)
	if len(cambios) > 0 {
		year, semestre, programaNombre := s.buildAuditInfo(periodoID, audit.ProgramaID)
		descripcion := fmt.Sprintf(
			"Actualización de plazos - Periodo: %d-%d, Programa: %s, Cambios: %s",
			year, semestre, programaNombre, strings.Join(cambios, ", "),
		)
		s.auditoria.Registrar(audit.UsuarioID, "actualizacion_plazos", descripcion, audit.IP, audit.UserAgent)
	}

	return updated, nil
}

func (s *PlazosService) buildAuditInfo(periodoID, programaID int) (int, int, string) {
	year, semestre, programa, err := s.repo.GetPeriodoProgramaInfo(periodoID, programaID)
	if err == nil {
		return year, semestre, programa
	}

	year, semestre, err = s.repo.GetPeriodoYearSemestre(periodoID)
	if err != nil {
		return 0, 0, fmt.Sprintf("Programa ID: %d", programaID)
	}
	return year, semestre, fmt.Sprintf("Programa ID: %d", programaID)
}

func collectCambios(req models.UpdatePlazosRequest, old *models.Plazos) []string {
	cambios := make([]string, 0, 3)
	if req.Documentos != nil && *req.Documentos != old.Documentos {
		cambios = append(cambios, fmt.Sprintf("documentos: %s", estadoString(*req.Documentos)))
	}
	if req.Inscripcion != nil && *req.Inscripcion != old.Inscripcion {
		cambios = append(cambios, fmt.Sprintf("inscripcion: %s", estadoString(*req.Inscripcion)))
	}
	if req.Modificaciones != nil && *req.Modificaciones != old.Modificaciones {
		cambios = append(cambios, fmt.Sprintf("modificaciones: %s", estadoString(*req.Modificaciones)))
	}
	return cambios
}

func estadoString(v bool) string {
	if v {
		return "activado"
	}
	return "desactivado"
}
