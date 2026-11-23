package general

import (
	"context"
	"errors"
	"github.com/google/uuid"
)

var ErrForbidden = errors.New("пользователь не имеет доступа к данным компании")

// Repository позволяет проверять HTTP-контракт без подключения к БД.
type Repository interface {
	Get(context.Context, uuid.UUID, uuid.UUID, int) (*Response, error)
}

type Service struct{ repository Repository }

func NewService(repository Repository) *Service { return &Service{repository: repository} }

func (s *Service) Get(ctx context.Context, userID, companyID uuid.UUID, limit int) (*Response, error) {
	if userID == uuid.Nil || companyID == uuid.Nil {
		return nil, ErrForbidden
	}
	return s.repository.Get(ctx, userID, companyID, limit)
}
