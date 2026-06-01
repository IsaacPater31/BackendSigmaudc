package services

import (
	"github.com/andrxsq/SIGMAUDC/internal/constants"
	"github.com/andrxsq/SIGMAUDC/internal/models"
	"github.com/andrxsq/SIGMAUDC/internal/repositories"
)

type AuditService struct {
	repo *repositories.AuditRepository
}

func NewAuditService(repo *repositories.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) GetAuditLogs(limit string) ([]models.AuditLog, error) {
	if limit == "" {
		limit = constants.DefaultAuditLimit
	}
	return s.repo.GetAuditLogs(limit)
}
