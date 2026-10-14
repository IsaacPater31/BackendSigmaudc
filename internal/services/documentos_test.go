// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package services

import (
	"strings"
	"testing"
)

// TestSanitizePathComponent verifica que los nombres de archivo provistos
// por el cliente no puedan escapar del directorio de destino.
func TestSanitizePathComponent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		mustNot []string
	}{
		{"nombre normal", "documento final", "documento_final", nil},
		{"traversal clásico", "../../etc/passwd", "etcpasswd", []string{"..", "/"}},
		{"traversal con backslash", "..\\..\\win", "win", []string{"..", "\\"}},
		{"solo puntos", "...", "archivo", nil},
		{"vacío", "", "archivo", nil},
		{"caracteres especiales", "mi archivo (1)!.pdf", "mi_archivo_1.pdf", []string{"(", ")", "!"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizePathComponent(tt.input)
			if got != tt.want {
				t.Errorf("sanitizePathComponent(%q) = %q, se esperaba %q", tt.input, got, tt.want)
			}
			for _, bad := range tt.mustNot {
				if strings.Contains(got, bad) {
					t.Errorf("sanitizePathComponent(%q) = %q contiene %q prohibido", tt.input, got, bad)
				}
			}
		})
	}
}
