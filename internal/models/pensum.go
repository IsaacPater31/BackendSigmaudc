package models

// Asignatura representa una asignatura del catálogo
type Asignatura struct {
	ID               int    `json:"id"`
	Codigo           string `json:"codigo"`
	Nombre           string `json:"nombre"`
	Creditos         int    `json:"creditos"`
	TipoID           int    `json:"tipo_id"`
	TipoNombre       string `json:"tipo_nombre"`
	TieneLaboratorio bool   `json:"tiene_laboratorio"`
}

// Pensum representa un pensum de un programa
type Pensum struct {
	ID         int    `json:"id"`
	ProgramaID int    `json:"programa_id"`
	Nombre     string `json:"nombre"`
	Activo     bool   `json:"activo"`
}

// PensumAsignatura representa una asignatura dentro de un pensum
type PensumAsignatura struct {
	ID           int    `json:"id"`
	PensumID     int    `json:"pensum_id"`
	AsignaturaID int    `json:"asignatura_id"`
	Semestre     int    `json:"semestre"`
	Categoria    string `json:"categoria"` // obligatoria, profundizacion, electiva
}

// Prerequisito representa un prerrequisito de una asignatura
type Prerequisito struct {
	ID             int      `json:"id"`
	AsignaturaID   int      `json:"asignatura_id"`
	PrerequisitoID int      `json:"prerequisito_id"`
	Codigo         string   `json:"codigo"`
	Nombre         string   `json:"nombre"`
	Completado     bool     `json:"completado"` // Si el prerrequisito está aprobado
	Semestre       int      `json:"semestre"`   // Semestre del prerrequisito para visualización
	Tipo           string   `json:"tipo"`
	PosicionX      *float64 `json:"posicion_x,omitempty"` // Posición X para visualización
	PosicionY      *float64 `json:"posicion_y,omitempty"` // Posición Y para visualización
}

// EstudianteAsignatura representa el estado de una asignatura para un estudiante
type EstudianteAsignatura struct {
	ID           int      `json:"id"`
	EstudianteID int      `json:"estudiante_id"`
	AsignaturaID int      `json:"asignatura_id"`
	Estado       string   `json:"estado"` // activa, matriculada, cursada, en_espera, pendiente_repeticion, obligatoria_repeticion
	Nota         *float64 `json:"nota"`
	Repeticiones int      `json:"repeticiones"`
}

// AsignaturaCompleta representa una asignatura con toda su información para el pensum
type AsignaturaCompleta struct {
	ID                     int            `json:"id"`
	Codigo                 string         `json:"codigo"`
	Nombre                 string         `json:"nombre"`
	Creditos               int            `json:"creditos"`
	TipoNombre             string         `json:"tipo_nombre"`
	TieneLaboratorio       bool           `json:"tiene_laboratorio"`
	Semestre               int            `json:"semestre"`
	Categoria              string         `json:"categoria"`
	Estado                 *string        `json:"estado"` // puede ser null si no está en estudiante_asignatura
	Nota                   *float64       `json:"nota"`
	Repeticiones           int            `json:"repeticiones"`
	GrupoID                *int           `json:"grupo_id,omitempty"`
	PeriodoCursada         *string        `json:"periodo_cursada,omitempty"`
	Prerequisitos          []Prerequisito `json:"prerequisitos"`
	PrerequisitosFaltantes []Prerequisito `json:"prerequisitos_faltantes"` // Prerrequisitos que aún no están aprobados
	PosicionX              *float64       `json:"posicion_x,omitempty"`    // Posición X para visualización
	PosicionY              *float64       `json:"posicion_y,omitempty"`    // Posición Y para visualización
}

// SemestrePensum representa un semestre con sus asignaturas
type SemestrePensum struct {
	Numero      int                  `json:"numero"`
	Asignaturas []AsignaturaCompleta `json:"asignaturas"`
}

// PensumEstudianteResponse representa la respuesta completa del pensum de un estudiante
type PensumEstudianteResponse struct {
	ProgramaNombre string           `json:"programa_nombre"`
	PensumNombre   string           `json:"pensum_nombre"`
	Semestres      []SemestrePensum `json:"semestres"`
}

type PensumItem struct {
	ID     int    `json:"id"`
	Nombre string `json:"nombre"`
}

type HorarioDisponible struct {
	Dia        string `json:"dia"`
	HoraInicio string `json:"hora_inicio"`
	HoraFin    string `json:"hora_fin"`
	Salon      string `json:"salon"`
}

type GrupoPensum struct {
	ID               int                 `json:"id"`
	Codigo           string              `json:"codigo"`
	AsignaturaID     int                 `json:"asignatura_id"`
	AsignaturaCodigo string              `json:"asignatura_codigo"`
	AsignaturaNombre string              `json:"asignatura_nombre"`
	Semestre         int                 `json:"semestre"`
	Creditos         int                 `json:"creditos"`
	Docente          string              `json:"docente"`
	CupoDisponible   int                 `json:"cupo_disponible"`
	CupoMax          int                 `json:"cupo_max"`
	Horarios         []HorarioDisponible `json:"horarios"`
}
