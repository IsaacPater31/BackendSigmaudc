package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

// Tipo específico para la clave del contexto (mejor práctica que usar string)
type contextKey string

const ClaimsContextKey contextKey = "jwt_claims"

func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			tokenString := ""
			if authHeader != "" {
				// Extraer el token del header "Bearer <token>"
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != "Bearer" {
					http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
					return
				}
				tokenString = parts[1]
			} else {
				// Fallback para canales SSE/EventSource (no soporta header Authorization nativo)
				tokenString = strings.TrimSpace(r.URL.Query().Get("token"))
				if tokenString == "" {
					http.Error(w, "Authorization header required", http.StatusUnauthorized)
					return
				}
			}

			// Parsear y validar el token
			claims := &models.JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				// Verificar el método de firma
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Validar que los claims no estén vacíos
			if claims.Sub == 0 {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			// Agregar los claims al contexto usando tipo específico
			ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaimsFromContext es una función helper para obtener los claims del contexto
func GetClaimsFromContext(ctx context.Context) (*models.JWTClaims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*models.JWTClaims)
	if !ok || claims == nil || claims.Sub == 0 {
		return nil, false
	}
	return claims, ok
}
