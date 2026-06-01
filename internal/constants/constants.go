// Package constants centraliza todos los valores literales (strings, números)
// que se usan repetidamente en la aplicación.
//
// Principios aplicados:
//   - DRY: un solo lugar de definición para cada constante.
//   - OCP: agregar valores nuevos no requiere modificar los handlers.
//   - GRASP High Cohesion: este paquete tiene un único propósito claro.
package constants

// ─── Roles de usuario ────────────────────────────────────────────────────────

const (
	// RolEstudiante identifica al rol de usuario "estudiante" en el sistema.
	RolEstudiante = "estudiante"

	// RolJefe identifica al rol de usuario "jefe departamental" en el sistema.
	RolJefe = "jefe_departamental"
)

// ─── Estados de documentos ───────────────────────────────────────────────────

const (
	// EstadoDocAprobado indica que el documento fue revisado y aceptado.
	EstadoDocAprobado = "aprobado"

	// EstadoDocPendiente indica que el documento está en espera de revisión.
	EstadoDocPendiente = "pendiente"

	// EstadoDocRechazado indica que el documento fue revisado y rechazado.
	EstadoDocRechazado = "rechazado"
)

// ─── Tipos de documento requeridos ───────────────────────────────────────────

const (
	// TipoCertificadoEPS identifica el tipo de documento "Certificado EPS".
	TipoCertificadoEPS = "certificado_eps"

	// TipoComprobanteMatricula identifica el tipo "Comprobante de Matrícula".
	TipoComprobanteMatricula = "comprobante_matricula"

	// DocsRequeridosInscripcion es la cantidad mínima de documentos aprobados
	// que necesita un estudiante para poder inscribir asignaturas.
	DocsRequeridosInscripcion = 2
)

// ─── Límites de archivos ──────────────────────────────────────────────────────

const (
	// MaxDocumentoBytes es el tamaño máximo permitido para documentos (5 MB).
	MaxDocumentoBytes = 5 * 1024 * 1024

	// MaxFotoBytes es el tamaño máximo permitido para fotos de perfil (8 MB).
	MaxFotoBytes = 8 * 1024 * 1024
)

// ─── Extensiones de archivo ───────────────────────────────────────────────────

// ExtensionesDocumento lista las extensiones permitidas para documentos académicos.
var ExtensionesDocumento = []string{".pdf", ".png", ".jpg", ".jpeg"}

// ExtensionesFoto lista las extensiones permitidas para fotos de perfil.
var ExtensionesFoto = []string{".jpg", ".jpeg", ".png"}

// ─── Opciones de datos personales ────────────────────────────────────────────

// SexosPermitidos define los valores válidos para el campo sexo de un usuario.
// Se define como mapa para lograr O(1) en la validación.
var SexosPermitidos = map[string]struct{}{
	"masculino": {},
	"femenino":  {},
	"otro":      {},
}

// ─── Paginación ───────────────────────────────────────────────────────────────

const (
	// DefaultAuditLimit es el número de registros de auditoría retornados por defecto.
	DefaultAuditLimit = "50"
)
