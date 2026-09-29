package users

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

func (s *Service) Create(ctx context.Context, user User) (*User, error) {
	created_user, err := s.repo.Create(ctx, user)
	if err != nil {
		return nil, err
	}
	return created_user, nil

}

func (s *Service) GetALL(ctx context.Context, companyID uuid.UUID) ([]User, error) {
	users, err := s.repo.GetAll(ctx, companyID)
	if err != nil {
		return nil, err
	}
	return users, nil

}
func (s *Service) GetById(ctx context.Context, id uuid.UUID, companyID uuid.UUID) (*User, error) {
	user, err := s.repo.GetById(ctx, id, companyID)
	if err != nil {
		return nil, err
	}
	return user, nil

}
