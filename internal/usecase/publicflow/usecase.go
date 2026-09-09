// Package publicflow orchestrates the anonymous respondent flow:
// token-gated evaluation loading, respondent creation, task attempt and answer
// submission, and finalization. See API_SPECIFICATION.md §8.
package publicflow

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/respondent"
	elDomain "github.com/itauq-golang/internal/domain/evaluationlink"
	eligDomain "github.com/itauq-golang/internal/domain/eligibility"
	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	elRepo "github.com/itauq-golang/internal/repository/evaluationlink"
	eligRepo "github.com/itauq-golang/internal/repository/eligibility"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	respRepo "github.com/itauq-golang/internal/repository/respondent"
	tsRepo "github.com/itauq-golang/internal/repository/taskscenario"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
)

var (
	// ErrLinkNotFound is returned when the token does not resolve to any
	// evaluation link. Maps to HTTP 404.
	ErrLinkNotFound = errors.New("evaluation link not found")
	// ErrLinkInactive is returned when the link is_active=false or past its
	// expires_at. Maps to HTTP 409.
	ErrLinkInactive = errors.New("evaluation link is inactive or expired")
	// ErrRespondentNotFound is returned when the respondent_id does not exist
	// or does not belong to the resolved evaluation link. Maps to HTTP 404.
	ErrRespondentNotFound = errors.New("respondent not found for this evaluation link")
	// ErrInvalidAnswers is returned when one or more answers fail instrument
	// validation (unknown item_id, mismatched category, or out-of-range score).
	// Maps to HTTP 422.
	ErrInvalidAnswers = errors.New("answers do not match the itauq instrument")
	// ErrInvalidSUSAnswers is returned when one or more SUS answers fail
	// instrument validation (unknown item_id or out-of-range score).
	// Maps to HTTP 422.
	ErrInvalidSUSAnswers = errors.New("sus answers do not match the sus instrument")
	// ErrInvalidAttempts is returned when one or more task attempts reference a
	// task scenario that does not belong to the questionnaire. Maps to HTTP 422.
	ErrInvalidAttempts = errors.New("task attempts reference unknown scenarios")
	// ErrIncomplete is returned when the respondent tries to finalize without
	// submitting all required answers/attempts. Maps to HTTP 400.
	ErrIncomplete = errors.New("respondent session is incomplete")
	// ErrEligibilityNotConfirmed is returned when the respondent's
	// checked_criteria_ids does not cover every active criterion for the
	// questionnaire. Maps to HTTP 422.
	ErrEligibilityNotConfirmed = errors.New("respondent did not confirm every eligibility criterion")
)

// Usecase wires together the repos needed to serve the public flow.
type Usecase struct {
	elRepo    elRepo.Repository
	qRepo     qRepo.Repository
	tsRepo    tsRepo.Repository
	respRepo  respRepo.Repository
	eligRepo  eligRepo.Repository
	itauq     *itauq.Loaded
	sus       *sus.Loaded
}

// NewUsecase builds a Usecase. The ITAUQ and SUS loaders are parsed at startup
// and shared across requests.
func NewUsecase(el elRepo.Repository, q qRepo.Repository, ts tsRepo.Repository, resp respRepo.Repository, elig eligRepo.Repository, instrument *itauq.Loaded, susInstrument *sus.Loaded) *Usecase {
	return &Usecase{elRepo: el, qRepo: q, tsRepo: ts, respRepo: resp, eligRepo: elig, itauq: instrument, sus: susInstrument}
}

// EvaluationPayload is the GET /public/evaluation/:token response.
type EvaluationPayload struct {
	Questionnaire     QuestionnaireView       `json:"questionnaire"`
	EligibilityCriteria []EligibilityCriterionView `json:"eligibility_criteria"`
	TaskScenarios     []TaskScenarioView      `json:"task_scenarios"`
	Itauq             ItauqView               `json:"itauq"`
	Sus               SusView                 `json:"sus"`
}

// QuestionnaireView is the questionnaire subset returned to respondents.
type QuestionnaireView struct {
	Title   string `json:"title"`
	AppName string `json:"app_name"`
}

// TaskScenarioView is the task scenario shape returned to respondents.
type TaskScenarioView struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Instruction string `json:"instruction"`
	TaskOrder   int    `json:"task_order"`
}

// EligibilityCriterionView is one statement the respondent must tick before
// the identity form. Returned inside the GET /public/evaluation/:token
// payload (see API_SPECIFICATION.md §8).
type EligibilityCriterionView struct {
	ID            string `json:"id"`
	Statement     string `json:"statement"`
	CriteriaOrder int    `json:"criteria_order"`
}

// ItauqView is the ITAUQ instrument returned to respondents, with {AppName}
// substituted by the questionnaire's app name.
type ItauqView struct {
	Version   string           `json:"version"`
	ScaleMin  int              `json:"scale_min"`
	ScaleMax  int              `json:"scale_max"`
	Questions []itauq.Question `json:"questions"`
}

// SusView is the SUS instrument returned to respondents.
type SusView struct {
	Version   string       `json:"version"`
	ScaleMin  int          `json:"scale_min"`
	ScaleMax  int          `json:"scale_max"`
	Questions []sus.Question `json:"questions"`
}

// Started is the response of POST .../respondents.
type Started struct {
	RespondentID string    `json:"respondent_id"`
	StartedAt    time.Time `json:"started_at"`
}

// Submitted is the response of POST .../submit.
type Submitted struct {
	SubmittedAt time.Time `json:"submitted_at"`
}

// resolveLink looks up the evaluation link by token and applies active/expires
// checks. Returns ErrLinkNotFound or ErrLinkInactive on the appropriate errors.
func (u *Usecase) resolveLink(ctx context.Context, token string) (*elDomain.EvaluationLink, error) {
	link, err := u.elRepo.GetByToken(ctx, token)
	if err != nil {
		if errors.Is(err, elRepo.ErrNotFound) {
			return nil, ErrLinkNotFound
		}
		return nil, fmt.Errorf("get evaluation link: %w", err)
	}
	if !link.IsActive {
		return nil, ErrLinkInactive
	}
	if link.ExpiresAt != nil && !link.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrLinkInactive
	}
	return link, nil
}

// resolveContext looks up both the link and its questionnaire.
func (u *Usecase) resolveContext(ctx context.Context, token string) (*elDomain.EvaluationLink, *qDomain.Questionnaire, error) {
	link, err := u.resolveLink(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	q, err := u.qRepo.Get(ctx, link.QuestionnaireID)
	if err != nil {
		if errors.Is(err, qRepo.ErrNotFound) {
			return nil, nil, fmt.Errorf("questionnaire missing: %w", err)
		}
		return nil, nil, err
	}
	return link, q, nil
}

// GetEvaluation builds the payload for GET /public/evaluation/:token.
func (u *Usecase) GetEvaluation(ctx context.Context, token string) (*EvaluationPayload, error) {
	link, q, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	scenarios, err := u.tsRepo.ListByQuestionnaire(ctx, link.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list task scenarios: %w", err)
	}

	views := make([]TaskScenarioView, 0, len(scenarios))
	for _, s := range scenarios {
		views = append(views, TaskScenarioView{
			ID:          s.ID,
			Title:       s.Title,
			Instruction: s.Instruction,
			TaskOrder:   s.TaskOrder,
		})
	}

	criteria, err := u.eligRepo.ListByQuestionnaire(ctx, link.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list eligibility criteria: %w", err)
	}
	critViews := make([]EligibilityCriterionView, 0, len(criteria))
	for _, c := range criteria {
		critViews = append(critViews, EligibilityCriterionView{
			ID:            c.ID,
			Statement:     c.Statement,
			CriteriaOrder: c.CriteriaOrder,
		})
	}

	questions := make([]itauq.Question, 0, len(u.itauq.Questions))
	for _, qu := range u.itauq.Questions {
		questions = append(questions, u.itauq.RenderQuestion(qu, q.AppName))
	}

	return &EvaluationPayload{
		Questionnaire:      QuestionnaireView{Title: q.Title, AppName: q.AppName},
		EligibilityCriteria: critViews,
		TaskScenarios:      views,
		Itauq: ItauqView{
			Version:   u.itauq.Version,
			ScaleMin:  u.itauq.ScaleMin,
			ScaleMax:  u.itauq.ScaleMax,
			Questions: questions,
		},
		Sus: SusView{
			Version:   u.sus.Version,
			ScaleMin:  u.sus.ScaleMin,
			ScaleMax:  u.sus.ScaleMax,
			Questions: u.sus.Questions,
		},
	}, nil
}

// StartRespondent handles POST /public/evaluation/:token/respondents. If the
// questionnaire has any active eligibility criteria, the respondent must have
// ticked every one of them in CheckedCriteriaIDs; partial coverage is rejected
// with ErrEligibilityNotConfirmed. With zero criteria, the gate is skipped so
// existing/migrated questionnaires are not affected.
func (u *Usecase) StartRespondent(ctx context.Context, token string, in domain.StartInput) (*Started, error) {
	link, _, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, errors.New("name is required")
	}
	if in.Age != nil && *in.Age < 0 {
		return nil, errors.New("age must be non-negative")
	}
	if in.Gender != nil {
		g := strings.ToLower(strings.TrimSpace(*in.Gender))
		if g != "male" && g != "female" && g != "other" && g != "prefer_not_to_say" {
			return nil, errors.New("gender must be one of male, female, other, prefer_not_to_say")
		}
		in.Gender = &g
	}
	in.Email = strings.TrimSpace(in.Email)
	in.Occupation = strings.TrimSpace(in.Occupation)

	// Eligibility gate: every active criterion for this questionnaire must
	// appear in CheckedCriteriaIDs. If the questionnaire defines no criteria,
	// the gate is skipped.
	requiredIDs, err := u.eligRepo.ListIDsForQuestionnaire(ctx, link.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list eligibility criteria: %w", err)
	}
	if len(requiredIDs) > 0 {
		checked := make(map[string]struct{}, len(in.CheckedCriteriaIDs))
		for _, id := range in.CheckedCriteriaIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				checked[id] = struct{}{}
			}
		}
		for _, id := range requiredIDs {
			if _, ok := checked[id]; !ok {
				return nil, ErrEligibilityNotConfirmed
			}
		}
	}

	now := time.Now().UTC()
	p := &domain.Respondent{
		ID:               uuid.NewString(),
		EvaluationLinkID: link.ID,
		Name:             in.Name,
		Email:            in.Email,
		Age:              in.Age,
		Gender:           in.Gender,
		Occupation:       in.Occupation,
		StartedAt:        now,
		CreatedAt:        now,
	}
	if err := u.respRepo.CreateRespondent(ctx, p); err != nil {
		return nil, err
	}

	// Persist the audit trail of which criteria were ticked. No-op when the
	// questionnaire has no criteria.
	if len(requiredIDs) > 0 {
		confs := make([]eligDomain.Confirmation, 0, len(in.CheckedCriteriaIDs))
		for _, id := range in.CheckedCriteriaIDs {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			confs = append(confs, eligDomain.Confirmation{
				ID:           uuid.NewString(),
				RespondentID: p.ID,
				CriteriaID:   id,
				IsChecked:    true,
				CreatedAt:    now,
			})
		}
		if err := u.eligRepo.CreateConfirmations(ctx, confs); err != nil {
			return nil, fmt.Errorf("save eligibility confirmations: %w", err)
		}
	}

	return &Started{RespondentID: p.ID, StartedAt: p.StartedAt}, nil
}

// SubmitTaskAttempts handles POST .../task-attempts. Validates that every
// task_scenario_id belongs to the questionnaire and persists the batch.
func (u *Usecase) SubmitTaskAttempts(ctx context.Context, token, respondentID string, in domain.AttemptsInput) ([]domain.Attempt, error) {
	link, _, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := u.authorizeRespondent(ctx, link.ID, respondentID); err != nil {
		return nil, err
	}
	if len(in.Attempts) == 0 {
		return nil, errors.New("attempts must not be empty")
	}

	scenarios, err := u.tsRepo.ListByQuestionnaire(ctx, link.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list task scenarios: %w", err)
	}
	scenarioByID := make(map[string]struct{}, len(scenarios))
	for _, s := range scenarios {
		scenarioByID[s.ID] = struct{}{}
	}

	now := time.Now().UTC()
	out := make([]domain.Attempt, 0, len(in.Attempts))
	for _, a := range in.Attempts {
		a.TaskScenarioID = strings.TrimSpace(a.TaskScenarioID)
		if _, ok := scenarioByID[a.TaskScenarioID]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrInvalidAttempts, a.TaskScenarioID)
		}
		if a.DurationSeconds != nil && *a.DurationSeconds < 0 {
			return nil, errors.New("duration_seconds must be non-negative")
		}
		out = append(out, domain.Attempt{
			ID:              uuid.NewString(),
			RespondentID:    respondentID,
			TaskScenarioID:  a.TaskScenarioID,
			IsSuccess:       a.IsSuccess,
			DurationSeconds: a.DurationSeconds,
			Notes:           a.Notes,
			CreatedAt:       now,
		})
	}

	if err := u.respRepo.CreateAttempts(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubmitAnswers handles POST .../answers. Validates against the ITAUQ instrument
// before inserting.
func (u *Usecase) SubmitAnswers(ctx context.Context, token, respondentID string, in domain.AnswersInput) ([]domain.Answer, error) {
	link, _, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := u.authorizeRespondent(ctx, link.ID, respondentID); err != nil {
		return nil, err
	}
	if len(in.Answers) == 0 {
		return nil, errors.New("answers must not be empty")
	}

	now := time.Now().UTC()
	out := make([]domain.Answer, 0, len(in.Answers))
	for _, a := range in.Answers {
		if a.ItemID < 1 || a.ItemID > len(u.itauq.Questions) {
			return nil, fmt.Errorf("%w: item_id %d out of range", ErrInvalidAnswers, a.ItemID)
		}
		if a.Score < u.itauq.ScaleMin || a.Score > u.itauq.ScaleMax {
			return nil, fmt.Errorf("%w: score %d out of range", ErrInvalidAnswers, a.Score)
		}
		expected, ok := u.itauq.ItemCategory(a.ItemID)
		if !ok {
			return nil, fmt.Errorf("%w: unknown item_id %d", ErrInvalidAnswers, a.ItemID)
		}
		if !strings.EqualFold(strings.TrimSpace(a.Category), expected) {
			return nil, fmt.Errorf("%w: item %d expects category %q, got %q", ErrInvalidAnswers, a.ItemID, expected, a.Category)
		}
		out = append(out, domain.Answer{
			ID:           uuid.NewString(),
			RespondentID: respondentID,
			ItemID:       a.ItemID,
			Category:     expected,
			Score:        a.Score,
			CreatedAt:    now,
		})
	}

	if err := u.respRepo.CreateAnswers(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SubmitSUSAnswers handles POST .../sus-answers. Validates against the SUS
// instrument (10 fixed items, score 1-5) before inserting.
func (u *Usecase) SubmitSUSAnswers(ctx context.Context, token, respondentID string, in domain.SUSAnswersInput) ([]domain.SUSAnswer, error) {
	link, _, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := u.authorizeRespondent(ctx, link.ID, respondentID); err != nil {
		return nil, err
	}
	if len(in.Answers) == 0 {
		return nil, errors.New("sus answers must not be empty")
	}

	now := time.Now().UTC()
	out := make([]domain.SUSAnswer, 0, len(in.Answers))
	for _, a := range in.Answers {
		if _, ok := u.sus.Item(a.ItemID); !ok {
			return nil, fmt.Errorf("%w: item_id %d out of range", ErrInvalidSUSAnswers, a.ItemID)
		}
		if a.Score < u.sus.ScaleMin || a.Score > u.sus.ScaleMax {
			return nil, fmt.Errorf("%w: score %d out of range", ErrInvalidSUSAnswers, a.Score)
		}
		out = append(out, domain.SUSAnswer{
			ID:           uuid.NewString(),
			RespondentID: respondentID,
			ItemID:       a.ItemID,
			Score:        a.Score,
			CreatedAt:    now,
		})
	}

	if err := u.respRepo.CreateSUSAnswers(ctx, out); err != nil {
		return nil, err
	}
	return out, nil
}

// FinalizeSubmit handles POST .../submit. Verifies all answers, SUS answers,
// and attempts exist before stamping submitted_at.
func (u *Usecase) FinalizeSubmit(ctx context.Context, token, respondentID string) (*Submitted, error) {
	link, _, err := u.resolveContext(ctx, token)
	if err != nil {
		return nil, err
	}
	if _, err := u.authorizeRespondent(ctx, link.ID, respondentID); err != nil {
		return nil, err
	}

	answerCount, err := u.respRepo.CountAnswers(ctx, respondentID)
	if err != nil {
		return nil, err
	}
	expectedAnswers := len(u.itauq.Questions)
	if answerCount != expectedAnswers {
		return nil, fmt.Errorf("%w: expected %d answers, got %d", ErrIncomplete, expectedAnswers, answerCount)
	}

	susCount, err := u.respRepo.CountSUSAnswers(ctx, respondentID)
	if err != nil {
		return nil, err
	}
	expectedSUS := len(u.sus.Questions)
	if susCount != expectedSUS {
		return nil, fmt.Errorf("%w: expected %d sus answers, got %d", ErrIncomplete, expectedSUS, susCount)
	}

	attemptCount, err := u.respRepo.CountAttempts(ctx, respondentID)
	if err != nil {
		return nil, err
	}
	scenarios, err := u.tsRepo.ListByQuestionnaire(ctx, link.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list task scenarios: %w", err)
	}
	if attemptCount != len(scenarios) {
		return nil, fmt.Errorf("%w: expected %d attempts, got %d", ErrIncomplete, len(scenarios), attemptCount)
	}

	now := time.Now().UTC()
	if err := u.respRepo.MarkSubmitted(ctx, respondentID, now); err != nil {
		return nil, err
	}
	return &Submitted{SubmittedAt: now}, nil
}

// authorizeRespondent verifies the respondent_id exists and belongs to the
// evaluation link resolved from the token. This is the session-key check.
func (u *Usecase) authorizeRespondent(ctx context.Context, linkID, respondentID string) (*domain.Respondent, error) {
	p, err := u.respRepo.GetRespondent(ctx, respondentID)
	if err != nil {
		if errors.Is(err, respRepo.ErrNotFound) {
			return nil, ErrRespondentNotFound
		}
		return nil, fmt.Errorf("get respondent: %w", err)
	}
	if p.EvaluationLinkID != linkID {
		return nil, ErrRespondentNotFound
	}
	return p, nil
}