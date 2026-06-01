// Package handlers implementa todos los controladores HTTP de la aplicación.
// Cada handler es responsable de orquestar una petición HTTP:
// validar el input, delegar a servicios y serializar la respuesta.
package handlers

import (
	"errors"
	"net/http"

	"github.com/andrxsq/SIGMAUDC/internal/middleware"
	"github.com/andrxsq/SIGMAUDC/internal/models"
)

// getClaims extrae y valida los claims JWT del contexto de la petición HTTP.
//
// Es una función de paquete (Pure Fabrication, GRASP) compartida por todos
// los handlers para eliminar la duplicación del mismo bloque en cada uno.
//
// Retorna los claims si el token es válido, o un error "unauthorized"
// en caso contrario.
func getClaims(r *http.Request) (*models.JWTClaims, error) {
	claims, ok := middleware.GetClaimsFromContext(r.Context())
	if !ok {
		return nil, errors.New("unauthorized")
	}
	return claims, nil
}
