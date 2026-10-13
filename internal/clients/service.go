package clients

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetAll(
	ctx context.Context,
	companyID uuid.UUID,
) ([]Client, error) {

	return s.repository.GetAll(ctx, companyID)
}
