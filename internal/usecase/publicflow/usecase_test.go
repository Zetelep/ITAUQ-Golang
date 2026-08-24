package publicflow

import (
	"context"
	"errors"
	"testing"
	"time"

	elDomain "github.com/itauq-golang/internal/domain/evaluationlink"
	domain "github.com/itauq-golang/internal/domain/respondent"
	tsDomain "github.com/itauq-golang/internal/domain/taskscenario"
	elRepo "github.com/itauq-golang/internal/repository/evaluationlink"
	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	respRepo "github.com/itauq-golang/internal/repository/respondent"
	tsRepo "github.com/itauq-golang/internal/repository/taskscenario"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
)

func mustInstrument(t *testing.T) *itauq.Loaded {
	t.Helper()
	l, err := itauq.Load()
	if err != nil {
		t.Fatalf("itauq.Load: %v", err)
	}
	return l
}

func mustSUS(t *testing.T) *sus.Loaded {
	t.Helper()
	l, err := sus.Load()
	if err != nil {
		t.Fatalf("sus.Load: %v", err)
	}
	return l
}

// fixture builds a usecase with a single active evaluation link pointing at
// a questionnaire with two task scenarios attached.
func fixture(t *testing.T) (*Usecase, string) {
	t.Helper()

	el := elRepo.NewMemoryRepository()
	q := qRepo.NewMemoryRepository()
	ts := tsRepo.NewMemoryRepository()
	resp := respRepo.NewMemoryRepository()

	const (
		questionnaireID = "questionnaire-1"
		token           = "abc12345"
	)

	now := time.Now().UTC().Add(-time.Hour)
	if err := q.Create(context.Background(), &qDomain.Questionnaire{
		ID: questionnaireID, Title: "Eval", AppName: "WisataKu",
		Status: qDomain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed questionnaire: %v", err)
	}
	if err := ts.Create(context.Background(), &tsDomain.TaskScenario{
		ID: "ts-1", QuestionnaireID: questionnaireID, Title: "Task 1",
		Instruction: "do it", TaskOrder: 1, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed task scenario 1: %v", err)
	}
	if err := ts.Create(context.Background(), &tsDomain.TaskScenario{
		ID: "ts-2", QuestionnaireID: questionnaireID, Title: "Task 2",
		Instruction: "do it again", TaskOrder: 2, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed task scenario 2: %v", err)
	}
	if err := el.Create(context.Background(), &elDomain.EvaluationLink{
		ID: "el-1", QuestionnaireID: questionnaireID, Token: token,
		CreatedBy: "creator-1", IsActive: true, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed evaluation link: %v", err)
	}

	uc := NewUsecase(el, q, ts, resp, mustInstrument(t), mustSUS(t))
	return uc, token
}

// swapLink builds a usecase whose only difference is the evaluation link (so
// individual tests can exercise inactive/expired branches).
func swapLink(t *testing.T, uc *Usecase, link *elDomain.EvaluationLink) {
	t.Helper()
	el := elRepo.NewMemoryRepository()
	if err := el.Create(context.Background(), link); err != nil {
		t.Fatalf("re-seed link: %v", err)
	}
	uc.elRepo = el
}

func TestGetEvaluationReturnsQuestionnaireAndScenarios(t *testing.T) {
	uc, token := fixture(t)

	payload, err := uc.GetEvaluation(context.Background(), token)
	if err != nil {
		t.Fatalf("GetEvaluation: %v", err)
	}
	if payload.Questionnaire.Title != "Eval" || payload.Questionnaire.AppName != "WisataKu" {
		t.Fatalf("questionnaire view: %+v", payload.Questionnaire)
	}
	if len(payload.TaskScenarios) != 2 {
		t.Fatalf("expected 2 task scenarios, got %d", len(payload.TaskScenarios))
	}
	if payload.Itauq.Version == "" || len(payload.Itauq.Questions) != 30 {
		t.Fatalf("itauq: %+v", payload.Itauq)
	}
	if !contains(payload.Itauq.Questions[0].Text, "WisataKu") {
		t.Fatalf("expected {AppName} substituted in question 1: %q", payload.Itauq.Questions[0].Text)
	}
	if len(payload.Sus.Questions) != 10 {
		t.Fatalf("expected 10 SUS questions, got %d", len(payload.Sus.Questions))
	}
	if payload.Sus.ScaleMin != 1 || payload.Sus.ScaleMax != 5 {
		t.Fatalf("sus scale: got %d-%d, want 1-5", payload.Sus.ScaleMin, payload.Sus.ScaleMax)
	}
}

func TestGetEvaluationInactiveLinkReturns409Error(t *testing.T) {
	uc, token := fixture(t)
	swapLink(t, uc, &elDomain.EvaluationLink{
		ID: "el-2", QuestionnaireID: "questionnaire-1", Token: token,
		IsActive: false, CreatedAt: time.Now(),
	})

	_, err := uc.GetEvaluation(context.Background(), token)
	if !errors.Is(err, ErrLinkInactive) {
		t.Fatalf("expected ErrLinkInactive, got %v", err)
	}
}

func TestGetEvaluationExpiredLinkReturns409Error(t *testing.T) {
	uc, token := fixture(t)
	past := time.Now().UTC().Add(-time.Hour)
	swapLink(t, uc, &elDomain.EvaluationLink{
		ID: "el-3", QuestionnaireID: "questionnaire-1", Token: token,
		IsActive: true, ExpiresAt: &past, CreatedAt: time.Now(),
	})

	_, err := uc.GetEvaluation(context.Background(), token)
	if !errors.Is(err, ErrLinkInactive) {
		t.Fatalf("expected ErrLinkInactive, got %v", err)
	}
}

func TestGetEvaluationUnknownTokenReturns404Error(t *testing.T) {
	uc, _ := fixture(t)
	_, err := uc.GetEvaluation(context.Background(), "nope")
	if !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got %v", err)
	}
}

func TestStartRespondentAndSubmitAnswers(t *testing.T) {
	uc, token := fixture(t)

	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{
		Name: "Ahmad", Age: intPtr(22), Gender: strPtr("male"),
	})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	if started.RespondentID == "" {
		t.Fatal("expected respondent_id")
	}

	// Wrong token should not authorize.
	_, err = uc.SubmitAnswers(context.Background(), "wrong", started.RespondentID, validAnswers())
	if !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("expected ErrLinkNotFound, got %v", err)
	}

	// Wrong respondent under same link should not authorize.
	_, err = uc.SubmitAnswers(context.Background(), token, "00000000-0000-0000-0000-000000000000", validAnswers())
	if !errors.Is(err, ErrRespondentNotFound) {
		t.Fatalf("expected ErrRespondentNotFound, got %v", err)
	}

	// Mismatched category should be 422.
	in := validAnswers()
	in.Answers[0].Category = "Trust"
	if _, err := uc.SubmitAnswers(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidAnswers) {
		t.Fatalf("expected ErrInvalidAnswers, got %v", err)
	}

	// Out-of-range score should be 422.
	in = validAnswers()
	in.Answers[0].Score = 8
	if _, err := uc.SubmitAnswers(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidAnswers) {
		t.Fatalf("expected ErrInvalidAnswers for score=8, got %v", err)
	}

	// Valid answers should be accepted and echoed back.
	saved, err := uc.SubmitAnswers(context.Background(), token, started.RespondentID, validAnswers())
	if err != nil {
		t.Fatalf("SubmitAnswers valid: %v", err)
	}
	if len(saved) != 30 {
		t.Fatalf("expected 30 saved answers, got %d", len(saved))
	}
}

func TestSubmitAnswersUnknownItemIDReturns422(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Siti"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	in := validAnswers()
	in.Answers[0].ItemID = 99
	if _, err := uc.SubmitAnswers(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidAnswers) {
		t.Fatalf("expected ErrInvalidAnswers, got %v", err)
	}
}

func TestSubmitTaskAttemptsRejectsUnknownScenario(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Rina"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	in := domain.AttemptsInput{Attempts: []domain.AttemptInput{
		{TaskScenarioID: "ts-1", IsSuccess: boolPtr(true), DurationSeconds: intPtr(30)},
		{TaskScenarioID: "ts-zzz", IsSuccess: boolPtr(false), DurationSeconds: intPtr(15)},
	}}
	if _, err := uc.SubmitTaskAttempts(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidAttempts) {
		t.Fatalf("expected ErrInvalidAttempts, got %v", err)
	}
}

func TestFinalizeSubmitIncompleteReturns400(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Budi"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	if _, err := uc.FinalizeSubmit(context.Background(), token, started.RespondentID); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("expected ErrIncomplete, got %v", err)
	}
}

func TestFinalizeSubmitHappyPath(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Budi"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}

	if _, err := uc.SubmitAnswers(context.Background(), token, started.RespondentID, validAnswers()); err != nil {
		t.Fatalf("SubmitAnswers: %v", err)
	}
	if _, err := uc.SubmitSUSAnswers(context.Background(), token, started.RespondentID, validSUSAnswers()); err != nil {
		t.Fatalf("SubmitSUSAnswers: %v", err)
	}
	if _, err := uc.SubmitTaskAttempts(context.Background(), token, started.RespondentID, domain.AttemptsInput{Attempts: []domain.AttemptInput{
		{TaskScenarioID: "ts-1", IsSuccess: boolPtr(true), DurationSeconds: intPtr(42)},
		{TaskScenarioID: "ts-2", IsSuccess: boolPtr(false), DurationSeconds: intPtr(15)},
	}}); err != nil {
		t.Fatalf("SubmitTaskAttempts: %v", err)
	}

	out, err := uc.FinalizeSubmit(context.Background(), token, started.RespondentID)
	if err != nil {
		t.Fatalf("FinalizeSubmit: %v", err)
	}
	if out.SubmittedAt.IsZero() {
		t.Fatal("expected submitted_at populated")
	}
}

func TestSubmitSUSAnswersHappyPath(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Susi"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	saved, err := uc.SubmitSUSAnswers(context.Background(), token, started.RespondentID, validSUSAnswers())
	if err != nil {
		t.Fatalf("SubmitSUSAnswers: %v", err)
	}
	if len(saved) != 10 {
		t.Fatalf("expected 10 saved SUS answers, got %d", len(saved))
	}
}

func TestSubmitSUSAnswersUnknownItemIDReturns422(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Susi"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	in := validSUSAnswers()
	in.Answers[0].ItemID = 11
	if _, err := uc.SubmitSUSAnswers(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidSUSAnswers) {
		t.Fatalf("expected ErrInvalidSUSAnswers for item_id=11, got %v", err)
	}
}

func TestSubmitSUSAnswersOutOfRangeScoreReturns422(t *testing.T) {
	uc, token := fixture(t)
	started, err := uc.StartRespondent(context.Background(), token, domain.StartInput{Name: "Susi"})
	if err != nil {
		t.Fatalf("StartRespondent: %v", err)
	}
	in := validSUSAnswers()
	in.Answers[0].Score = 6
	if _, err := uc.SubmitSUSAnswers(context.Background(), token, started.RespondentID, in); !errors.Is(err, ErrInvalidSUSAnswers) {
		t.Fatalf("expected ErrInvalidSUSAnswers for score=6, got %v", err)
	}
}

// helpers

func validAnswers() domain.AnswersInput {
	cats := []string{
		"Attractiveness", "Attractiveness", "Attractiveness",
		"Efficiency", "Efficiency", "Efficiency",
		"Dependability", "Dependability", "Dependability",
		"Stimulation", "Stimulation", "Stimulation",
		"Trust", "Trust", "Trust",
		"Novelty", "Novelty", "Novelty",
		"User Satisfaction", "User Satisfaction", "User Satisfaction",
		"Accessibility", "Accessibility", "Accessibility",
		"Social Interaction", "Social Interaction", "Social Interaction",
		"Learnability", "Learnability", "Learnability",
	}
	in := domain.AnswersInput{}
	for i, c := range cats {
		in.Answers = append(in.Answers, domain.AnswerInput{ItemID: i + 1, Category: c, Score: 4})
	}
	return in
}

func validSUSAnswers() domain.SUSAnswersInput {
	in := domain.SUSAnswersInput{}
	for i := 1; i <= 10; i++ {
		in.Answers = append(in.Answers, domain.SUSAnswerInput{ItemID: i, Score: 4})
	}
	return in
}

func intPtr(n int) *int       { return &n }
func strPtr(s string) *string { return &s }
func boolPtr(b bool) *bool    { return &b }

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}