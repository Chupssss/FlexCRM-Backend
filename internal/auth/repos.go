package auth

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	PasswordHash []byte
	First_name   string
	Last_name    string
	Role         string
}

// Функция создания пользователя и компании в БД
func (repo *RepoConn) CreateUserCompany(ctx context.Context, user *CreateUserParameters) (uuid.UUID, uuid.UUID, error) {
	tx, err := repo.conn.Begin(ctx) // начало транзакции
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, uuid.Nil, err
	}
	defer tx.Rollback(ctx)
	var company_id uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO companies (name, created_at, updated_at) VALUES ($1, $2, $3) RETURNING id", user.Company_name, time.Now(), time.Now()).Scan(&company_id)
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, uuid.Nil, err
	}
	var user_id uuid.UUID
	err = tx.QueryRow(ctx, "INSERT INTO users (company_id, email, password_hash, first_name, last_name, role, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id", company_id, user.Email, user.PasswordHash, user.First_name, user.Last_name, user.Role, time.Now(), time.Now()).Scan(&user_id)
	if err != nil {
		var pgerr *pgconn.PgError
		if errors.As(err, &pgerr) {
			if pgerr.Code == "23505" {
				return uuid.Nil, uuid.Nil, ErrEmailAlreadyExists
			}
		}
		log.Printf("%v", err)
		return uuid.Nil, uuid.Nil, err
	}
	err = tx.Commit(ctx) // заканчиваем транзакцию
	if err != nil {
		log.Printf("%v", err)
		return uuid.Nil, uuid.Nil, err
	}
	return user_id, company_id, nil
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

// структура с данными пользователя
type User_data struct {
	User_id      uuid.UUID
	Company_id   uuid.UUID
	Role         string
	IsActive     bool
	Passwordhash []byte
}

// Функция получения данных пользователя по его email
func (repo *RepoConn) GetUserByEmail(ctx context.Context, email string) (*User_data, error) {
	var user User_data
	err := repo.conn.QueryRow(ctx, "SELECT id, company_id, role, is_active, password_hash FROM users WHERE email = $1", email).Scan(&user.User_id,
		&user.Company_id,
		&user.Role,
		&user.IsActive,
		&user.Passwordhash,
	)
	if err != nil {
		log.Printf("no row where emails are similar: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

// Функция получения id_company и role пользователя по его email
func (repo *RepoConn) GetUserByid(ctx context.Context, id uuid.UUID) (uuid.UUID, string, error) {
	var company_id uuid.UUID
	var role string
	err := repo.conn.QueryRow(ctx, "SELECT company_id, role FROM users WHERE id = $1", id).Scan(&company_id, &role)
	if err != nil {
		log.Printf("no row where user_id are similar: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, "", ErrUserNotFound
		}
		return uuid.Nil, "", err
	}
	return company_id, role, nil
}

// Функция получения данных refresh токена по данному refresh токену
func (repo *RepoConn) GetRefreshDataByRefreshtoken(ctx context.Context, hash_refresh_token []byte) (uuid.UUID, time.Time, error) {
	var user_id uuid.UUID
	var expires_at time.Time
	err := repo.conn.QueryRow(ctx, "SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1", hash_refresh_token).Scan(&user_id, &expires_at)
	if err != nil {
		log.Printf("no row where token_hash are similar: %v", err)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, time.Time{}, ErrRefreshTokenNotFound
		}
		return uuid.Nil, time.Time{}, err
	}
	return user_id, expires_at, nil
}

// Функция удаления Refresh token
func (repo *RepoConn) DeleteRefreshByRefreshtoken(ctx context.Context, hash_refresh_token []byte) error {
	tag, err := repo.conn.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", hash_refresh_token)
	if err != nil {
		log.Printf("error on delete refresh-token: %v", err)
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Printf("no row where token_hash are similar: %v", err)
		return ErrRefreshTokenNotFound
	}
	return nil
}
