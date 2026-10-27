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

func (r *Repository) Get_Stages(ctx context.Context, companyID uuid.UUID, id uuid.UUID) ([]Stage, error) {
	rows, err := r.db.Query(ctx,
		`
	select ps.id, ps.pipeline_id, ps.name, ps.position, ps.stage_type
	FROM pipeline_stages ps
	JOIN pipelines p ON p.id = ps.pipeline_id
	WHERE ps.pipeline_id = $1
  	AND p.company_id = $2
	ORDER BY ps.position ASC;
	`, id, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stages := make([]Stage, 0)
	for rows.Next() {
		var stage Stage
		err := rows.Scan(
			&stage.ID,
			&stage.PipelineID,
			&stage.Name,
			&stage.Position,
			&stage.StageType,
		)
		if err != nil {
			return nil, err
		}
		stages = append(stages, stage)
	}
	return stages, rows.Err()
}
