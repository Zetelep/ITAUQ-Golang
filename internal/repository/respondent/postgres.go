package respondent

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/itauq-golang/internal/domain/respondent"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const respondentColumns = `id, evaluation_link_id, name, email, age, gender::text, occupation, started_at, submitted_at, created_at`

func (r *PostgresRepository) CreateRespondent(ctx context.Context, p *domain.Respondent) error {
	var gender any
	if p.Gender != nil {
		gender = string(*p.Gender)
	}
	_, err := r.db.Exec(ctx, `INSERT INTO public.respondents
		(id, evaluation_link_id, name, email, age, gender, occupation, started_at, submitted_at, created_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6::gender_type, NULLIF($7, ''), $8, $9, $10)`,
		p.ID, p.EvaluationLinkID, p.Name, p.Email, p.Age, gender, p.Occupation, p.StartedAt, p.SubmittedAt, p.CreatedAt)
	if err != nil {
		return fmt.Errorf("create respondent: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetRespondent(ctx context.Context, id string) (*domain.Respondent, error) {
	row := r.db.QueryRow(ctx, "SELECT "+respondentColumns+" FROM public.respondents WHERE id = $1", id)
	p, err := scanRespondent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get respondent: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) CreateAttempts(ctx context.Context, attempts []domain.Attempt) error {
	if len(attempts) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, a := range attempts {
		batch.Queue(`INSERT INTO public.task_scenario_attempts
			(id, respondent_id, task_scenario_id, is_success, duration_seconds, notes, created_at)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)`,
			a.ID, a.RespondentID, a.TaskScenarioID, a.IsSuccess, a.DurationSeconds, a.Notes, a.CreatedAt)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range attempts {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("create attempts: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) CreateAnswers(ctx context.Context, answers []domain.Answer) error {
	if len(answers) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(`INSERT INTO public.questionnaire_answers
			(id, respondent_id, item_id, category, score, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, a.RespondentID, a.ItemID, a.Category, a.Score, a.CreatedAt)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range answers {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("create answers: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) CreateSUSAnswers(ctx context.Context, answers []domain.SUSAnswer) error {
	if len(answers) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, a := range answers {
		batch.Queue(`INSERT INTO public.sus_answers
			(id, respondent_id, item_id, score, created_at)
			VALUES ($1, $2, $3, $4, $5)`,
			a.ID, a.RespondentID, a.ItemID, a.Score, a.CreatedAt)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()
	for range answers {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("create sus answers: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) CountAnswers(ctx context.Context, respondentID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.questionnaire_answers WHERE respondent_id = $1", respondentID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count answers: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) CountAttempts(ctx context.Context, respondentID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.task_scenario_attempts WHERE respondent_id = $1", respondentID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count attempts: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) CountSUSAnswers(ctx context.Context, respondentID string) (int, error) {
	var n int
	err := r.db.QueryRow(ctx, "SELECT count(*) FROM public.sus_answers WHERE respondent_id = $1", respondentID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count sus answers: %w", err)
	}
	return n, nil
}

func (r *PostgresRepository) MarkSubmitted(ctx context.Context, id string, at time.Time) error {
	tag, err := r.db.Exec(ctx, "UPDATE public.respondents SET submitted_at = $2 WHERE id = $1", id, at)
	if err != nil {
		return fmt.Errorf("mark submitted: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanRespondent(row rowScanner) (domain.Respondent, error) {
	var p domain.Respondent
	var email, occupation, gender *string
	if err := row.Scan(&p.ID, &p.EvaluationLinkID, &p.Name, &email, &p.Age, &gender, &occupation, &p.StartedAt, &p.SubmittedAt, &p.CreatedAt); err != nil {
		return p, err
	}
	if email != nil {
		p.Email = *email
	}
	if occupation != nil {
		p.Occupation = *occupation
	}
	p.Gender = gender
	return p, nil
}