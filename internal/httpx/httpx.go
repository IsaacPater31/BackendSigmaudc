// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey,
// Isaac David Paternina Noble, Harry Perez Perea, Camilo Rodríguez Garay,
// José Félix Bustamante Soto y Juan Felipe Eraso Navarro.
// Todos los derechos reservados.

// Package httpx centraliza la escritura de respuestas HTTP en formato JSON.
//
// Principios aplicados:
//   - DRY: un solo punto de serialización de respuestas y errores.
//   - Consistencia: toda la API responde errores con el mismo contrato
//     {"error": "..."} en application/json, nunca texto plano.
package httpx

import (
	"encoding/json"
	"net/http"
)

// ErrorBody es el cuerpo estándar de respuesta de error de la API.
type ErrorBody struct {
	Error string `json:"error"`
}

// JSON serializa v como application/json y la escribe con el código status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error escribe una respuesta de error JSON con el código y mensaje dados.
// Sustituye a http.Error para que la API nunca responda texto plano.
func Error(w http.ResponseWriter, message string, status int) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	JSON(w, status, ErrorBody{Error: message})
}
