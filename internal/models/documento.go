package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NullStringJSON es un tipo que serializa sql.NullString como string o null en JSON
type NullStringJSON struct {
	sql.NullString
}

// MarshalJSON implementa la serialización JSON personalizada
func (n NullStringJSON) MarshalJSON() ([]byte, error) {
	if n.Valid {
		return json.Marshal(n.String)
	}
	return json.Marshal(nil)
}

// UnmarshalJSON implementa la deserialización JSON personalizada
func (n *NullStringJSON) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		n.Valid = false
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.String = s
	n.Valid = true
	return nil
}

// DocumentoEstudiante representa un documento subido por un estudiante
type DocumentoEstudiante struct {
	ID            int            `json:"id"`
	EstudianteID  int            `json:"estudiante_id"`
	ProgramaID    int            `json:"programa_id"`
	PeriodoID     int            `json:"periodo_id"`
	TipoDocumento string         `json:"tipo_documento"` // "certificado_eps" o "comprobante_matricula"
	ArchivoURL    string         `json:"archivo_url"`
	Estado        string         `json:"estado"` // "pendiente", "aprobado", "rechazado"
	Observacion   NullStringJSON `json:"observacion,omitempty"`
	RevisadoPor   sql.NullInt64  `json:"revisado_por,omitempty"`
	FechaSubida   time.Time      `json:"fecha_subida"`
	FechaRevision sql.NullTime   `json:"fecha_revision,omitempty"`
	// Campos adicionales para respuesta
	EstudianteNombre    string `json:"estudiante_nombre,omitempty"`
	EstudianteApellido  string `json:"estudiante_apellido,omitempty"`
	EstudianteCodigo    string `json:"estudiante_codigo,omitempty"`
	RevisorNombre       string `json:"revisor_nombre,omitempty"`
	RevisorApellido     string `json:"revisor_apellido,omitempty"`
}

// SubirDocumentoRequest representa la solicitud para subir un documento
type SubirDocumentoRequest struct {
	TipoDocumento string `json:"tipo_documento"` // "certificado_eps" o "comprobante_matricula"
}

// RevisarDocumentoRequest representa la solicitud para revisar un documento
type RevisarDocumentoRequest struct {
	Estado      string `json:"estado"`       // "aprobado" o "rechazado"
	Observacion string `json:"observacion"`  // Obligatorio si estado = "rechazado"
}

// DocumentosEstudianteResponse representa la respuesta con los documentos de un estudiante
type DocumentosEstudianteResponse struct {
	Documentos         []DocumentoEstudiante `json:"documentos"`
	PeriodoActivo      *PeriodoAcademico     `json:"periodo_activo,omitempty"`
	PlazoDocumentos    bool                  `json:"plazo_documentos"`     // Si el plazo está activo
	PuedeSubir         bool                  `json:"puede_subir"`           // Si puede subir documentos
	DocumentosAprobados bool                 `json:"documentos_aprobados"` // Si todos los documentos están aprobados
	PlazoMensaje       string                `json:"plazo_mensaje,omitempty"`
}

