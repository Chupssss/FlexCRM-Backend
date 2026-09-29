package auth

import (
	"context"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Структура сервиса содержащая структуру с подключением к бд
type Service struct {
	repo *RepoConn
}

// Функция создания структуры сервиса с сохранением подключения к бд
func NewService(repo *RepoConn) *Service {
	return &Service{
		repo: repo,
	}
}

// Структура возвращаемая при регистрации в хэндлер
type AuthResult struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

// Функция регистрации пользователя и компании
func (serv *Service) Register(ctx context.Context, user *admin_company) (*AuthResult, error) {
	// создание юзера и компании
	password_hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("%v", err)
		return nil, err
	}
	id, err := serv.repo.CreateUserCompany(ctx, &CreateUserParameters{
		Company_name: user.Company_name,
		Email:        user.Email,
		PasswordHash: password_hash,
		First_name:   user.First_name,
		Last_name:    user.Last_name,
		Role:         "ADMIN",
	})
	if err != nil {
		return nil, err
	}
	// создание refresh токена

	// создание accessтокена
}
