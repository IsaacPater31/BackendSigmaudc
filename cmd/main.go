// Package main es el punto de entrada de la aplicación SIGMAUDC.
// Carga la configuración, conecta la base de datos, inicializa los handlers
// con sus dependencias inyectadas y arranca el servidor HTTP.
//
// Principios aplicados:
//   - DIP: los handlers reciben sus dependencias (db, AuditoriaService) por inyección.
//   - SRP: main solo orquesta el arranque; la lógica vive en los paquetes internos.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/andrxsq/SIGMAUDC/internal/config"
	"github.com/andrxsq/SIGMAUDC/internal/database"
	"github.com/andrxsq/SIGMAUDC/internal/handlers"
	"github.com/andrxsq/SIGMAUDC/internal/middleware"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
	"github.com/andrxsq/SIGMAUDC/internal/services"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// ── 1. Variables de entorno ────────────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Println("No se encontró .env, usando variables de entorno del sistema")
	} else {
		log.Println("Archivo .env cargado correctamente")
	}

	// ── 2. Configuración ──────────────────────────────────────────────────────
	cfg := config.Load()

	// ── 3. Base de datos ──────────────────────────────────────────────────────
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Error conectando a la base de datos:", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatal("Error ejecutando migraciones:", err)
	}

	// ── 4. Servicios compartidos ──────────────────────────────────────────────
	// AuditoriaService se crea una vez y se inyecta en todos los handlers
	// que necesitan registrar eventos (DIP + GRASP Information Expert).
	auditoria := services.NewAuditoriaService(db)

	// ── 5. Handlers ───────────────────────────────────────────────────────────
	plazosRepository := repositories.NewPlazosRepository(db)
	plazosService := services.NewPlazosService(plazosRepository, auditoria)
	authRepository := repositories.NewAuthRepository(db)
	authService := services.NewAuthService(authRepository, auditoria, cfg.JWTSecret)
	auditRepository := repositories.NewAuditRepository(db)
	auditService := services.NewAuditService(auditRepository)
	profileRepository := repositories.NewProfileRepository(db)
	profileService := services.NewProfileService(profileRepository)
	documentosRepository := repositories.NewDocumentosRepository(db)
	documentosService := services.NewDocumentosService(documentosRepository, auditoria, os.Getenv("UPLOAD_DIR"))
	pensumRepository := repositories.NewPensumRepository(db)
	pensumService := services.NewPensumService(pensumRepository)
	matriculaRepository := repositories.NewMatriculaRepository(db)
	matriculaService := services.NewMatriculaService(matriculaRepository, pensumRepository)

	authHandler := handlers.NewAuthHandler(authService)
	auditHandler := handlers.NewAuditHandler(auditService)
	plazosHandler := handlers.NewPlazosHandler(plazosService)
	documentosHandler := handlers.NewDocumentosHandler(documentosService)
	pensumHandler := handlers.NewPensumHandler(pensumService)
	matriculaHandler := handlers.NewMatriculaHandler(db, matriculaService)
	estudianteHandler := handlers.NewEstudianteHandler(profileService)
	jefeHandler := handlers.NewJefeHandler(profileService)

	// ── 6. Router y rutas ────────────────────────────────────────────────────
	r := mux.NewRouter()

	// Rutas públicas (sin autenticación)
	r.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/auth/set-password", authHandler.SetPassword).Methods("POST")

	// Subrouter protegido: todas las rutas bajo /api requieren JWT válido
	protected := r.PathPrefix("/api").Subrouter()
	protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))

	// Subrouters por rol para aplicar autorización consistente.
	estudianteRoutes := protected.PathPrefix("").Subrouter()
	estudianteRoutes.Use(middleware.RequireRoles("estudiante"))

	jefeRoutes := protected.PathPrefix("").Subrouter()
	jefeRoutes.Use(middleware.RequireRoles("jefe_departamental"))

	// Perfil del usuario autenticado
	protected.HandleFunc("/me", authHandler.GetCurrentUser).Methods("GET")

	// Auditoría
	jefeRoutes.HandleFunc("/audit", auditHandler.GetAuditLogs).Methods("GET")

	// Plazos del periodo activo (periodos se gestionan en BD)
	protected.HandleFunc("/plazos/activo", plazosHandler.GetActivePeriodoPlazos).Methods("GET")
	jefeRoutes.HandleFunc("/periodos/{periodo_id}/plazos", plazosHandler.UpdatePlazos).Methods("PUT")

	// Documentos académicos
	estudianteRoutes.HandleFunc("/documentos", documentosHandler.GetDocumentosEstudiante).Methods("GET")
	estudianteRoutes.HandleFunc("/documentos", documentosHandler.SubirDocumento).Methods("POST")
	jefeRoutes.HandleFunc("/documentos/programa", documentosHandler.GetDocumentosPorPrograma).Methods("GET")
	jefeRoutes.HandleFunc("/documentos/{id}/revisar", documentosHandler.RevisarDocumento).Methods("PUT")

	// Pensum y asignaturas
	estudianteRoutes.HandleFunc("/pensum", pensumHandler.GetPensumEstudiante).Methods("GET")
	jefeRoutes.HandleFunc("/pensum/list", pensumHandler.ListPensums).Methods("GET")
	jefeRoutes.HandleFunc("/pensum/{id}/asignaturas", pensumHandler.GetAsignaturasPensum).Methods("GET")
	jefeRoutes.HandleFunc("/pensum/{id}/grupos", pensumHandler.GetGruposPensum).Methods("GET")

	// Datos personales del estudiante
	estudianteRoutes.HandleFunc("/estudiante/datos", estudianteHandler.GetDatosEstudiante).Methods("GET")
	estudianteRoutes.HandleFunc("/estudiante/datos", estudianteHandler.UpdateDatosEstudiante).Methods("PUT")
	estudianteRoutes.HandleFunc("/estudiante/foto", estudianteHandler.SubirFotoEstudiante).Methods("POST")

	// Datos personales del jefe departamental
	jefeRoutes.HandleFunc("/jefe/datos", jefeHandler.GetDatosJefe).Methods("GET")
	jefeRoutes.HandleFunc("/jefe/datos", jefeHandler.UpdateDatosJefe).Methods("PUT")
	jefeRoutes.HandleFunc("/jefe/foto", jefeHandler.SubirFotoJefe).Methods("POST")

	// Matrícula e inscripción
	estudianteRoutes.HandleFunc("/matricula/validar-inscripcion", matriculaHandler.ValidarInscripcion).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/asignaturas-disponibles", matriculaHandler.GetAsignaturasDisponibles).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/horario-actual", matriculaHandler.GetHorarioActual).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/asignaturas/{id}/grupos", matriculaHandler.GetGruposAsignatura).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/inscribir", matriculaHandler.InscribirAsignaturas).Methods("POST")
	jefeRoutes.HandleFunc("/grupo/{id}/horario", matriculaHandler.UpdateGrupoHorario).Methods("PUT")

	// Modificaciones de matrícula (jefatura)
	jefeRoutes.HandleFunc("/modificaciones/estudiante", matriculaHandler.GetStudentMatricula).Methods("GET")
	jefeRoutes.HandleFunc("/modificaciones/estudiante/{id}/disponibles", matriculaHandler.JefeGetModificacionesData).Methods("GET")
	jefeRoutes.HandleFunc("/modificaciones/estudiante/{id}/inscribir", matriculaHandler.JefeInscribirAsignaturas).Methods("POST")
	jefeRoutes.HandleFunc("/modificaciones/estudiante/{id}/desmatricular", matriculaHandler.JefeDesmatricularGrupo).Methods("POST")

	// Modificaciones de matrícula (estudiante)
	estudianteRoutes.HandleFunc("/matricula/validar-modificaciones", matriculaHandler.ValidarModificaciones).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/modificaciones", matriculaHandler.GetModificacionesData).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/retirar-materia", matriculaHandler.RetirarMateria).Methods("POST")
	estudianteRoutes.HandleFunc("/matricula/agregar-materia", matriculaHandler.AgregarMateriaModificaciones).Methods("POST")

	// Solicitudes de modificación
	estudianteRoutes.HandleFunc("/matricula/solicitudes-modificacion", matriculaHandler.GetSolicitudesEstudiante).Methods("GET")
	estudianteRoutes.HandleFunc("/matricula/solicitudes-modificacion", matriculaHandler.CrearSolicitudModificacion).Methods("POST")
	jefeRoutes.HandleFunc("/jefe/solicitudes-modificacion", matriculaHandler.GetSolicitudesPorPrograma).Methods("GET")
	jefeRoutes.HandleFunc("/jefe/solicitudes-modificacion/{id}/vista-previa", matriculaHandler.GetSolicitudVistaPrevia).Methods("GET")
	jefeRoutes.HandleFunc("/jefe/solicitudes-modificacion/{id}", matriculaHandler.ValidarSolicitudModificacion).Methods("PUT")
	protected.HandleFunc("/matricula/modificaciones/stream", matriculaHandler.StreamModificacionesEvents).Methods("GET")

	// Archivos estáticos protegidos por JWT.
	r.PathPrefix("/uploads/").Handler(
		middleware.JWTAuthMiddleware(cfg.JWTSecret)(
			http.StripPrefix("/uploads/", http.FileServer(http.Dir("./uploads/"))),
		),
	)

	// ── 7. Middlewares globales ───────────────────────────────────────────────

	// corsHandler aplica las cabeceras CORS y registra cada petición en el log.
	// El origen permitido viene de la variable de entorno CORS_ORIGIN (ver config.go).
	// En producción debe ser el dominio exacto del frontend, no "*".
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("📥 %s %s", r.Method, r.URL.Path)

			w.Header().Set("Access-Control-Allow-Origin", cfg.CORSOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}

	// ── 8. Arrancar servidor ──────────────────────────────────────────────────
	log.Printf("🚀 Servidor iniciado en el puerto %s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, corsHandler(r)))
}
