package users

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	companyID uuid.UUID,
	req CreateUserRequest,
) (*User, error) {

	// Проверка данных
	if companyID == uuid.Nil ||
		strings.TrimSpace(req.FirstName) == "" ||
		strings.TrimSpace(req.Email) == "" ||
		strings.TrimSpace(req.Password) == "" {
		return nil, ErrInvalidUserData
	}

	// Проверка роли
	if req.Role != "MANAGER" && req.Role != "EMPLOYEE" {
		return nil, ErrInvalidUserData
	}

	// Хеширование пароля
	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// Формирование пользователя
	user := User{
		ID:            uuid.New(),
		CompanyID:     companyID,
		Email:         strings.TrimSpace(req.Email),
		Password_Hash: string(passwordHash),
		First_Name:    strings.TrimSpace(req.FirstName),
		Last_Name:     &req.LastName,
		Role:          req.Role,
		Is_active:     true,
		Created_at:    now,
		Updated_at:    now,
	}

	return s.repo.Create(ctx, user)
}

func (s *Service) Update(
	ctx context.Context,
	userID uuid.UUID,
	companyID uuid.UUID,
	req UpdateUserRequest,
) (*User, error) {

	if req.FirstName != nil &&
		strings.TrimSpace(*req.FirstName) == "" {
		return nil, ErrInvalidUserData
	}

	if req.Role != nil {
		switch *req.Role {
		case "MANAGER", "EMPLOYEE":
		default:
			return nil, ErrInvalidUserData
		}
	}

	// Получаем пользователя только из текущей компании
	existingUser, err := s.repo.GetById(ctx, userID, companyID)
	if err != nil {
		return nil, err
	}

	// Запрещаем понижать или деактивировать ADMIN
	if existingUser.Role == "ADMIN" {
		if req.Role != nil && *req.Role != "ADMIN" {
			return nil, ErrAdminProtected
		}

		if req.IsActive != nil && !*req.IsActive {
			return nil, ErrAdminProtected
		}
	}

	return s.repo.Update(ctx, userID, companyID, req)
}

func (s *Service) GetAll(
	ctx context.Context,
	companyID uuid.UUID,
) ([]User, error) {

	if companyID == uuid.Nil {
		return nil, ErrInvalidUserData
	}

	return s.repo.GetAll(ctx, companyID)
}
func (s *Service) GetByID(
	ctx context.Context,
	userID uuid.UUID,
	companyID uuid.UUID,
) (*User, error) {

	if userID == uuid.Nil || companyID == uuid.Nil {
		return nil, ErrInvalidUserData
	}

	return s.repo.GetById(ctx, userID, companyID)
}
