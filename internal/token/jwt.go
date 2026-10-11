package token

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Структура с JWT секретом
type TokenManager struct {
	jwtSecret string
}

// Функция создания структуры TokenManager
func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		jwtSecret: secret,
	}
}

// Специальные поля для payload JWT
type CustomClaims struct {
	User_id    uuid.UUID `json:"user_id"`
	Role       string    `json:"role"`
	Company_id uuid.UUID `json:"company_id"`
	jwt.RegisteredClaims
}

// Функция создания jwt токена
func (manager *TokenManager) CreateToken(user_id, company_id uuid.UUID, role string, expires_at time.Time) (string, error) {
	claims := CustomClaims{
		User_id:    user_id,
		Role:       role,
		Company_id: company_id,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expires_at),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedtoken, err := token.SignedString([]byte(manager.jwtSecret))
	if err != nil {
		log.Printf("%v", err)
		return "", err
	}
	return signedtoken, nil
}

// Функция верификации jwt токена
func (manager *TokenManager) VerifyToken(tokenStr string) (*CustomClaims, error) {
	claims := &CustomClaims{}
	token, err := jwt.NewParser().ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(manager.jwtSecret), nil
	})
	if err != nil {
		log.Printf("%v", err)
		return nil, err
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid error")
	}
	return claims, nil
}
