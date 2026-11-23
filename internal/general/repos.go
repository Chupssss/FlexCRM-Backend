package general

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *PostgresRepository { return &PostgresRepository{db: db} }

// Все блоки ответа читаются из одного снимка БД и только в рамках компании из JWT.
func (r *PostgresRepository) Get(ctx context.Context, userID, companyID uuid.UUID, limit int) (*Response, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	result := &Response{Tasks: make([]Task, 0), Activities: make([]Activity, 0)}
	err = tx.QueryRow(ctx, `SELECT first_name, COALESCE(last_name,''), role FROM users
 WHERE id=$1 AND company_id=$2 AND is_active=TRUE`, userID, companyID).Scan(&result.FirstName, &result.LastName, &result.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrForbidden
	}
	if err != nil {
		return nil, err
	}
	err = tx.QueryRow(ctx, `SELECT
 (SELECT COUNT(*) FROM deals WHERE company_id=$1),
 (SELECT COUNT(*) FROM clients WHERE company_id=$1),
 (SELECT COUNT(*) FROM tasks WHERE company_id=$1 AND completed_at IS NULL AND status NOT IN ('DONE','CANCELLED'))`, companyID).Scan(&result.CountOfAllDeals, &result.CountClients, &result.CountTasks)
	if err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT title,due_at FROM tasks
 WHERE company_id=$1 AND completed_at IS NULL AND status NOT IN ('DONE','CANCELLED')
 ORDER BY due_at ASC NULLS LAST,created_at ASC,id ASC LIMIT $2`, companyID, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var task Task
		if err := rows.Scan(&task.Title, &task.DueAt); err != nil {
			rows.Close()
			return nil, err
		}
		result.Tasks = append(result.Tasks, task)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.Query(ctx, `SELECT COALESCE(NULLIF(trim(concat_ws(' ',u.first_name,u.last_name)),''),'Система'),a.action,a.created_at
 FROM activities a LEFT JOIN users u ON u.id=a.actor_id AND u.company_id=a.company_id
 WHERE a.company_id=$1
 ORDER BY a.created_at DESC,a.entity_type,a.entity_id,a.action LIMIT $2`, companyID, limit)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var activity Activity
		if err := rows.Scan(&activity.ActorName, &activity.Action, &activity.CreatedAt); err != nil {
			rows.Close()
			return nil, err
		}
		result.Activities = append(result.Activities, activity)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
