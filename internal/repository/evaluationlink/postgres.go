package evaluationlink

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/evaluationlink"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const evaluationLinkColumns = `id, questionnaire_id, token, created_by, is_active, expires_at, created_at`

func (r *PostgresRepository) Create(ctx context.Context, l *domain.EvaluationLink) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.evaluation_links
		(id, questionnaire_id, token, created_by, is_active, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		l.ID, l.QuestionnaireID, l.Token, l.CreatedBy, l.IsActive, l.ExpiresAt, l.CreatedAt)
	if err != nil {
		return fmt.Errorf("create evaluation link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListByQuestionnaire(ctx context.Context, questionnaireID string) ([]domain.EvaluationLink, error) {
	rows, err := r.db.Query(ctx, "SELECT "+evaluationLinkColumns+" FROM public.evaluation_links WHERE questionnaire_id = $1 ORDER BY created_at DESC", questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list evaluation links: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EvaluationLink, 0)
	for rows.Next() {
		l, err := scanEvaluationLink(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation links: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.EvaluationLink, error) {
	row := r.db.QueryRow(ctx, "SELECT "+evaluationLinkColumns+" FROM public.evaluation_links WHERE id = $1", id)
	l, err := scanEvaluationLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get evaluation link: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) GetByToken(ctx context.Context, token string) (*domain.EvaluationLink, error) {
	row := r.db.QueryRow(ctx, "SELECT "+evaluationLinkColumns+" FROM public.evaluation_links WHERE token = $1", token)
	l, err := scanEvaluationLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get evaluation link by token: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.EvaluationLink, error) {
	setClauses := ""
	args := []any{id}
	argIdx := 2

	if in.IsActive != nil {
		setClauses += fmt.Sprintf("is_active = $%d", argIdx)
		args = append(args, *in.IsActive)
		argIdx++
	}
	if in.ExpiresAt != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("expires_at = $%d", argIdx)
		args = append(args, *in.ExpiresAt)
		argIdx++
	}

	if setClauses == "" {
		return r.Get(ctx, id)
	}

	query := fmt.Sprintf("UPDATE public.evaluation_links SET %s WHERE id = $1 RETURNING "+evaluationLinkColumns, setClauses)
	row := r.db.QueryRow(ctx, query, args...)
	l, err := scanEvaluationLink(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update evaluation link: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM public.evaluation_links WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete evaluation link: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanEvaluationLink(row rowScanner) (domain.EvaluationLink, error) {
	var l domain.EvaluationLink
	if err := row.Scan(&l.ID, &l.QuestionnaireID, &l.Token, &l.CreatedBy, &l.IsActive, &l.ExpiresAt, &l.CreatedAt); err != nil {
		return l, err
	}
	return l, nil
}
