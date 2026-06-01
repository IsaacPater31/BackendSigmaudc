package models

type HorarioClase struct {
	AsignaturaID     int    `json:"asignatura_id"`
	AsignaturaCodigo string `json:"asignatura_codigo"`
	AsignaturaNombre string `json:"asignatura_nombre"`
	GrupoID          int    `json:"grupo_id"`
	GrupoCodigo      string `json:"grupo_codigo"`
	Docente          string `json:"docente"`
	Dia              string `json:"dia"`
	HoraInicio       string `json:"hora_inicio"`
	HoraFin          string `json:"hora_fin"`
	Salon            string `json:"salon"`
}

type MateriaMatriculada struct {
	HistorialID  int                 `json:"historial_id"`
	AsignaturaID int                 `json:"asignatura_id"`
	Codigo       string              `json:"codigo"`
	Nombre       string              `json:"nombre"`
	Creditos     int                 `json:"creditos"`
	GrupoID      int                 `json:"grupo_id"`
	GrupoCodigo  string              `json:"grupo_codigo"`
	Docente      string              `json:"docente"`
	Horarios     []HorarioDisponible `json:"horarios"`
	EsAtrasada   bool                `json:"es_atrasada"`
	EsPerdida    bool                `json:"es_perdida"`
	PuedeRetirar bool                `json:"puede_retirar"`
}
