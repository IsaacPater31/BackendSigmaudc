// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package middleware

import (
	"github.com/andrxsq/SIGMAUDC/internal/httpx"
	"net/http"
)

// RequireRoles restringe el acceso a usuarios cuyo claim Rol coincida
// con alguno de los roles permitidos.
func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetClaimsFromContext(r.Context())
			if !ok {
				httpx.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if _, exists := allowed[claims.Rol]; !exists {
				httpx.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
