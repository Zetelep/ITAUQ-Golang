package questionnaire

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/questionnaire"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const questionnaireColumns = `id, administrator_id, title, app_name, description, itauq_version, status::text, created_at, updated_at`

func (r *PostgresRepository) Create(ctx context.Context, q *domain.Questionnaire) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.questionnaires
		(id, administrator_id, title, app_name, description, itauq_version, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9)`,
		q.ID, q.AdministratorID, q.Title, q.AppName, q.Description, q.ItauqVersion, string(q.Status), q.CreatedAt, q.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create questionnaire: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, f ListFilter) ([]domain.Questionnaire, int, error) {
	where := " WHERE 1=1"
	args := []any{}
	argIdx := 1

	if f.AdministratorID != "" {
		where += fmt.Sprintf(" AND administrator_id = $%d", argIdx)
		args = append(args, f.AdministratorID)
		argIdx++
	}
	if f.Status != "" {
		where += fmt.Sprintf(" AND status::text = $%d", argIdx)
		args = append(args, string(f.Status))
		argIdx++
	}

	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.questionnaires"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count questionnaires: %w", err)
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	limitArg := fmt.Sprintf("$%d", argIdx)
	offsetArg := fmt.Sprintf("$%d", argIdx+1)

	rows, err := r.db.Query(ctx, "SELECT "+questionnaireColumns+" FROM public.questionnaires"+where+" ORDER BY created_at DESC LIMIT "+limitArg+" OFFSET "+offsetArg, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list questionnaires: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Questionnaire, 0)
	for rows.Next() {
		q, err := scanQuestionnaire(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, q)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate questionnaires: %w", err)
	}
	return items, total, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.Questionnaire, error) {
	row := r.db.QueryRow(ctx, "SELECT "+questionnaireColumns+" FROM public.questionnaires WHERE id = $1", id)
	q, err := scanQuestionnaire(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get questionnaire: %w", err)
	}
	return &q, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.Questionnaire, error) {
	setClauses := "updated_at = now()"
	args := []any{id}
	argIdx := 2

	if in.Title != nil {
		setClauses += fmt.Sprintf(", title = $%d", argIdx)
		args = append(args, *in.Title)
		argIdx++
	}
	if in.AppName != nil {
		setClauses += fmt.Sprintf(", app_name = $%d", argIdx)
		args = append(args, *in.AppName)
		argIdx++
	}
	if in.Description != nil {
		setClauses += fmt.Sprintf(", description = $%d", argIdx)
		args = append(args, *in.Description)
		argIdx++
	}
	if in.Status != nil {
		setClauses += fmt.Sprintf(", status = $%d", argIdx)
		args = append(args, string(*in.Status))
		argIdx++
	}

	query := fmt.Sprintf("UPDATE public.questionnaires SET %s WHERE id = $1 RETURNING %s", setClauses, questionnaireColumns)
	row := r.db.QueryRow(ctx, query, args...)
	q, err := scanQuestionnaire(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update questionnaire: %w", err)
	}
	return &q, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM public.questionnaires WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete questionnaire: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanQuestionnaire(row rowScanner) (domain.Questionnaire, error) {
	var q domain.Questionnaire
	var status string
	var description sql.NullString
	if err := row.Scan(&q.ID, &q.AdministratorID, &q.Title, &q.AppName, &description, &q.ItauqVersion, &status, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return q, err
	}
	if description.Valid {
		q.Description = description.String
	}
	q.Status = domain.Status(status)
	return q, nil
}
