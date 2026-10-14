// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey
// e Isaac David Paternina Noble. Todos los derechos reservados.

package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONWritesBodyAndContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	JSON(rec, http.StatusCreated, map[string]string{"estado": "ok"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("código esperado %d, obtenido %d", http.StatusCreated, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type esperado application/json, obtenido %q", ct)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if body["estado"] != "ok" {
		t.Fatalf("cuerpo inesperado: %v", body)
	}
}

func TestErrorWritesStandardContract(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, "credenciales inválidas", http.StatusUnauthorized)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("código esperado %d, obtenido %d", http.StatusUnauthorized, rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type esperado application/json, obtenido %q", ct)
	}
	if nosniff := rec.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Fatalf("X-Content-Type-Options esperado nosniff, obtenido %q", nosniff)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("respuesta no es JSON válido: %v", err)
	}
	if body.Error != "credenciales inválidas" {
		t.Fatalf("mensaje inesperado: %q", body.Error)
	}
}
