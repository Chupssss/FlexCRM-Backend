package pipelines

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}
func (s *Service) Get_Option(ctx context.Context, companyID uuid.UUID) ([]PipelineOption, error) {
	return s.repo.Get_Option(ctx, companyID)
}
