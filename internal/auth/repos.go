package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Структура содержащая подключечение к бд
type RepoConn struct {
	conn *pgxpool.Pool
}

// Функция сохранения подключения к бд в структуре
func SaveRepoConn(conn *pgxpool.Pool) *RepoConn {
	return &RepoConn{
		conn: conn,
	}
}

type CreateUserParameters struct {
	Company_name string
	Email        string
	PasswordHash string
	First_name   string
	Last_name    string
	Role         string
}

// Функция создания пользователя и компании в БД
func (repo *RepoConn) CreateUserCompany(ctx context.Context, user *CreateUserParameters) (uuid.UUID, error) {
	tx, err := repo.conn.Begin(ctx) // начало транзакции
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var company_id uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO companies (name, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id", user.Company_name, time.Now(), time.Now()).Scan(&company_id)
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, err
	}
	var user_id uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO users (company_id, email, password_hash, first_name, last_name, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id", company_id, user.Email, user.PasswordHash, user.First_name, user.Last_name, user.Role, time.Now(), time.Now()).Scan(&user_id)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == "23505" {
				return uuid.Nil, ErrEmailAlreadyExists
			}
		}
		log.Printf("%v", err)
		return uuid.Nil, err
	}
	err = tx.Commit(ctx) // заканчиваем транзакцию
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, err
	}
	return user_id, nil
}

// Функция создания refresh-token
func (repo *RepoConn) CreateRefreshToken(ctx context.Context, id uuid.UUID, hash_refresh_token []byte, refresh_expires_time time.Time) error {
	_, err := repo.conn.Exec(ctx, "INSERT INTO refresh_tokens (user_id, token_hash, expires_at, created_at) VALUES ($1, $2, $3, $4)", id, hash_refresh_token, refresh_expires_time, time.Now())
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	if err != nil {
		log.Printf("%v", err)
		return err
	}
	return nil
}
