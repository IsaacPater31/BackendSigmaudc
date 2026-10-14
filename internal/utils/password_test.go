// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package utils

import "testing"

// TestValidatePassword cubre las reglas de la política de contraseñas:
// mínimo 8 caracteres, al menos una letra, al menos un número y
// caracteres estrictamente alfanuméricos.
func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantOK   bool
	}{
		{"vacia", "", false},
		{"menor a 8 caracteres", "abc1", false},
		{"exactamente 8 valida", "abcd1234", true},
		{"sin numeros", "abcdefgh", false},
		{"sin letras", "12345678", false},
		{"con simbolos", "abcd123!", false},
		{"con espacios", "abcd 1234", false},
		{"con tilde", "ábcd1234", false},
		{"larga valida", "Matricula2025Segura", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, msg := ValidatePassword(tt.password)
			if ok != tt.wantOK {
				t.Errorf("ValidatePassword(%q) = %v, se esperaba %v (mensaje: %q)",
					tt.password, ok, tt.wantOK, msg)
			}
			if !ok && msg == "" {
				t.Errorf("ValidatePassword(%q) falló sin mensaje de error", tt.password)
			}
		})
	}
}
