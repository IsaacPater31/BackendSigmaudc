package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/gorilla/mux"
)

type PensumHandler struct {
	service *services.PensumService
	db      *sql.DB
}

func NewPensumHandler(service *services.PensumService) *PensumHandler {
	return &PensumHandler{service: service}
}

func (h *PensumHandler) GetPensumEstudiante(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolEstudiante {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	resp, err := h.service.GetPensumEstudiante(claims.Sub)
	if errors.Is(err, services.ErrPensumNoAsignado) {
		http.Error(w, "Pensum no asignado al estudiante", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Error obteniendo pensum", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *PensumHandler) ListPensums(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	list, err := h.service.ListPensums()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (h *PensumHandler) GetAsignaturasPensum(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	pensumID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || pensumID <= 0 {
		http.Error(w, "ID de pensum inválido", http.StatusBadRequest)
		return
	}
	asignaturas, err := h.service.GetAsignaturasPensum(pensumID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(asignaturas)
}

func (h *PensumHandler) GetGruposPensum(w http.ResponseWriter, r *http.Request) {
	claims, err := getClaims(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if claims.Rol != constants.RolJefe {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	pensumID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil || pensumID <= 0 {
		http.Error(w, "ID de pensum inválido", http.StatusBadRequest)
		return
	}
	grupos, err := h.service.GetGruposPensum(pensumID)
	if err != nil {
		http.Error(w, "No hay periodo académico activo", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(grupos)
}

// Compatibilidad temporal para MatriculaHandler hasta completar fase 3.
var errPensumNoAsignado = errors.New("pensum no asignado")

type historyRecord = repositories.HistorialRecord

func (h *PensumHandler) getPensumInfo(estudianteID int) (int, string, string, error) {
	repo := repositories.NewPensumRepository(h.db)
	pensumID, pensumNombre, programaNombre, err := repo.GetPensumInfo(estudianteID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", "", errPensumNoAsignado
	}
	return pensumID, pensumNombre, programaNombre, err
}

func (h *PensumHandler) getAsignaturas(pensumID int) ([]models.AsignaturaCompleta, error) {
	return repositories.NewPensumRepository(h.db).GetAsignaturas(pensumID)
}

func (h *PensumHandler) buildPrereqMap(pensumID int) (map[int][]models.Prerequisito, error) {
	return repositories.NewPensumRepository(h.db).BuildPrereqMap(pensumID)
}

func (h *PensumHandler) buildHistorialMap(estudianteID int) (map[int][]historyRecord, error) {
	return repositories.NewPensumRepository(h.db).BuildHistorialMap(estudianteID)
}

func periodOrdinal(year, semestre int) int { return year*2 + (semestre - 1) }

func hasApprovedEntry(historial map[int][]historyRecord, asignaturaID int) bool {
	for _, entry := range historial[asignaturaID] {
		if (entry.Estado == "aprobada" && entry.Nota.Valid && entry.Nota.Float64 >= 3.0) || entry.Estado == "convalidada" {
			return true
		}
	}
	return false
}

func shouldObligatoria(history []historyRecord, lastReprob historyRecord, activeOrdinal *int) bool {
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

func determineEstado(history []historyRecord, activePeriodo *models.PeriodoAcademico, activeOrdinal *int, tienePrereqPendientes bool) (string, *float64, *int, *string, int) {
	repeticiones := 0
	var lastReprob *historyRecord
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
