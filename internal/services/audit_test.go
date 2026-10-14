// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package services

import "testing"

// TestParseAuditLimit verifica que el query param limit se sanea a un
// entero dentro de rango antes de llegar al SQL.
func TestParseAuditLimit(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"vacio usa default", "", 50},
		{"valor válido", "100", 100},
		{"no numérico usa default", "abc", 50},
		{"negativo usa default", "-5", 50},
		{"cero usa default", "0", 50},
		{"con espacios", " 25 ", 25},
		{"supera el máximo", "99999", maxAuditLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseAuditLimit(tt.input); got != tt.want {
				t.Errorf("parseAuditLimit(%q) = %d, se esperaba %d", tt.input, got, tt.want)
			}
		})
	}
}
