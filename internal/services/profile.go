package services

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

var (
	ErrEstudianteNoEncontrado = errors.New("estudiante no encontrado")
	ErrJefeNoEncontrado       = errors.New("jefe no encontrado")
	ErrSexoInvalido           = errors.New("sexo invalido")
	ErrFormatoImagenInvalido  = errors.New("formato imagen invalido")
)

type ProfileService struct {
	repo *repositories.ProfileRepository
}

func NewProfileService(repo *repositories.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

func (s *ProfileService) GetDatosEstudiante(usuarioID int) (*models.EstudianteDatosResponse, error) {
	datos, promedio, err := s.repo.GetDatosEstudiante(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEstudianteNoEncontrado
	}
	if err != nil {
		return nil, err
	}
	if promedio.Valid {
		datos.Promedio = &promedio.Float64
	}
	return datos, nil
}

func (s *ProfileService) UpdateDatosEstudiante(usuarioID int, req models.UpdateDatosRequest) error {
	sexo, err := sanitizeSexo(req.Sexo)
	if err != nil {
		return err
	}
	estudianteID, err := s.repo.GetEstudianteID(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEstudianteNoEncontrado
	}
	if err != nil {
		return err
	}
	return s.repo.UpdateEstudianteDatos(estudianteID, req, sexo)
}

func (s *ProfileService) UploadEstudianteFoto(usuarioID int, file multipart.File, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !isAllowedExt(ext, constants.ExtensionesFoto) {
		return "", ErrFormatoImagenInvalido
	}
	estudianteID, err := s.repo.GetEstudianteID(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrEstudianteNoEncontrado
	}
	if err != nil {
		return "", err
	}

	dir := filepath.Join("uploads", "profiles", fmt.Sprintf("%d", estudianteID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	destPath := filepath.Join(dir, fmt.Sprintf("profile%s", ext))
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	photoURL := fmt.Sprintf("/uploads/profiles/%d/profile%s", estudianteID, ext)
	if err := s.repo.UpdateEstudianteFoto(estudianteID, photoURL); err != nil {
		return "", err
	}
	return photoURL, nil
}

func (s *ProfileService) GetDatosJefe(usuarioID int) (*models.JefeDatosResponse, error) {
	datos, err := s.repo.GetDatosJefe(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrJefeNoEncontrado
	}
	return datos, err
}

func (s *ProfileService) UpdateDatosJefe(usuarioID int, req models.UpdateDatosRequest) error {
	sexo, err := sanitizeSexo(req.Sexo)
	if err != nil {
		return err
	}
	jefeID, err := s.repo.GetJefeID(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrJefeNoEncontrado
	}
	if err != nil {
		return err
	}
	return s.repo.UpdateJefeDatos(jefeID, req, sexo)
}

func (s *ProfileService) UploadJefeFoto(usuarioID int, file multipart.File, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	if !isAllowedExt(ext, constants.ExtensionesFoto) {
		return "", ErrFormatoImagenInvalido
	}
	jefeID, err := s.repo.GetJefeID(usuarioID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrJefeNoEncontrado
	}
	if err != nil {
		return "", err
	}
	dir := filepath.Join("uploads", "profiles", "jefes", fmt.Sprintf("%d", jefeID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	destPath := filepath.Join(dir, fmt.Sprintf("profile%s", ext))
	dst, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	photoURL := fmt.Sprintf("/uploads/profiles/jefes/%d/profile%s", jefeID, ext)
	if err := s.repo.UpdateJefeFoto(jefeID, photoURL); err != nil {
		return "", err
	}
	return photoURL, nil
}

func sanitizeSexo(sexo string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(sexo))
	if s == "" {
		s = "otro"
	}
	if _, ok := constants.SexosPermitidos[s]; !ok {
		return "", ErrSexoInvalido
	}
	return s, nil
}

func isAllowedExt(ext string, allowed []string) bool {
	for _, v := range allowed {
		if ext == v {
			return true
		}
	}
	return false
}
