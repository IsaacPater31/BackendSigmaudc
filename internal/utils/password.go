package utils

import (
	"regexp"
	"unicode"
)

// ValidatePassword valida que la contraseña cumpla con los requisitos:
// - Mínimo 8 caracteres
// - Debe contener al menos una letra
// - Debe contener al menos un número
func ValidatePassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "La contraseña debe tener al menos 8 caracteres"
	}

	hasLetter := false
	hasNumber := false

	for _, char := range password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsNumber(char) {
			hasNumber = true
		}
	}

	if !hasLetter {
		return false, "La contraseña debe contener al menos una letra"
	}

	if !hasNumber {
		return false, "La contraseña debe contener al menos un número"
	}

	// Verificar que sea alfanumérica (solo letras y números)
	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumericRegex.MatchString(password) {
		return false, "La contraseña solo puede contener letras y números (alfanumérica)"
	}

	return true, ""
}

