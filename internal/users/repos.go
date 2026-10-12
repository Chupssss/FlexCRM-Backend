package users

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, user User) (*User, error) {
	query := `
	INSERT INTO users (id, company_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	RETURNING id
	`
	err := r.db.QueryRow(ctx, query, user.ID, user.CompanyID, user.Email, user.Password_Hash, user.First_Name, user.Last_Name, user.Role, user.Is_active, user.Created_at, user.Updated_at).Scan(&user.ID)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *Repository) Update(
	ctx context.Context,
	userID uuid.UUID,
	companyID uuid.UUID,
	req UpdateUserRequest,
) (*User, error) {

	query := `
		UPDATE users
		SET
			first_name = COALESCE($1, first_name),
			last_name = COALESCE($2, last_name),
			role = COALESCE($3, role),
			is_active = COALESCE($4, is_active),
			updated_at = NOW()
		WHERE id = $5 AND company_id = $6
		RETURNING
			id, company_id, email, password_hash,
			first_name, last_name, role, is_active,
			created_at, updated_at
	`

	var user User

	err := r.db.QueryRow(
		ctx,
		query,
		req.FirstName,
		req.LastName,
		req.Role,
		req.IsActive,
		userID,
		companyID,
	).Scan(
		&user.ID,
		&user.CompanyID,
		&user.Email,
		&user.Password_Hash,
		&user.First_Name,
		&user.Last_Name,
		&user.Role,
		&user.Is_active,
		&user.Created_at,
		&user.Updated_at,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return &user, nil
}
func (r *Repository) GetAll(ctx context.Context, companyID uuid.UUID) ([]User, error) {
	query := `
	SELECT id, company_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
	FROM users
	WHERE company_id = $1
	`
	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {

		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.CompanyID, &user.Email, &user.Password_Hash, &user.First_Name, &user.Last_Name, &user.Role, &user.Is_active, &user.Created_at, &user.Updated_at); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
func (r *Repository) GetById(ctx context.Context, ID uuid.UUID, companyID uuid.UUID) (*User, error) {
	query := `
	SELECT id, company_id, email, password_hash, first_name, last_name, role, is_active, created_at, updated_at
	from users
	WHERE id = $1 AND company_id = $2
	`
	var user User
	row := r.db.QueryRow(ctx, query, ID, companyID)
	err := row.Scan(&user.ID, &user.CompanyID, &user.Email, &user.Password_Hash, &user.First_Name, &user.Last_Name, &user.Role, &user.Is_active, &user.Created_at, &user.Updated_at)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
