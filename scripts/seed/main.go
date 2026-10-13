package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Seed struct {
	Company Company  `json:"company"`
	Users   []User   `json:"users"`
	Clients []Client `json:"clients"`
}

type Company struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"passwordHash"`
	FirstName    string `json:"firstName"`
	LastName     string `json:"lastName"`
	Role         string `json:"role"`
}

type Client struct {
	ID                string  `json:"id"`
	ResponsibleUserID *string `json:"responsibleUserId"`
	CreatedBy         string  `json:"createdBy"`
	ClientType        string  `json:"clientType"`

	FirstName   *string `json:"firstName"`
	LastName    *string `json:"lastName"`
	MiddleName  *string `json:"middleName"`
	CompanyName *string `json:"companyName"`

	Email       *string `json:"email"`
	Phone       *string `json:"phone"`
	Description *string `json:"description"`
}

func main() {
	// Загружаем .env из корня проекта.
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is empty")
	}

	// Читаем JSON.
	data, err := os.ReadFile("scripts/seed/seed.json")
	if err != nil {
		log.Fatal("cannot read seed.json: ", err)
	}

	var seed Seed

	if err := json.Unmarshal(data, &seed); err != nil {
		log.Fatal("cannot parse seed.json: ", err)
	}

	ctx := context.Background()

	// Подключаемся к PostgreSQL.
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("cannot create database pool: ", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		log.Fatal("cannot connect to database: ", err)
	}

	fmt.Println("database connected")

	// Всё выполняем в одной транзакции.
	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatal("cannot begin transaction: ", err)
	}

	defer tx.Rollback(ctx)

	now := time.Now()

	// -------------------------
	// COMPANY
	// -------------------------

	_, err = tx.Exec(
		ctx,
		`
		INSERT INTO companies (
			id,
			name,
			description,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
		`,
		seed.Company.ID,
		seed.Company.Name,
		seed.Company.Description,
		now,
		now,
	)

	if err != nil {
		log.Fatal("cannot insert company: ", err)
	}

	fmt.Println("company inserted")

	// -------------------------
	// USERS
	// -------------------------

	for _, user := range seed.Users {
		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO users (
				id,
				company_id,
				email,
				password_hash,
				first_name,
				last_name,
				role,
				is_active,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, TRUE, $8, $9
			)
			ON CONFLICT (id) DO UPDATE SET
				email = EXCLUDED.email,
				password_hash = EXCLUDED.password_hash,
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				role = EXCLUDED.role,
				updated_at = EXCLUDED.updated_at
			`,
			user.ID,
			seed.Company.ID,
			user.Email,
			user.PasswordHash,
			user.FirstName,
			user.LastName,
			user.Role,
			now,
			now,
		)

		if err != nil {
			log.Fatalf(
				"cannot insert user %s: %v",
				user.Email,
				err,
			)
		}
	}

	fmt.Printf("%d users inserted\n", len(seed.Users))

	// -------------------------
	// CLIENTS
	// -------------------------

	for _, client := range seed.Clients {
		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO clients (
				id,
				company_id,
				responsible_user_id,
				created_by,
				client_type,
				first_name,
				last_name,
				middle_name,
				company_name,
				email,
				phone,
				description,
				created_at,
				updated_at
			)
			VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, $12, $13, $14
			)
			ON CONFLICT (id) DO UPDATE SET
				responsible_user_id = EXCLUDED.responsible_user_id,
				client_type = EXCLUDED.client_type,
				first_name = EXCLUDED.first_name,
				last_name = EXCLUDED.last_name,
				middle_name = EXCLUDED.middle_name,
				company_name = EXCLUDED.company_name,
				email = EXCLUDED.email,
				phone = EXCLUDED.phone,
				description = EXCLUDED.description,
				updated_at = EXCLUDED.updated_at
			`,
			client.ID,
			seed.Company.ID,
			client.ResponsibleUserID,
			client.CreatedBy,
			client.ClientType,
			client.FirstName,
			client.LastName,
			client.MiddleName,
			client.CompanyName,
			client.Email,
			client.Phone,
			client.Description,
			now,
			now,
		)

		if err != nil {
			log.Fatalf(
				"cannot insert client %s: %v",
				client.ID,
				err,
			)
		}
	}

	fmt.Printf("%d clients inserted\n", len(seed.Clients))

	// Сохраняем транзакцию.
	if err := tx.Commit(ctx); err != nil {
		log.Fatal("cannot commit transaction: ", err)
	}

	fmt.Println("seed completed successfully")
}
