// Package config gestiona la configuración de la aplicación cargada
// desde variables de entorno. Es el único punto donde se leen variables
// de entorno, evitando accesos dispersos a os.Getenv en toda la app.
//
// Principios aplicados:
//   - SRP: un solo lugar para la configuración.
//   - DRY: los valores por defecto están centralizados aquí.
package config

import (
	"os"
)

// Config agrupa todos los parámetros de configuración de la aplicación.
// Se carga una sola vez al arrancar y se inyecta en los componentes que la necesitan.
type Config struct {
	// DatabaseURL es la cadena de conexión completa a PostgreSQL.
	DatabaseURL string

	// JWTSecret es la clave secreta para firmar y verificar tokens JWT.
	JWTSecret string

	// Port es el puerto en el que escucha el servidor HTTP.
	Port string

	// CORSOrigin es el origen permitido en las cabeceras CORS.
	// Usar "*" permite cualquier origen (solo recomendable en desarrollo).
	CORSOrigin string

	// UploadDir es el directorio base donde se almacenan los archivos subidos.
	UploadDir string
}

// Load lee las variables de entorno y retorna una Config completamente inicializada.
// Hace panic si alguna variable crítica (DATABASE_URL, JWT_SECRET) no está configurada.
func Load() *Config {
	databaseURL := getEnv("DATABASE_URL", "")
	jwtSecret := getEnv("JWT_SECRET", "")

	// Validar que las variables críticas estén configuradas antes de arrancar
	if databaseURL == "" {
		panic("DATABASE_URL no está configurada en el archivo .env")
	}
	if jwtSecret == "" {
		panic("JWT_SECRET no está configurada en el archivo .env")
	}

	return &Config{
		DatabaseURL: databaseURL,
		JWTSecret:   jwtSecret,
		Port:        getEnv("PORT", "8080"),
		CORSOrigin:  getEnv("CORS_ORIGIN", "*"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
	}
}

// getEnv retorna el valor de la variable de entorno key,
// o defaultValue si no está definida o está vacía.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
