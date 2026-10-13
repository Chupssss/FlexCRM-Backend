package auth

import (
	"FlexCRM-Backend/internal/token"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"time"

	"github.com/google/uuid"
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

// Функция регистрации Админа и компании
func (serv *Service) Register(ctx context.Context, user *admin_company) (*AuthResult, error) {
	if len(user.Password) < 8 || len(user.Password) > 72 {
		log.Printf("%v", ErrInvalidPassword)
		return nil, ErrInvalidPassword
	}
	// создание юзера и компании
	password_hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	role := "ADMIN"
	user_id, company_id, err := serv.repo.CreateUserCompany(ctx, &CreateAdminCompanyParams{
		Company_name: user.Company_name,
		Email:        user.Email,
		PasswordHash: string(password_hash),
		First_name:   user.First_name,
		Last_name:    user.Last_name,
		Role:         role,
	})
	if err != nil {
		return nil, err
	}
	res, err := serv.CreateTokens(ctx, user_id, company_id, role)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Функция Логина пользователя
func (serv *Service) Login(ctx context.Context, us *user) (*AuthResult, error) {
	user_data, err := serv.repo.GetUserByEmail(ctx, us.Email)
	if err != nil {
		return nil, err
	}
	// Проверка пароля
	err = bcrypt.CompareHashAndPassword(user_data.Passwordhash, []byte(us.Password))
	if err != nil {
		log.Printf("Password doesn`t compare:%v", err)
		return nil, ErrDifferentPass
	}
	// Проверка активирован ли пользователь
	if !user_data.IsActive {
		return nil, ErrUserDeactive
	}
	res, err := serv.CreateTokens(ctx, user_data.User_id, user_data.Company_id, user_data.Role)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Функция по созданию refresh и access токенов
func (serv *Service) CreateTokens(ctx context.Context, user_id, company_id uuid.UUID, role string) (*AuthResult, error) {
	// создание refresh токена
	refresh_token := rand.Text()
	refresh_expires := time.Now().AddDate(0, 0, 30)
	hash_refresh_token := sha256.Sum256([]byte(refresh_token))
	err := serv.repo.CreateRefreshToken(ctx, user_id, hash_refresh_token[:], refresh_expires)
	if err != nil {
		return nil, err
	}
	// создание access-токена
	access_expires := time.Now().Add(15 * time.Minute)
	access_token, err := serv.tokenManager.CreateToken(user_id, company_id, role, access_expires) //создание access токена длительностью 15 минут
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

// Обработка Refresh token
func (serv *Service) RefreshTokens(ctx context.Context, refresh_token string) (*AuthResult, error) {
	hash_refresh_token := sha256.Sum256([]byte(refresh_token)) // хэшируем для бд
	user_id, exp_at, err := serv.repo.GetRefreshDataByRefreshtoken(ctx, hash_refresh_token[:])
	if err != nil {
		return nil, err
	}
	if time.Now().After(exp_at) {
		return nil, ErrRefreshTokenExpired
	}
	err = serv.repo.DeleteRefreshByRefreshtoken(ctx, hash_refresh_token[:])
	if err != nil {
		return nil, err
	}
	company_id, role, err := serv.repo.GetUserByid(ctx, user_id)
	if err != nil {
		return nil, err
	}
	res, err := serv.CreateTokens(ctx, user_id, company_id, role)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Функция LogOut пользователя(удаление refresh-token)
func (serv *Service) LogOut(ctx context.Context, refresh_token string) error {
	hash_refresh_token := sha256.Sum256([]byte(refresh_token)) // хэшируем для бд
	err := serv.repo.DeleteRefreshByRefreshtoken(ctx, hash_refresh_token[:])
	if err != nil {
		return err
	}
	return nil
}

// Функция регистрации работника и менеджера
func (serv *Service) RegisterUser(ctx context.Context, user_data *ManagerOrEmployee, copmany_id uuid.UUID) (uuid.UUID, error) {
	if user_data.Role != "MANAGER" && user_data.Role != "EMPLOYEE" {
		log.Printf("%v", ErrInvalidRole)
		return uuid.Nil, ErrInvalidRole
	}
	if len(user_data.Password) < 8 || len(user_data.Password) > 72 {
		log.Printf("%v", ErrInvalidPassword)
		return uuid.Nil, ErrInvalidPassword
	}
	password_hash, err := bcrypt.GenerateFromPassword([]byte(user_data.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, err
	}

	user_id, err := serv.repo.CreateUser(ctx, &UserData{
		Company_id:   copmany_id,
		Email:        user_data.Email,
		PasswordHash: string(password_hash),
		First_name:   user_data.First_name,
		Last_name:    user_data.Last_name,
		Role:         user_data.Role,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return user_id, nil
}
