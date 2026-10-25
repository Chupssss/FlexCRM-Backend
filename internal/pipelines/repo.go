package pipelines

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get_Option(ctx context.Context, companyID uuid.UUID) ([]PipelineOption, error) {
	rows, err := r.db.Query(ctx,
		`
		select id, name from pipelines
		where company_id = $1
		and is_active = true
		order by name
		`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	options := make([]PipelineOption, 0)
	for rows.Next() {
		var option PipelineOption
		if err := rows.Scan(
			&option.ID,
			&option.Name,
		); err != nil {
			return nil, err
		}
		options = append(options, option)
	}
	return options, rows.Err()
}
