package eligibility

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/eligibility"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository is the Supabase/Postgres-backed implementation. Tables
// referenced: public.questionnaire_eligibility_criteria and
// public.respondent_eligibility_confirmations.
type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const criterionColumns = `id, questionnaire_id, statement, criteria_order, created_at`

func (r *PostgresRepository) Create(ctx context.Context, c *domain.EligibilityCriterion) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.questionnaire_eligibility_criteria
		(id, questionnaire_id, statement, criteria_order, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		c.ID, c.QuestionnaireID, c.Statement, c.CriteriaOrder, c.CreatedAt)
	if err != nil {
		return fmt.Errorf("create eligibility criterion: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListByQuestionnaire(ctx context.Context, questionnaireID string) ([]domain.EligibilityCriterion, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+criterionColumns+" FROM public.questionnaire_eligibility_criteria WHERE questionnaire_id = $1 ORDER BY criteria_order ASC, created_at ASC",
		questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list eligibility criteria: %w", err)
	}
	defer rows.Close()

	items := make([]domain.EligibilityCriterion, 0)
	for rows.Next() {
		c, err := scanCriterion(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eligibility criteria: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) ListIDsForQuestionnaire(ctx context.Context, questionnaireID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		"SELECT id FROM public.questionnaire_eligibility_criteria WHERE questionnaire_id = $1",
		questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list eligibility criteria ids: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan eligibility criterion id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate eligibility criteria ids: %w", err)
	}
	return ids, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.EligibilityCriterion, error) {
	row := r.db.QueryRow(ctx, "SELECT "+criterionColumns+" FROM public.questionnaire_eligibility_criteria WHERE id = $1", id)
	c, err := scanCriterion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get eligibility criterion: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.EligibilityCriterion, error) {
	setClauses := ""
	args := []any{id}
	argIdx := 2

	if in.Statement != nil {
		setClauses += fmt.Sprintf("statement = $%d", argIdx)
		args = append(args, *in.Statement)
		argIdx++
	}
	if in.CriteriaOrder != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("criteria_order = $%d", argIdx)
		args = append(args, *in.CriteriaOrder)
		argIdx++
	}

	if setClauses == "" {
		return r.Get(ctx, id)
	}

	query := fmt.Sprintf("UPDATE public.questionnaire_eligibility_criteria SET %s WHERE id = $1 RETURNING %s", setClauses, criterionColumns)
	row := r.db.QueryRow(ctx, query, args...)
	c, err := scanCriterion(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update eligibility criterion: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM public.questionnaire_eligibility_criteria WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete eligibility criterion: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateConfirmations(ctx context.Context, confs []domain.Confirmation) error {
	if len(confs) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, c := range confs {
		batch.Queue(`INSERT INTO public.respondent_eligibility_confirmations
			(id, respondent_id, criteria_id, is_checked, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			c.ID, c.RespondentID, c.CriteriaID, c.IsChecked, c.CreatedAt)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range confs {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("insert eligibility confirmation: %w", err)
		}
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanCriterion(row rowScanner) (domain.EligibilityCriterion, error) {
	var c domain.EligibilityCriterion
	if err := row.Scan(&c.ID, &c.QuestionnaireID, &c.Statement, &c.CriteriaOrder, &c.CreatedAt); err != nil {
		return c, err
	}
	return c, nil
}
