package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/lib/pq"
)

type matriculaValidationError struct {
	status int
	msg    string
}

func (e *matriculaValidationError) Error() string {
	return e.msg
}

func matriculaValErr(status int, format string, args ...interface{}) error {
	return &matriculaValidationError{status: status, msg: fmt.Sprintf(format, args...)}
}

func respondMatriculaValidation(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	var ve *matriculaValidationError
	if errors.As(err, &ve) {
		http.Error(w, ve.msg, ve.status)
		return true
	}
	log.Printf("Error de validación interno: %v", err)
	http.Error(w, "Internal server error", http.StatusInternalServerError)
	return true
}

func dedupePositiveIDs(ids []int, label string) ([]int, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	seen := make(map[int]struct{}, len(ids))
	result := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, matriculaValErr(http.StatusBadRequest, "%s inválido.", label)
		}
		if _, ok := seen[id]; ok {
			return nil, matriculaValErr(http.StatusBadRequest, "No puedes repetir el mismo %s en la solicitud.", label)
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

type solicitudGrupoAgregarStored struct {
	GrupoID          int    `json:"grupo_id"`
	GrupoCodigo      string `json:"grupo_codigo"`
	AsignaturaID     int    `json:"asignatura_id"`
	AsignaturaCodigo string `json:"asignatura_codigo"`
	AsignaturaNombre string `json:"asignatura_nombre"`
	Creditos         int    `json:"creditos"`
	Docente          string `json:"docente,omitempty"`
}

type solicitudGrupoRetirarStored struct {
	HistorialID      int    `json:"historial_id"`
	GrupoID          int    `json:"grupo_id"`
	GrupoCodigo      string `json:"grupo_codigo"`
	AsignaturaID     int    `json:"asignatura_id"`
	AsignaturaCodigo string `json:"asignatura_codigo"`
	AsignaturaNombre string `json:"asignatura_nombre"`
	Creditos         int    `json:"creditos"`
}

func parseGrupoIDsFromPayload(raw json.RawMessage) ([]int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var asInts []int
	if err := json.Unmarshal(raw, &asInts); err == nil {
		return dedupePositiveIDs(asInts, "grupo")
	}
	var asObjects []struct {
		GrupoID int `json:"grupo_id"`
	}
	if err := json.Unmarshal(raw, &asObjects); err != nil {
		return nil, matriculaValErr(http.StatusBadRequest, "Formato inválido en grupos a agregar.")
	}
	ids := make([]int, 0, len(asObjects))
	for _, item := range asObjects {
		ids = append(ids, item.GrupoID)
	}
	return dedupePositiveIDs(ids, "grupo")
}

func parseHistorialIDsFromPayload(raw json.RawMessage) ([]int, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var asInts []int
	if err := json.Unmarshal(raw, &asInts); err == nil {
		return dedupePositiveIDs(asInts, "historial")
	}
	var asObjects []struct {
		HistorialID int `json:"historial_id"`
	}
	if err := json.Unmarshal(raw, &asObjects); err != nil {
		return nil, matriculaValErr(http.StatusBadRequest, "Formato inválido en grupos a retirar.")
	}
	ids := make([]int, 0, len(asObjects))
	for _, item := range asObjects {
		ids = append(ids, item.HistorialID)
	}
	return dedupePositiveIDs(ids, "historial")
}

func (h *MatriculaHandler) buildAsignaturaMapModificacion(ctx *inscripcionContext) (map[int]models.AsignaturaCompleta, error) {
	pensumHandler := &PensumHandler{db: h.db}
	asignaturas, err := pensumHandler.getAsignaturas(ctx.PensumID)
	if err != nil {
		return nil, err
	}
	asignaturaMap := make(map[int]models.AsignaturaCompleta, len(asignaturas))
	for _, asig := range asignaturas {
		asignaturaMap[asig.ID] = asig
	}
	nucleoComun, err := h.fetchNucleoComunOtrasCarreras(ctx.ProgramaID)
	if err == nil {
		for _, asig := range nucleoComun {
			asignaturaMap[asig.ID] = asig
		}
	}
	return asignaturaMap, nil
}

func (h *MatriculaHandler) validateObligatoriasRepeticionInscripcion(ctx *inscripcionContext, asignaturaMap map[int]models.AsignaturaCompleta, stateMap map[int]string) error {
	obligatorias := make([]int, 0)
	for id, state := range stateMap {
		if state == "obligatoria_repeticion" {
			obligatorias = append(obligatorias, id)
		}
	}
	if len(obligatorias) == 0 {
		return nil
	}
	query := `
		SELECT asignatura_id, COUNT(*) FILTER (WHERE cupo_disponible > 0) AS disponibles
		FROM grupo
		WHERE periodo_id = $1 AND asignatura_id = ANY($2)
		GROUP BY asignatura_id
	`
	rows, err := h.db.Query(query, ctx.Periodo.ID, pq.Array(obligatorias))
	if err != nil {
		return err
	}
	defer rows.Close()
	disponibles := map[int]int{}
	for rows.Next() {
		var asignaturaID, cantidad int
		if err := rows.Scan(&asignaturaID, &cantidad); err != nil {
			continue
		}
		disponibles[asignaturaID] = cantidad
	}
	for _, id := range obligatorias {
		if disponibles[id] == 0 {
			asig := asignaturaMap[id]
			return matriculaValErr(http.StatusConflict,
				"La asignatura %s %s está en repetición obligatoria y no tiene cupos disponibles, por lo tanto no puedes matricular otras asignaturas.",
				asig.Codigo, asig.Nombre)
		}
	}
	return nil
}

func (h *MatriculaHandler) loadSelectedGroups(ctx *inscripcionContext, grupoIDs []int) (map[int]groupRecord, error) {
	if len(grupoIDs) == 0 {
		return map[int]groupRecord{}, nil
	}
	query := `
		SELECT g.id, g.codigo, g.asignatura_id, g.cupo_disponible, g.cupo_max, a.creditos, COALESCE(g.docente, '')
		FROM grupo g
		JOIN asignatura a ON a.id = g.asignatura_id
		WHERE g.periodo_id = $1 AND g.id = ANY($2)
	`
	rows, err := h.db.Query(query, ctx.Periodo.ID, pq.Array(grupoIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	selected := make(map[int]groupRecord)
	for rows.Next() {
		var reg groupRecord
		var docente string
		if err := rows.Scan(&reg.ID, &reg.Codigo, &reg.AsignaturaID, &reg.CupoDisponible, &reg.CupoMax, &reg.Creditos, &docente); err != nil {
			return nil, err
		}
		selected[reg.ID] = reg
	}
	if len(selected) != len(grupoIDs) {
		return nil, matriculaValErr(http.StatusBadRequest, "Algunos grupos solicitados no existen o no pertenecen al periodo activo.")
	}
	return selected, nil
}

func (h *MatriculaHandler) validatePrerrequisitosSeleccion(
	selectedAsignaturas map[int]int,
	prereqs map[int][]models.Prerequisito,
	historialMap map[int][]historyRecord,
	asignaturaMap map[int]models.AsignaturaCompleta,
	accion string,
) error {
	selectedSet := make(map[int]struct{}, len(selectedAsignaturas))
	for asignaturaID := range selectedAsignaturas {
		selectedSet[asignaturaID] = struct{}{}
	}
	for asignaturaID := range selectedSet {
		for _, prereq := range prereqs[asignaturaID] {
			if prereq.Tipo == "correquisito" {
				if hasApprovedEntry(historialMap, prereq.PrerequisitoID) {
					continue
				}
				if _, ok := selectedSet[prereq.PrerequisitoID]; !ok {
					asigNombre := asignaturaMap[asignaturaID].Nombre
					preNombre := asignaturaMap[prereq.PrerequisitoID].Nombre
					return matriculaValErr(http.StatusBadRequest,
						"Para %s %s debes llevar también %s como correquisito.", accion, asigNombre, preNombre)
				}
				continue
			}
			if !hasApprovedEntry(historialMap, prereq.PrerequisitoID) {
				return matriculaValErr(http.StatusBadRequest,
					"Te falta aprobar %s para %s %s.",
					assignmentDisplay(prereq.PrerequisitoID, asignaturaMap),
					accion,
					asignaturaMap[asignaturaID].Nombre)
			}
		}
	}
	return nil
}

func (h *MatriculaHandler) validateConflictosHorario(nuevos []horarioBloque, existentes []horarioBloque) error {
	checked := make([]horarioBloque, 0, len(nuevos))
	for _, bloque := range nuevos {
		for _, existente := range existentes {
			if horariosOverlap(bloque, existente) {
				return matriculaValErr(http.StatusConflict, "Conflicto de horario con asignaturas ya matriculadas.")
			}
		}
		for _, previo := range checked {
			if horariosOverlap(bloque, previo) {
				return matriculaValErr(http.StatusConflict, "Hay conflicto de horario entre dos grupos seleccionados.")
			}
		}
		checked = append(checked, bloque)
	}
	return nil
}

func (h *MatriculaHandler) fetchHorariosMatriculadosExcluyendo(estudianteID, periodoID int, excluirGrupoIDs []int) ([]horarioBloque, error) {
	all, err := h.fetchHorariosInscritos(estudianteID, periodoID)
	if err != nil {
		return nil, err
	}
	if len(excluirGrupoIDs) == 0 {
		return all, nil
	}
	excluir := make(map[int]struct{}, len(excluirGrupoIDs))
	for _, id := range excluirGrupoIDs {
		excluir[id] = struct{}{}
	}
	result := make([]horarioBloque, 0, len(all))
	for _, bloque := range all {
		if _, ok := excluir[bloque.GrupoID]; !ok {
			result = append(result, bloque)
		}
	}
	return result, nil
}

func (h *MatriculaHandler) isAsignaturaMatriculada(estudianteID, periodoID, asignaturaID int) (bool, error) {
	var count int
	err := h.db.QueryRow(`
		SELECT COUNT(*) FROM historial_academico
		WHERE id_estudiante = $1 AND id_asignatura = $2 AND id_periodo = $3 AND estado = 'matriculada'
	`, estudianteID, asignaturaID, periodoID).Scan(&count)
	return count > 0, err
}

// validateGruposAgregarInscripcion valida reglas de inscripción inicial.
func (h *MatriculaHandler) validateGruposAgregarInscripcion(ctx *inscripcionContext, grupoIDs []int) (map[int]groupRecord, error) {
	grupoIDs, err := dedupePositiveIDs(grupoIDs, "grupo")
	if err != nil {
		return nil, err
	}
	if len(grupoIDs) == 0 {
		return nil, matriculaValErr(http.StatusBadRequest, "Debes seleccionar al menos un grupo para inscribir.")
	}

	pensumHandler := &PensumHandler{db: h.db}
	asignaturas, err := pensumHandler.getAsignaturas(ctx.PensumID)
	if err != nil {
		return nil, err
	}
	prereqs, err := pensumHandler.buildPrereqMap(ctx.PensumID)
	if err != nil {
		return nil, err
	}
	historialMap, err := pensumHandler.buildHistorialMap(ctx.EstudianteID)
	if err != nil {
		return nil, err
	}

	activeOrdinal := periodOrdinal(ctx.Periodo.Year, ctx.Periodo.Semestre)
	asignaturaMap := make(map[int]models.AsignaturaCompleta, len(asignaturas))
	stateMap := make(map[int]string, len(asignaturas))
	for _, asig := range asignaturas {
		asignaturaMap[asig.ID] = asig
		prereqsFalt := 0
		for _, p := range prereqs[asig.ID] {
			if p.Tipo != "correquisito" && !hasApprovedEntry(historialMap, p.PrerequisitoID) {
				prereqsFalt++
			}
		}
		state, _, _, _, _ := determineEstado(historialMap[asig.ID], ctx.Periodo, &activeOrdinal, prereqsFalt > 0)
		stateMap[asig.ID] = state
	}

	if err := h.validateObligatoriasRepeticionInscripcion(ctx, asignaturaMap, stateMap); err != nil {
		return nil, err
	}

	selectedGroups, err := h.loadSelectedGroups(ctx, grupoIDs)
	if err != nil {
		return nil, err
	}

	selectedAsignaturas := make(map[int]int)
	creditosNuevos := 0
	for _, grupoID := range grupoIDs {
		reg := selectedGroups[grupoID]
		if reg.CupoDisponible <= 0 {
			return nil, matriculaValErr(http.StatusConflict, "El grupo %s ya no tiene cupos disponibles.", reg.Codigo)
		}

		state, exists := stateMap[reg.AsignaturaID]
		if !exists {
			return nil, matriculaValErr(http.StatusBadRequest, "Asignatura fuera del pensum.")
		}
		switch state {
		case "matriculada":
			return nil, matriculaValErr(http.StatusConflict, "Ya estás matriculado en %s.", asignaturaMap[reg.AsignaturaID].Codigo)
		case "cursada":
			return nil, matriculaValErr(http.StatusConflict, "No puedes volver a inscribir %s porque ya la aprobaste.", asignaturaMap[reg.AsignaturaID].Codigo)
		case "en_espera":
			return nil, matriculaValErr(http.StatusConflict, "No puedes inscribir %s hasta que apruebes los prerrequisitos.", asignaturaMap[reg.AsignaturaID].Codigo)
		}

		asigInfo := asignaturaMap[reg.AsignaturaID]
		isAtrasada := state == "pendiente_repeticion" || state == "obligatoria_repeticion"
		if asigInfo.Semestre > ctx.Semestre && !isAtrasada {
			return nil, matriculaValErr(http.StatusConflict,
				"No puedes inscribir %s porque pertenece a un semestre superior. Debes solicitarla por modificaciones.", asigInfo.Codigo)
		}

		if _, ok := selectedAsignaturas[reg.AsignaturaID]; ok {
			return nil, matriculaValErr(http.StatusBadRequest, "Solo puedes seleccionar un grupo por asignatura.")
		}
		selectedAsignaturas[reg.AsignaturaID] = reg.ID
		creditosNuevos += reg.Creditos
	}

	if err := h.validateGroupsHorarioCompleto(grupoIDs); err != nil {
		return nil, matriculaValErr(http.StatusConflict, "%s", err.Error())
	}
	if err := h.validatePrerrequisitosSeleccion(selectedAsignaturas, prereqs, historialMap, asignaturaMap, "inscribir"); err != nil {
		return nil, err
	}

	existingHorarios, err := h.fetchHorariosInscritos(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, err
	}
	nuevosHorarios, err := h.fetchGroupScheduleBlocks(grupoIDs)
	if err != nil {
		return nil, err
	}
	if err := h.validateConflictosHorario(nuevosHorarios, existingHorarios); err != nil {
		return nil, err
	}

	creditosInscritos, err := h.fetchInscritosCredits(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, err
	}
	creditosMax, err := h.fetchCreditLimit(ctx.PensumID, ctx.Semestre)
	if err != nil {
		return nil, err
	}
	if creditosInscritos+creditosNuevos > creditosMax {
		return nil, matriculaValErr(http.StatusConflict,
			"Inscribir estos grupos supera el límite de %d créditos para el semestre %d.", creditosMax, ctx.Semestre)
	}

	return selectedGroups, nil
}

func (h *MatriculaHandler) validateRetirosModificacion(ctx *inscripcionContext, historialIDs []int) ([]solicitudGrupoRetirarStored, int, error) {
	historialIDs, err := dedupePositiveIDs(historialIDs, "historial")
	if err != nil {
		return nil, 0, err
	}
	if len(historialIDs) == 0 {
		return nil, 0, nil
	}

	result := make([]solicitudGrupoRetirarStored, 0, len(historialIDs))
	creditosRetirados := 0
	for _, historialID := range historialIDs {
		var item solicitudGrupoRetirarStored
		var estado string
		err := h.db.QueryRow(`
			SELECT h.id, h.grupo_id, h.id_asignatura, h.estado,
			       g.codigo, a.codigo, a.nombre, a.creditos
			FROM historial_academico h
			JOIN grupo g ON g.id = h.grupo_id
			JOIN asignatura a ON a.id = h.id_asignatura
			WHERE h.id = $1 AND h.id_estudiante = $2 AND h.id_periodo = $3
		`, historialID, ctx.EstudianteID, ctx.Periodo.ID).Scan(
			&item.HistorialID, &item.GrupoID, &item.AsignaturaID, &estado,
			&item.GrupoCodigo, &item.AsignaturaCodigo, &item.AsignaturaNombre, &item.Creditos,
		)
		if err != nil {
			return nil, 0, matriculaValErr(http.StatusBadRequest, "Retiro inválido o no pertenece a tu matrícula actual.")
		}
		if estado != "matriculada" {
			return nil, 0, matriculaValErr(http.StatusConflict, "La materia %s no está en estado matriculada.", item.AsignaturaCodigo)
		}
		esAtrasada, esPerdida := h.determinarEstadoMateria(ctx.EstudianteID, item.AsignaturaID, ctx.Periodo.ID)
		if esAtrasada || esPerdida {
			return nil, 0, matriculaValErr(http.StatusConflict,
				"No puedes retirar %s porque está marcada como atrasada o perdida.", item.AsignaturaNombre)
		}
		result = append(result, item)
		creditosRetirados += item.Creditos
	}
	return result, creditosRetirados, nil
}

func (h *MatriculaHandler) validateGruposAgregarModificacion(
	ctx *inscripcionContext,
	grupoIDs []int,
	retiros []solicitudGrupoRetirarStored,
) (map[int]groupRecord, int, error) {
	grupoIDs, err := dedupePositiveIDs(grupoIDs, "grupo")
	if err != nil {
		return nil, 0, err
	}
	if len(grupoIDs) == 0 {
		return map[int]groupRecord{}, 0, nil
	}

	retiroAsignaturas := make(map[int]struct{}, len(retiros))
	retiroGrupos := make([]int, 0, len(retiros))
	for _, r := range retiros {
		retiroAsignaturas[r.AsignaturaID] = struct{}{}
		retiroGrupos = append(retiroGrupos, r.GrupoID)
	}

	pensumHandler := &PensumHandler{db: h.db}
	prereqs, err := pensumHandler.buildPrereqMap(ctx.PensumID)
	if err != nil {
		return nil, 0, err
	}
	historialMap, err := pensumHandler.buildHistorialMap(ctx.EstudianteID)
	if err != nil {
		return nil, 0, err
	}
	asignaturaMap, err := h.buildAsignaturaMapModificacion(ctx)
	if err != nil {
		return nil, 0, err
	}

	activeOrdinal := periodOrdinal(ctx.Periodo.Year, ctx.Periodo.Semestre)
	selectedGroups, err := h.loadSelectedGroups(ctx, grupoIDs)
	if err != nil {
		return nil, 0, err
	}

	selectedAsignaturas := make(map[int]int)
	creditosNuevos := 0
	for _, grupoID := range grupoIDs {
		reg := selectedGroups[grupoID]
		if reg.CupoDisponible <= 0 {
			return nil, 0, matriculaValErr(http.StatusConflict, "El grupo %s ya no tiene cupos disponibles.", reg.Codigo)
		}

		asig, ok := asignaturaMap[reg.AsignaturaID]
		if !ok {
			return nil, 0, matriculaValErr(http.StatusBadRequest, "Asignatura no disponible para modificaciones.")
		}
		if !puedeAgregarAsignaturaEnModificaciones(asig.Semestre, ctx.Semestre) {
			return nil, 0, matriculaValErr(http.StatusConflict, "%s", mensajeSemestreNoPermitidoModificacion(asig.Codigo))
		}

		prereqsFalt := 0
		for _, p := range prereqs[reg.AsignaturaID] {
			if p.Tipo != "correquisito" && !hasApprovedEntry(historialMap, p.PrerequisitoID) {
				prereqsFalt++
			}
		}
		state, _, _, _, _ := determineEstado(historialMap[reg.AsignaturaID], ctx.Periodo, &activeOrdinal, prereqsFalt > 0)
		if state == "cursada" {
			return nil, 0, matriculaValErr(http.StatusConflict, "No puedes agregar %s porque ya la aprobaste.", asig.Codigo)
		}
		if state == "en_espera" {
			return nil, 0, matriculaValErr(http.StatusConflict, "No puedes agregar %s hasta que apruebes los prerrequisitos.", asig.Codigo)
		}

		matriculada, err := h.isAsignaturaMatriculada(ctx.EstudianteID, ctx.Periodo.ID, reg.AsignaturaID)
		if err != nil {
			return nil, 0, err
		}
		if matriculada {
			if _, seRetira := retiroAsignaturas[reg.AsignaturaID]; !seRetira {
				return nil, 0, matriculaValErr(http.StatusConflict, "Ya estás matriculado en la asignatura %s.", asig.Codigo)
			}
		}

		if _, ok := selectedAsignaturas[reg.AsignaturaID]; ok {
			return nil, 0, matriculaValErr(http.StatusBadRequest, "Solo puedes seleccionar un grupo por asignatura.")
		}
		selectedAsignaturas[reg.AsignaturaID] = reg.ID
		creditosNuevos += reg.Creditos
	}

	if err := h.validateGroupsHorarioCompleto(grupoIDs); err != nil {
		return nil, 0, matriculaValErr(http.StatusConflict, "%s", err.Error())
	}
	if err := h.validatePrerrequisitosSeleccion(selectedAsignaturas, prereqs, historialMap, asignaturaMap, "agregar"); err != nil {
		return nil, 0, err
	}

	existingHorarios, err := h.fetchHorariosMatriculadosExcluyendo(ctx.EstudianteID, ctx.Periodo.ID, retiroGrupos)
	if err != nil {
		return nil, 0, err
	}
	nuevosHorarios, err := h.fetchGroupScheduleBlocks(grupoIDs)
	if err != nil {
		return nil, 0, err
	}
	if err := h.validateConflictosHorario(nuevosHorarios, existingHorarios); err != nil {
		return nil, 0, err
	}

	creditosInscritos, err := h.fetchInscritosCredits(ctx.EstudianteID, ctx.Periodo.ID)
	if err != nil {
		return nil, 0, err
	}
	creditosRetirados := 0
	for _, r := range retiros {
		creditosRetirados += r.Creditos
	}
	creditosMax, err := h.fetchCreditLimit(ctx.PensumID, ctx.Semestre)
	if err != nil {
		return nil, 0, err
	}
	creditosProyectados := creditosInscritos - creditosRetirados + creditosNuevos
	if creditosProyectados > creditosMax {
		return nil, 0, matriculaValErr(http.StatusConflict,
			"La solicitud supera el límite de %d créditos para el semestre %d.", creditosMax, ctx.Semestre)
	}

	return selectedGroups, creditosNuevos, nil
}

func (h *MatriculaHandler) enrichGruposAgregarStored(ctx *inscripcionContext, grupoIDs []int) ([]solicitudGrupoAgregarStored, error) {
	if len(grupoIDs) == 0 {
		return nil, nil
	}
	query := `
		SELECT g.id, g.codigo, a.id, a.codigo, a.nombre, a.creditos, COALESCE(g.docente, '')
		FROM grupo g
		JOIN asignatura a ON a.id = g.asignatura_id
		WHERE g.periodo_id = $1 AND g.id = ANY($2)
	`
	rows, err := h.db.Query(query, ctx.Periodo.ID, pq.Array(grupoIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int]solicitudGrupoAgregarStored)
	for rows.Next() {
		var item solicitudGrupoAgregarStored
		if err := rows.Scan(&item.GrupoID, &item.GrupoCodigo, &item.AsignaturaID, &item.AsignaturaCodigo, &item.AsignaturaNombre, &item.Creditos, &item.Docente); err != nil {
			return nil, err
		}
		byID[item.GrupoID] = item
	}
	result := make([]solicitudGrupoAgregarStored, 0, len(grupoIDs))
	for _, id := range grupoIDs {
		item, ok := byID[id]
		if !ok {
			return nil, matriculaValErr(http.StatusBadRequest, "Grupo %d no encontrado en el periodo activo.", id)
		}
		result = append(result, item)
	}
	return result, nil
}

// validateSolicitudModificacion valida retiros + agregados y devuelve payloads enriquecidos desde BD.
func (h *MatriculaHandler) validateSolicitudModificacion(
	ctx *inscripcionContext,
	gruposAgregarRaw json.RawMessage,
	gruposRetirarRaw json.RawMessage,
) (json.RawMessage, json.RawMessage, error) {
	grupoIDs, err := parseGrupoIDsFromPayload(gruposAgregarRaw)
	if err != nil {
		return nil, nil, err
	}
	historialIDs, err := parseHistorialIDsFromPayload(gruposRetirarRaw)
	if err != nil {
		return nil, nil, err
	}
	if len(grupoIDs) == 0 && len(historialIDs) == 0 {
		return nil, nil, matriculaValErr(http.StatusBadRequest, "Debes seleccionar al menos un grupo para agregar o una materia para retirar.")
	}

	retiros, _, err := h.validateRetirosModificacion(ctx, historialIDs)
	if err != nil {
		return nil, nil, err
	}
	if _, _, err := h.validateGruposAgregarModificacion(ctx, grupoIDs, retiros); err != nil {
		return nil, nil, err
	}

	agregarStored, err := h.enrichGruposAgregarStored(ctx, grupoIDs)
	if err != nil {
		return nil, nil, err
	}
	agregarJSON, err := json.Marshal(agregarStored)
	if err != nil {
		return nil, nil, err
	}
	retirarJSON, err := json.Marshal(retiros)
	if err != nil {
		return nil, nil, err
	}
	if retirarJSON == nil {
		retirarJSON = json.RawMessage("[]")
	}
	if agregarJSON == nil {
		agregarJSON = json.RawMessage("[]")
	}
	return agregarJSON, retirarJSON, nil
}

// loadModificacionesContextForApproval carga contexto para re-validar al aprobar (sin exigir plazo activo).
func (h *MatriculaHandler) loadModificacionesContextForApproval(estudianteID, periodoID int) (*inscripcionContext, error) {
	var semestre int
	var estado string
	err := h.db.QueryRow(`SELECT semestre, estado FROM estudiante WHERE id = $1`, estudianteID).Scan(&semestre, &estado)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, matriculaValErr(http.StatusNotFound, "Estudiante no encontrado.")
	}
	if err != nil {
		return nil, err
	}

	var periodo models.PeriodoAcademico
	err = h.db.QueryRow(`
		SELECT id, year, semestre, activo, archivado
		FROM periodo_academico WHERE id = $1
	`, periodoID).Scan(&periodo.ID, &periodo.Year, &periodo.Semestre, &periodo.Activo, &periodo.Archivado)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, matriculaValErr(http.StatusBadRequest, "Periodo de la solicitud no encontrado.")
	}
	if err != nil {
		return nil, err
	}

	var pensumID int
	var pensumNombre, programaNombre string
	err = h.db.QueryRow(`
		SELECT p.id, p.nombre, pr.nombre
		FROM estudiante_pensum ep
		JOIN pensum p ON ep.pensum_id = p.id
		JOIN programa pr ON p.programa_id = pr.id
		WHERE ep.estudiante_id = $1
	`, estudianteID).Scan(&pensumID, &pensumNombre, &programaNombre)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, matriculaValErr(http.StatusBadRequest, "El estudiante no tiene pensum asignado.")
	}
	if err != nil {
		return nil, err
	}

	var programaID int
	err = h.db.QueryRow(`
		SELECT u.programa_id
		FROM estudiante e
		JOIN usuario u ON u.id = e.usuario_id
		WHERE e.id = $1
	`, estudianteID).Scan(&programaID)
	if err != nil {
		return nil, err
	}

	return &inscripcionContext{
		EstudianteID:   estudianteID,
		Semestre:       semestre,
		Estado:         estado,
		PensumID:       pensumID,
		PensumNombre:   pensumNombre,
		ProgramaID:     programaID,
		ProgramaNombre: programaNombre,
		Periodo:        &periodo,
	}, nil
}
