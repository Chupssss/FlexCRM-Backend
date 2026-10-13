package clients

import (
	"context"

	"github.com/google/uuid"
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

func (r *Repository) GetAll(
	ctx context.Context,
	companyID uuid.UUID,
) ([]Client, error) {

	query := `
		SELECT
			c.id,
			c.client_type,
			c.first_name,
			c.last_name,
			c.company_name,
			c.email,
			c.phone,
			c.created_at,
			c.updated_at,

			pc.id,
			pc.first_name,
			pc.last_name,
			pc.email,
			pc.phone,

			u.id,
			u.first_name,
			u.last_name

		FROM clients c

		LEFT JOIN contacts pc
			ON pc.client_id = c.id
			AND pc.is_primary = TRUE

		LEFT JOIN users u
			ON u.id = c.responsible_user_id

		WHERE c.company_id = $1

		ORDER BY c.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	clients := make([]Client, 0)

	for rows.Next() {
		var client Client

		var contactID *uuid.UUID
		var contactFirstName *string
		var contactLastName *string
		var contactEmail *string
		var contactPhone *string

		var responsibleID *uuid.UUID
		var responsibleFirstName *string
		var responsibleLastName *string

		err := rows.Scan(
			&client.ID,
			&client.Type,
			&client.FirstName,
			&client.LastName,
			&client.CompanyName,
			&client.Email,
			&client.Phone,
			&client.CreatedAt,
			&client.UpdatedAt,

			&contactID,
			&contactFirstName,
			&contactLastName,
			&contactEmail,
			&contactPhone,

			&responsibleID,
			&responsibleFirstName,
			&responsibleLastName,
		)
		if err != nil {
			return nil, err
		}

		if contactID != nil && contactFirstName != nil {
			client.PrimaryContact = &Contact{
				ID:        *contactID,
				FirstName: *contactFirstName,
				LastName:  contactLastName,
				Email:     contactEmail,
				Phone:     contactPhone,
			}
		}

		if responsibleID != nil && responsibleFirstName != nil {
			client.Responsible = &User{
				ID:        *responsibleID,
				FirstName: *responsibleFirstName,
				LastName:  responsibleLastName,
			}
		}

		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return clients, nil
}
