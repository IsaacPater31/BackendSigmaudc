package services

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

type MatriculaContext struct {
	EstudianteID   int
	Semestre       int
	Estado         string
	PensumID       int
	PensumNombre   string
	ProgramaID     int
	ProgramaNombre string
	Periodo        *models.PeriodoAcademico
	Plazos         models.Plazos
}

type MatriculaService struct {
	repo       *repositories.MatriculaRepository
	pensumRepo *repositories.PensumRepository
}

type ModificacionesCoreData struct {
	Periodo              *models.PeriodoAcademico
	MateriasMatriculadas []models.MateriaMatriculada
	CreditosInscritos    int
	CreditosMax          int
	CreditosDisponibles  int
	EstadoEstudiante     string
}

var (
	ErrMatriculaMissingStudentParam = errors.New("missing student param")
	ErrMatriculaInvalidStudentID    = errors.New("invalid student id")
	ErrMatriculaStudentNotFound     = errors.New("student not found")
)

func NewMatriculaService(repo *repositories.MatriculaRepository, pensumRepo *repositories.PensumRepository) *MatriculaService {
	return &MatriculaService{repo: repo, pensumRepo: pensumRepo}
}

func (s *MatriculaService) PrepareInscripcionContext(claims *models.JWTClaims) (*MatriculaContext, string, error) {
	ctx, razon, err := s.prepareBaseContext(claims)
	if err != nil || razon != "" {
		return ctx, razon, err
	}
	if !ctx.Plazos.Inscripcion {
		return nil, "El plazo de inscripción no está activo para tu programa en este periodo.", nil
	}
	docsAprobados, err := s.repo.CountApprovedRequiredDocs(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, "", err
	}
	if docsAprobados < constants.DocsRequeridosInscripcion {
		return nil, "No puedes inscribir asignaturas porque tus documentos requeridos (certificado EPS y comprobante de matrícula) aún no han sido aprobados. Por favor, sube los documentos y espera su aprobación.", nil
	}
	return ctx, "", nil
}

func (s *MatriculaService) PrepareModificacionesContext(claims *models.JWTClaims) (*MatriculaContext, string, error) {
	ctx, razon, err := s.prepareBaseContext(claims)
	if err != nil || razon != "" {
		return ctx, razon, err
	}
	if !ctx.Plazos.Modificaciones {
		return nil, "El plazo de modificaciones no está activo para tu programa en este periodo.", nil
	}
	return ctx, "", nil
}

func (s *MatriculaService) PrepareModificacionesContextForEstudiante(estudianteID int) (*MatriculaContext, string, error) {
	semestre, estado, err := s.repo.GetEstudianteBaseByID(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "Estudiante no encontrado", nil
	}
	if err != nil {
		return nil, "", err
	}

	pensumID, pensumNombre, programaNombre, err := s.pensumRepo.GetPensumInfo(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No tienes un pensum asignado. Contacta a coordinación para asignarte uno.", nil
	}
	if err != nil {
		return nil, "", err
	}

	programaID, err := s.repo.GetProgramaIDByEstudianteID(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "Estudiante no encontrado", nil
	}
	if err != nil {
		return nil, "", err
	}

	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No hay un periodo académico activo.", nil
	}
	if err != nil {
		return nil, "", err
	}

	plazos, err := s.repo.GetPlazos(periodo.ID, programaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No hay plazos configurados para el programa del estudiante en el periodo activo.", nil
	}
	if err != nil {
		return nil, "", err
	}

	if !plazos.Modificaciones {
		return nil, "El plazo de modificaciones no está activo para el programa de este estudiante en este periodo.", nil
	}

	return &MatriculaContext{
		EstudianteID:   estudianteID,
		Semestre:       semestre,
		Estado:         estado,
		PensumID:       pensumID,
		PensumNombre:   pensumNombre,
		ProgramaID:     programaID,
		ProgramaNombre: programaNombre,
		Periodo:        periodo,
		Plazos:         *plazos,
	}, "", nil
}

func (s *MatriculaService) GetHorarioActual(usuarioID int) (map[string]interface{}, int, error) {
	estudianteID, err := s.repo.GetEstudianteIDByUsuario(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 404, nil
	}
	if err != nil {
		return nil, 500, err
	}

	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]interface{}{
			"periodo": nil,
			"clases":  []interface{}{},
		}, 200, nil
	}
	if err != nil {
		return nil, 500, err
	}

	clases, err := s.repo.GetHorarioActualClases(estudianteID, periodo.ID)
	if err != nil {
		return nil, 500, err
	}

	response := map[string]interface{}{
		"periodo": map[string]interface{}{
			"id":       periodo.ID,
			"year":     periodo.Year,
			"semestre": periodo.Semestre,
		},
		"clases": clases,
	}
	return response, 200, nil
}

func (s *MatriculaService) GetStudentMatricula(codigo, idStr string) (map[string]interface{}, error) {
	var estudianteID int
	var err error
	if idStr != "" {
		estudianteID, err = strconv.Atoi(idStr)
		if err != nil || estudianteID <= 0 {
			return nil, ErrMatriculaInvalidStudentID
		}
	} else if codigo != "" {
		estudianteID, err = s.repo.GetEstudianteIDByCodigo(codigo)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrMatriculaStudentNotFound
		}
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrMatriculaMissingStudentParam
	}

	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]interface{}{"periodo": nil, "clases": []interface{}{}}, nil
	}
	if err != nil {
		return nil, err
	}

	clases, err := s.repo.GetHorarioActualClases(estudianteID, periodo.ID)
	if err != nil {
		return nil, err
	}
	info, err := s.repo.GetEstudianteUsuarioInfo(estudianteID)
	if err != nil {
		return nil, err
	}

	estudianteMap := map[string]interface{}{
		"id":     estudianteID,
		"nombre": nil,
		"usuario": map[string]interface{}{
			"id":          info.UsuarioID,
			"codigo":      nil,
			"email":       nil,
			"rol":         nil,
			"programa_id": nil,
		},
	}
	if info.EstudianteNombre.Valid {
		estudianteMap["nombre"] = info.EstudianteNombre.String
	}
	usuarioSub := estudianteMap["usuario"].(map[string]interface{})
	if info.UsuarioCodigo.Valid {
		usuarioSub["codigo"] = info.UsuarioCodigo.String
	}
	if info.UsuarioEmail.Valid {
		usuarioSub["email"] = info.UsuarioEmail.String
	}
	if info.UsuarioRol.Valid {
		usuarioSub["rol"] = info.UsuarioRol.String
	}
	if info.UsuarioProgramaID.Valid {
		usuarioSub["programa_id"] = int(info.UsuarioProgramaID.Int64)
	}

	return map[string]interface{}{
		"estudiante": estudianteMap,
		"periodo": map[string]interface{}{
			"id":       periodo.ID,
			"year":     periodo.Year,
			"semestre": periodo.Semestre,
		},
		"clases": clases,
	}, nil
}

func (s *MatriculaService) ValidarModificaciones(claims *models.JWTClaims) (map[string]interface{}, error) {
	ctx, razon, err := s.PrepareModificacionesContext(claims)
	if err != nil {
		return nil, err
	}
	if razon != "" {
		return map[string]interface{}{
			"puede_modificar": false,
			"razon":           razon,
		}, nil
	}

	count, err := s.repo.CountMateriasMatriculadas(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return map[string]interface{}{
			"puede_modificar": false,
			"razon":           "No tienes asignaturas matriculadas en el periodo activo. Debes realizar la inscripción inicial primero.",
		}, nil
	}

	return map[string]interface{}{
		"puede_modificar": true,
		"razon":           "",
		"periodo":         ctx.Periodo,
	}, nil
}

func (s *MatriculaService) GetInscritosCredits(estudianteID, periodoID int) (int, error) {
	return s.repo.GetInscritosCredits(estudianteID, periodoID)
}

func (s *MatriculaService) GetCreditLimit(pensumID, semestre int) (int, error) {
	limite, err := s.repo.GetCreditLimit(pensumID, semestre)
	if err == nil {
		return limite, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return s.repo.GetCreditLimitFallback(pensumID, semestre)
}

func (s *MatriculaService) GetMateriasMatriculadas(estudianteID, periodoID int) ([]models.MateriaMatriculada, error) {
	base, err := s.repo.GetMateriasMatriculadasBase(estudianteID, periodoID)
	if err != nil {
		return nil, err
	}

	groupIDs := make([]int, 0, len(base))
	for _, m := range base {
		groupIDs = append(groupIDs, m.GrupoID)
	}
	horariosByGroup, err := s.repo.GetHorariosForGroups(groupIDs)
	if err != nil {
		return nil, err
	}

	result := make([]models.MateriaMatriculada, 0, len(base))
	for _, m := range base {
		esAtrasada, esPerdida := s.determinarEstadoMateria(estudianteID, m.AsignaturaID, periodoID)
		result = append(result, models.MateriaMatriculada{
			HistorialID:  m.HistorialID,
			AsignaturaID: m.AsignaturaID,
			Codigo:       m.Codigo,
			Nombre:       m.Nombre,
			Creditos:     m.Creditos,
			GrupoID:      m.GrupoID,
			GrupoCodigo:  m.GrupoCodigo,
			Docente:      m.Docente,
			Horarios:     horariosByGroup[m.GrupoID],
			EsAtrasada:   esAtrasada,
			EsPerdida:    esPerdida,
			PuedeRetirar: !esAtrasada && !esPerdida,
		})
	}
	return result, nil
}

func (s *MatriculaService) BuildModificacionesCoreData(ctx *MatriculaContext, noMateriasMsg string) (*ModificacionesCoreData, string, error) {
	if ctx == nil || ctx.Periodo == nil {
		return nil, "No hay un periodo académico activo. No puedes realizar modificaciones en este momento.", nil
	}

	materiasMatriculadas, err := s.GetMateriasMatriculadas(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, "", err
	}
	if len(materiasMatriculadas) == 0 {
		return nil, noMateriasMsg, nil
	}

	creditosInscritos, err := s.GetInscritosCredits(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, "", err
	}
	creditosMax, err := s.GetCreditLimit(ctx.PensumID, ctx.Semestre)
	if err != nil {
		return nil, "", err
	}
	creditosDisponibles := creditosMax - creditosInscritos
	if creditosDisponibles < 0 {
		creditosDisponibles = 0
	}

	return &ModificacionesCoreData{
		Periodo:              ctx.Periodo,
		MateriasMatriculadas: materiasMatriculadas,
		CreditosInscritos:    creditosInscritos,
		CreditosMax:          creditosMax,
		CreditosDisponibles:  creditosDisponibles,
		EstadoEstudiante:     ctx.Estado,
	}, "", nil
}

func (s *MatriculaService) prepareBaseContext(claims *models.JWTClaims) (*MatriculaContext, string, error) {
	estudianteID, semestre, estado, err := s.repo.GetEstudianteBase(claims.Sub)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "Estudiante no encontrado", nil
	}
	if err != nil {
		return nil, "", err
	}
	pensumID, pensumNombre, programaNombre, err := s.pensumRepo.GetPensumInfo(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No tienes un pensum asignado. Contacta a coordinación para asignarte uno.", nil
	}
	if err != nil {
		return nil, "", err
	}
	periodo, err := s.repo.GetPeriodoActivo()
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No hay un periodo académico activo.", nil
	}
	if err != nil {
		return nil, "", err
	}
	plazos, err := s.repo.GetPlazos(periodo.ID, claims.ProgramaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "No hay plazos configurados para tu programa en el periodo activo.", nil
	}
	if err != nil {
		return nil, "", err
	}
	return &MatriculaContext{
		EstudianteID:   estudianteID,
		Semestre:       semestre,
		Estado:         estado,
		PensumID:       pensumID,
		PensumNombre:   pensumNombre,
		ProgramaID:     claims.ProgramaID,
		ProgramaNombre: programaNombre,
		Periodo:        periodo,
		Plazos:         *plazos,
	}, "", nil
}

func (s *MatriculaService) determinarEstadoMateria(estudianteID, asignaturaID, periodoActualID int) (bool, bool) {
	periodoYear, periodoSemestre, err := s.repo.GetPeriodoYearSemestreByID(periodoActualID)
	if err != nil {
		return false, false
	}
	periodoActualOrdinal := periodOrdinalService(periodoYear, periodoSemestre)

	historial, err := s.repo.GetHistorialPrevioAsignatura(estudianteID, asignaturaID, periodoActualID)
	if err != nil {
		return false, false
	}

	var tieneReprobada bool
	var ultimoOrdinal int
	for _, h := range historial {
		ord := periodOrdinalService(h.Year, h.Semestre)
		if ultimoOrdinal == 0 {
			ultimoOrdinal = ord
		}
		if h.Estado == "reprobada" {
			tieneReprobada = true
		}
		if h.Estado == "aprobada" || h.Estado == "convalidada" {
			break
		}
	}

	esPerdida := tieneReprobada
	esAtrasada := ultimoOrdinal > 0 && periodoActualOrdinal > ultimoOrdinal+2
	return esAtrasada, esPerdida
}

func periodOrdinalService(year, semestre int) int {
	return year*2 + semestre
}

func (s *MatriculaService) GetNucleoComunOtrasCarreras(programaID int) ([]models.AsignaturaCompleta, error) {
	return s.repo.GetNucleoComunOtrasCarreras(programaID)
}

func (s *MatriculaService) GetProgramasNucleoComun(asignaturaID int) ([]repositories.ProgramaInfo, error) {
	return s.repo.GetProgramasNucleoComun(asignaturaID)
}

func (s *MatriculaService) GetProgramaPorGrupo(asignaturaID int) (*repositories.ProgramaInfo, error) {
	info, err := s.repo.GetProgramaPorGrupo(asignaturaID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return info, nil
}
