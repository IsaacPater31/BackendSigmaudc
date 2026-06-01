package services

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

var ErrPensumNoAsignado = errors.New("pensum no asignado")

type PensumService struct {
	repo *repositories.PensumRepository
}

func NewPensumService(repo *repositories.PensumRepository) *PensumService {
	return &PensumService{repo: repo}
}

func (s *PensumService) GetPensumEstudiante(usuarioID int) (*models.PensumEstudianteResponse, error) {
	estudianteID, err := s.repo.GetEstudianteID(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("estudiante no encontrado")
	}
	if err != nil {
		return nil, err
	}
	pensumID, pensumNombre, programaNombre, err := s.repo.GetPensumInfo(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrPensumNoAsignado
	}
	if err != nil {
		return nil, err
	}
	activePeriodo, err := s.repo.GetActivePeriodo()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		activePeriodo = nil
	}
	asignaturas, err := s.repo.GetAsignaturas(pensumID)
	if err != nil {
		return nil, err
	}
	prereqMap, err := s.repo.BuildPrereqMap(pensumID)
	if err != nil {
		return nil, err
	}
	historialMap, err := s.repo.BuildHistorialMap(estudianteID)
	if err != nil {
		return nil, err
	}

	activeOrdinal := (*int)(nil)
	if activePeriodo != nil {
		ord := periodOrdinal(activePeriodo.Year, activePeriodo.Semestre)
		activeOrdinal = &ord
	}
	semestresMap := make(map[int][]models.AsignaturaCompleta)
	for _, asig := range asignaturas {
		hist := historialMap[asig.ID]
		prereqs := make([]models.Prerequisito, 0, len(prereqMap[asig.ID]))
		for _, prereq := range prereqMap[asig.ID] {
			prereq.Completado = hasApprovedEntry(historialMap, prereq.PrerequisitoID)
			prereqs = append(prereqs, prereq)
		}
		var faltantes []models.Prerequisito
		for _, p := range prereqs {
			if !p.Completado {
				faltantes = append(faltantes, p)
			}
		}
		state, nota, grupoID, periodoCursada, repeticiones := determineEstado(hist, activePeriodo, activeOrdinal, len(faltantes) > 0)
		asig.Estado = &state
		asig.Nota = nota
		asig.Repeticiones = repeticiones
		asig.Prerequisitos = prereqs
		asig.PrerequisitosFaltantes = faltantes
		asig.GrupoID = grupoID
		asig.PeriodoCursada = periodoCursada
		semestresMap[asig.Semestre] = append(semestresMap[asig.Semestre], asig)
	}
	var semestres []models.SemestrePensum
	for semestre := range semestresMap {
		semestres = append(semestres, models.SemestrePensum{Numero: semestre, Asignaturas: semestresMap[semestre]})
	}
	sort.Slice(semestres, func(i, j int) bool { return semestres[i].Numero < semestres[j].Numero })
	return &models.PensumEstudianteResponse{
		ProgramaNombre: programaNombre,
		PensumNombre:   pensumNombre,
		Semestres:      semestres,
	}, nil
}

func (s *PensumService) ListPensums() ([]models.PensumItem, error) {
	return s.repo.ListPensums()
}

func (s *PensumService) GetAsignaturasPensum(pensumID int) ([]models.AsignaturaCompleta, error) {
	return s.repo.GetAsignaturas(pensumID)
}

func (s *PensumService) GetGruposPensum(pensumID int) ([]models.GrupoPensum, error) {
	periodo, err := s.repo.GetActivePeriodo()
	if err != nil {
		return nil, err
	}
	grupos, groupIDs, err := s.repo.GetGruposPensum(pensumID, periodo.ID)
	if err != nil {
		return nil, err
	}
	horariosMap, err := s.repo.FetchHorariosForGroups(groupIDs)
	if err == nil {
		for i := range grupos {
			grupos[i].Horarios = horariosMap[grupos[i].ID]
		}
	}
	return grupos, nil
}

func determineEstado(history []repositories.HistorialRecord, activePeriodo *models.PeriodoAcademico, activeOrdinal *int, tienePrereqPendientes bool) (string, *float64, *int, *string, int) {
	repeticiones := 0
	var lastReprob *repositories.HistorialRecord
	for _, entry := range history {
		if entry.Estado == "reprobada" {
			repeticiones++
			lastReprob = &entry
		}
		if activePeriodo != nil && entry.PeriodoID == activePeriodo.ID && entry.Estado == "matriculada" {
			var grupo *int
			if entry.GrupoID.Valid {
				g := int(entry.GrupoID.Int64)
				grupo = &g
			}
			var nota *float64
			if entry.Nota.Valid {
				n := entry.Nota.Float64
				nota = &n
			}
			return "matriculada", nota, grupo, nil, repeticiones
		}
	}
	for i := len(history) - 1; i >= 0; i-- {
		entry := history[i]
		if entry.Estado == "aprobada" && entry.Nota.Valid && entry.Nota.Float64 >= 3.0 {
			nota := entry.Nota.Float64
			periodo := fmt.Sprintf("%d-%d", entry.Year, entry.Semestre)
			return "cursada", &nota, nil, &periodo, repeticiones
		}
		if entry.Estado == "convalidada" {
			periodo := fmt.Sprintf("%d-%d", entry.Year, entry.Semestre)
			return "cursada", nil, nil, &periodo, repeticiones
		}
	}
	var notaPendiente *float64
	if lastReprob != nil && lastReprob.Nota.Valid {
		n := lastReprob.Nota.Float64
		notaPendiente = &n
	}
	if repeticiones >= 2 || (lastReprob != nil && shouldObligatoria(history, *lastReprob, activeOrdinal)) {
		return "obligatoria_repeticion", notaPendiente, nil, nil, repeticiones
	}
	if repeticiones == 1 {
		return "pendiente_repeticion", notaPendiente, nil, nil, repeticiones
	}
	if tienePrereqPendientes {
		return "en_espera", nil, nil, nil, repeticiones
	}
	return "activa", nil, nil, nil, repeticiones
}

func shouldObligatoria(history []repositories.HistorialRecord, lastReprob repositories.HistorialRecord, activeOrdinal *int) bool {
	if activeOrdinal == nil || *activeOrdinal < lastReprob.Ordinal+2 {
		return false
	}
	next := lastReprob.Ordinal + 1
	for _, entry := range history {
		if entry.Ordinal == next {
			return false
		}
	}
	return true
}

func hasApprovedEntry(historial map[int][]repositories.HistorialRecord, asignaturaID int) bool {
	for _, entry := range historial[asignaturaID] {
		if (entry.Estado == "aprobada" && entry.Nota.Valid && entry.Nota.Float64 >= 3.0) || entry.Estado == "convalidada" {
			return true
		}
	}
	return false
}

func periodOrdinal(year, semestre int) int { return year*2 + (semestre - 1) }
