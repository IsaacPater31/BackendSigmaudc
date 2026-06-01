// Package database provee la función de conexión a PostgreSQL y
// configura el pool de conexiones para uso eficiente en producción.
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Connect abre y valida una conexión al servidor PostgreSQL especificado
// por databaseURL, configura el pool de conexiones y retorna el objeto *sql.DB.
//
// Configuración del pool:
//   - Máximo 25 conexiones abiertas simultáneamente.
//   - Máximo 5 conexiones mantenidas inactivas en el pool.
//   - Tiempo de vida máximo de una conexión: 5 minutos.
func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("error al abrir la base de datos: %w", err)
	}

	// Configurar pool de conexiones para mejorar rendimiento y evitar
	// agotamiento de recursos bajo carga concurrente.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verificar que la conexión es válida antes de continuar.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error al verificar conexión con la base de datos: %w", err)
	}

	return db, nil
}
