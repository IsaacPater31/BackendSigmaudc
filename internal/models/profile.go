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
