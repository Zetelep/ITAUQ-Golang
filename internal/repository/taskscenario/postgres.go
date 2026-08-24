package taskscenario

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/taskscenario"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const taskScenarioColumns = `id, questionnaire_id, title, instruction, task_order, created_at`

func (r *PostgresRepository) Create(ctx context.Context, t *domain.TaskScenario) error {
	_, err := r.db.Exec(ctx, `INSERT INTO public.task_scenarios
		(id, questionnaire_id, title, instruction, task_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.QuestionnaireID, t.Title, t.Instruction, t.TaskOrder, t.CreatedAt)
	if err != nil {
		return fmt.Errorf("create task scenario: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListByQuestionnaire(ctx context.Context, questionnaireID string) ([]domain.TaskScenario, error) {
	rows, err := r.db.Query(ctx, "SELECT "+taskScenarioColumns+" FROM public.task_scenarios WHERE questionnaire_id = $1 ORDER BY task_order ASC, created_at ASC", questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list task scenarios: %w", err)
	}
	defer rows.Close()

	items := make([]domain.TaskScenario, 0)
	for rows.Next() {
		t, err := scanTaskScenario(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate task scenarios: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (*domain.TaskScenario, error) {
	row := r.db.QueryRow(ctx, "SELECT "+taskScenarioColumns+" FROM public.task_scenarios WHERE id = $1", id)
	t, err := scanTaskScenario(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task scenario: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.TaskScenario, error) {
	setClauses := ""
	args := []any{id}
	argIdx := 2

	if in.Title != nil {
		setClauses += fmt.Sprintf("title = $%d", argIdx)
		args = append(args, *in.Title)
		argIdx++
	}
	if in.Instruction != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("instruction = $%d", argIdx)
		args = append(args, *in.Instruction)
		argIdx++
	}
	if in.TaskOrder != nil {
		if setClauses != "" {
			setClauses += ", "
		}
		setClauses += fmt.Sprintf("task_order = $%d", argIdx)
		args = append(args, *in.TaskOrder)
		argIdx++
	}

	if setClauses == "" {
		return r.Get(ctx, id)
	}

	query := fmt.Sprintf("UPDATE public.task_scenarios SET %s WHERE id = $1 RETURNING %s", setClauses, taskScenarioColumns)
	row := r.db.QueryRow(ctx, query, args...)
	t, err := scanTaskScenario(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update task scenario: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, "DELETE FROM public.task_scenarios WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete task scenario: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanTaskScenario(row rowScanner) (domain.TaskScenario, error) {
	var t domain.TaskScenario
	if err := row.Scan(&t.ID, &t.QuestionnaireID, &t.Title, &t.Instruction, &t.TaskOrder, &t.CreatedAt); err != nil {
		return t, err
	}
	return t, nil
}
