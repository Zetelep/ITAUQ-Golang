package evaluation

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository against the Supabase/Postgres
// schema. It does NOT depend on the v_* views mentioned in the spec — the
// views don't exist yet and scoring is cheap to do in Go from the raw rows.
type PostgresRepository struct{ db *pgxpool.Pool }

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

const respondentRowColumns = `
	r.id,
	r.name,
	q.id,
	q.title,
	q.app_name,
	q.administrator_id,
	COALESCE(p.full_name, ''),
	r.submitted_at
`

// ListRespondents joins respondents -> evaluation_links -> questionnaires ->
// profiles to materialize RespondentRow. Scope by AdministratorID (set for
// administrator callers) or QuestionnaireID (optional filter). Sorted newest
// submitted_at first; respondents that haven't submitted yet sort last.
func (r *PostgresRepository) ListRespondents(ctx context.Context, f ListFilter) ([]RespondentRow, int, error) {
	where := " WHERE 1=1"
	args := []any{}
	idx := 1

	if f.AdministratorID != "" {
		where += fmt.Sprintf(" AND q.administrator_id = $%d", idx)
		args = append(args, f.AdministratorID)
		idx++
	}
	if f.QuestionnaireID != "" {
		where += fmt.Sprintf(" AND q.id = $%d", idx)
		args = append(args, f.QuestionnaireID)
		idx++
	}

	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT count(*)
		FROM public.respondents r
		JOIN public.evaluation_links l ON l.id = r.evaluation_link_id
		JOIN public.questionnaires q   ON q.id = l.questionnaire_id
		LEFT JOIN public.profiles p    ON p.id = q.administrator_id
	`+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count respondents: %w", err)
	}

	args = append(args, f.PageSize, (f.Page-1)*f.PageSize)
	limitArg := fmt.Sprintf("$%d", idx)
	offsetArg := fmt.Sprintf("$%d", idx+1)

	rows, err := r.db.Query(ctx, `
		SELECT `+respondentRowColumns+`
		FROM public.respondents r
		JOIN public.evaluation_links l ON l.id = r.evaluation_link_id
		JOIN public.questionnaires q   ON q.id = l.questionnaire_id
		LEFT JOIN public.profiles p    ON p.id = q.administrator_id
	`+where+`
		ORDER BY r.submitted_at DESC NULLS LAST, r.created_at DESC
		LIMIT `+limitArg+` OFFSET `+offsetArg, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list respondents: %w", err)
	}
	defer rows.Close()

	out := make([]RespondentRow, 0)
	for rows.Next() {
		row, err := scanRespondentRow(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate respondents: %w", err)
	}
	return out, total, nil
}

// GetRespondent returns the joined row for a single respondent id.
func (r *PostgresRepository) GetRespondent(ctx context.Context, id string) (*RespondentRow, error) {
	row := r.db.QueryRow(ctx, `
		SELECT `+respondentRowColumns+`
		FROM public.respondents r
		JOIN public.evaluation_links l ON l.id = r.evaluation_link_id
		JOIN public.questionnaires q   ON q.id = l.questionnaire_id
		LEFT JOIN public.profiles p    ON p.id = q.administrator_id
		WHERE r.id = $1`, id)
	out, err := scanRespondentRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRespondentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get respondent: %w", err)
	}
	return &out, nil
}

// ListAnswers returns the raw item_id -> score map for a respondent.
func (r *PostgresRepository) ListAnswers(ctx context.Context, respondentID string) (map[int]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT item_id, score FROM public.questionnaire_answers WHERE respondent_id = $1`,
		respondentID)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	defer rows.Close()
	out := make(map[int]int)
	for rows.Next() {
		var id, score int
		if err := rows.Scan(&id, &score); err != nil {
			return nil, fmt.Errorf("scan answer: %w", err)
		}
		out[id] = score
	}
	return out, rows.Err()
}

// ListSUSAnswers returns the raw item_id -> score map for a respondent.
func (r *PostgresRepository) ListSUSAnswers(ctx context.Context, respondentID string) (map[int]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT item_id, score FROM public.sus_answers WHERE respondent_id = $1`,
		respondentID)
	if err != nil {
		return nil, fmt.Errorf("list sus answers: %w", err)
	}
	defer rows.Close()
	out := make(map[int]int)
	for rows.Next() {
		var id, score int
		if err := rows.Scan(&id, &score); err != nil {
			return nil, fmt.Errorf("scan sus answer: %w", err)
		}
		out[id] = score
	}
	return out, rows.Err()
}

// ListTaskAttempts returns the attempts joined with their scenario title.
func (r *PostgresRepository) ListTaskAttempts(ctx context.Context, respondentID string) ([]TaskAttemptRow, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.task_scenario_id, s.title, a.is_success, a.duration_seconds
		FROM public.task_scenario_attempts a
		JOIN public.task_scenarios s ON s.id = a.task_scenario_id
		WHERE a.respondent_id = $1
		ORDER BY s.task_order ASC, s.created_at ASC`,
		respondentID)
	if err != nil {
		return nil, fmt.Errorf("list task attempts: %w", err)
	}
	defer rows.Close()
	out := make([]TaskAttemptRow, 0)
	for rows.Next() {
		var a TaskAttemptRow
		if err := rows.Scan(&a.TaskScenarioID, &a.Title, &a.IsSuccess, &a.DurationSeconds); err != nil {
			return nil, fmt.Errorf("scan attempt: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetQuestionnaire returns the identity row for a questionnaire.
func (r *PostgresRepository) GetQuestionnaire(ctx context.Context, id string) (*QuestionnaireInfo, error) {
	row := r.db.QueryRow(ctx,
		`SELECT id, title, app_name, administrator_id FROM public.questionnaires WHERE id = $1`, id)
	var q QuestionnaireInfo
	if err := row.Scan(&q.ID, &q.Title, &q.AppName, &q.AdministratorID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuestionnaireNotFound
		}
		return nil, fmt.Errorf("get questionnaire: %w", err)
	}
	return &q, nil
}

// ListRespondentIDs returns the submitted+in-progress respondent IDs for a
// questionnaire, used to feed the report aggregator. The report is meaningful
// even if some respondents haven't submitted yet (their empty scores are
// simply skipped in averaging).
func (r *PostgresRepository) ListRespondentIDs(ctx context.Context, questionnaireID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT r.id
		FROM public.respondents r
		JOIN public.evaluation_links l ON l.id = r.evaluation_link_id
		WHERE l.questionnaire_id = $1`,
		questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list respondent ids: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan respondent id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// GetRespondentIdentity returns age/gender/occupation for the detail endpoint.
func (r *PostgresRepository) GetRespondentIdentity(ctx context.Context, id string) (*RespondentIdentity, error) {
	var out RespondentIdentity
	var gender, occupation *string
	err := r.db.QueryRow(ctx,
		`SELECT age, gender::text, occupation FROM public.respondents WHERE id = $1`, id,
	).Scan(&out.Age, &gender, &occupation)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRespondentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get respondent identity: %w", err)
	}
	out.Gender = gender
	if occupation != nil {
		out.Occupation = *occupation
	}
	return &out, nil
}

type rowScanner interface{ Scan(...any) error }

func scanRespondentRow(row rowScanner) (RespondentRow, error) {
	var out RespondentRow
	if err := row.Scan(
		&out.RespondentID,
		&out.RespondentName,
		&out.QuestionnaireID,
		&out.QuestionnaireTitle,
		&out.AppName,
		&out.AdministratorID,
		&out.AdministratorName,
		&out.SubmittedAt,
	); err != nil {
		return out, err
	}
	return out, nil
}
