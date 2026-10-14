// SIGMA-UDC — Sistema Integral de Gestión de Matrícula Académica
// Copyright (c) 2025 Andrés Javier Quintana Alape, Gustavo Calderón Kelsey,
// Isaac David Paternina Noble, Harry Perez Perea, Camilo Rodríguez Garay,
// José Félix Bustamante Soto y Juan Felipe Eraso Navarro.
// Todos los derechos reservados.

package models

type EstudianteDatosResponse struct {
	EstudianteID int      `json:"estudiante_id"`
	Codigo       string   `json:"codigo"`
	Nombre       string   `json:"nombre"`
	Apellido     string   `json:"apellido"`
	Email        string   `json:"email"`
	Programa     string   `json:"programa"`
	Semestre     int      `json:"semestre"`
	Promedio     *float64 `json:"promedio,omitempty"`
	Estado       string   `json:"estado"`
	Sexo         string   `json:"sexo"`
	FotoPerfil   string   `json:"foto_perfil"`
}

type UpdateDatosRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	Sexo     string `json:"sexo"`
}

type JefeDatosResponse struct {
	JefeID     int    `json:"jefe_id"`
	Codigo     string `json:"codigo"`
	Nombre     string `json:"nombre"`
	Apellido   string `json:"apellido"`
	Email      string `json:"email"`
	Programa   string `json:"programa"`
	Sexo       string `json:"sexo"`
	FotoPerfil string `json:"foto_perfil"`
}
