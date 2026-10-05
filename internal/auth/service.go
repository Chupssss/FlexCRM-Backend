package auth

import (
	"FlexCRM-Backend/internal/token"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Структура сервиса содержащая структуру с подключением к бд
type Service struct {
	repo         *RepoConn
	tokenManager *token.TokenManager
}

// Функция создания структуры сервиса с сохранением подключения к бд
func NewService(repo *RepoConn, tokenManager *token.TokenManager) *Service {
	return &Service{
		repo:         repo,
		tokenManager: tokenManager,
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
	refresh_token := rand.Text()
	refresh_expires := time.Now().AddDate(0, 0, 30)
	hash_refresh_token := sha256.Sum256([]byte(refresh_token))
	err = serv.repo.CreateRefreshToken(ctx, id, hash_refresh_token[:], refresh_expires)
	if err != nil {
		return nil, err
	}
	// создание access-токена
	access_expires := time.Now().Add(15 * time.Minute)
	access_token, err := serv.tokenManager.CreateToken(id, access_expires) //создание access токена длительностью 15 минут
	if err != nil {
		return nil, err
	}
	res := AuthResult{
		AccessToken:      access_token,
		RefreshToken:     refresh_token,
		AccessExpiresAt:  access_expires,
		RefreshExpiresAt: refresh_expires,
	}
	return &res, nil
}
