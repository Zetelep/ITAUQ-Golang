package publicflow

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	elDomain "github.com/itauq-golang/internal/domain/evaluationlink"
	qDomain "github.com/itauq-golang/internal/domain/questionnaire"
	tsDomain "github.com/itauq-golang/internal/domain/taskscenario"
	elRepo "github.com/itauq-golang/internal/repository/evaluationlink"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	respRepo "github.com/itauq-golang/internal/repository/respondent"
	tsRepo "github.com/itauq-golang/internal/repository/taskscenario"
	"github.com/itauq-golang/internal/usecase/publicflow"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	el := elRepo.NewMemoryRepository()
	q := qRepo.NewMemoryRepository()
	ts := tsRepo.NewMemoryRepository()
	resp := respRepo.NewMemoryRepository()

	const (
		questionnaireID = "questionnaire-1"
		token           = "tok1234"
	)
	now := time.Now().UTC().Add(-time.Hour)
	if err := q.Create(context.Background(), &qDomain.Questionnaire{
		ID: questionnaireID, Title: "Eval", AppName: "WisataKu",
		Status: qDomain.StatusActive, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed q: %v", err)
	}
	if err := ts.Create(context.Background(), &tsDomain.TaskScenario{
		ID: "ts-1", QuestionnaireID: questionnaireID, Title: "Task 1",
		Instruction: "do it", TaskOrder: 1, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed ts: %v", err)
	}
	if err := el.Create(context.Background(), &elDomain.EvaluationLink{
		ID: "el-1", QuestionnaireID: questionnaireID, Token: token,
		IsActive: true, CreatedAt: now,
	}); err != nil {
		t.Fatalf("seed el: %v", err)
	}

	inst, err := itauq.Load()
	if err != nil {
		t.Fatalf("itauq: %v", err)
	}
	susInst, err := sus.Load()
	if err != nil {
		t.Fatalf("sus: %v", err)
	}
	uc := publicflow.NewUsecase(el, q, ts, resp, inst, susInst)

	r := gin.New()
	g := r.Group("/public/evaluation")
	h := NewHandler(uc)
	g.GET("/:token", h.GetEvaluation)
	g.POST("/:token/respondents", h.StartRespondent)
	g.POST("/:token/respondents/:respondent_id/task-attempts", h.SubmitTaskAttempts)
	g.POST("/:token/respondents/:respondent_id/answers", h.SubmitAnswers)
	g.POST("/:token/respondents/:respondent_id/sus-answers", h.SubmitSUSAnswers)
	g.POST("/:token/respondents/:respondent_id/submit", h.FinalizeSubmit)
	return r, token
}

func TestHandlerGetEvaluationHappy(t *testing.T) {
	r, token := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/public/evaluation/"+token, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Questionnaire  map[string]any   `json:"questionnaire"`
			TaskScenarios  []map[string]any `json:"task_scenarios"`
			Itauq          map[string]any   `json:"itauq"`
			Sus            map[string]any   `json:"sus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !body.Success {
		t.Fatalf("expected success=true, body=%s", w.Body.String())
	}
	if body.Data.Questionnaire["app_name"] != "WisataKu" {
		t.Fatalf("expected app_name substituted, got %+v", body.Data.Questionnaire)
	}
	if len(body.Data.TaskScenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(body.Data.TaskScenarios))
	}
	if qs, ok := body.Data.Sus["questions"].([]any); !ok || len(qs) != 10 {
		t.Fatalf("expected sus with 10 questions, got %+v", body.Data.Sus)
	}
}

func TestHandlerSubmitSUSAnswersRejectsOutOfRangeScore(t *testing.T) {
	r, token := newTestRouter(t)

	// start respondent
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/public/evaluation/"+token+"/respondents", bytes.NewBufferString(`{"name":"A"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var start struct {
		Data struct {
			RespondentID string `json:"respondent_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &start)

	w = httptest.NewRecorder()
	body := bytes.NewBufferString(`{"answers":[{"item_id":1,"score":99}]}`)
	req, _ = http.NewRequest(http.MethodPost, "/public/evaluation/"+token+"/respondents/"+start.Data.RespondentID+"/sus-answers", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("sus answers: got %d, want 422. body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerGetEvaluationUnknownTokenReturns404(t *testing.T) {
	r, _ := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/public/evaluation/missing", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerStartRespondentRejectsEmptyName(t *testing.T) {
	r, token := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/public/evaluation/"+token+"/respondents", bytes.NewBufferString(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d, want 400. body=%s", w.Code, w.Body.String())
	}
}

func TestHandlerSubmitAnswersRejectsOutOfRangeScore(t *testing.T) {
	r, token := newTestRouter(t)

	// start respondent
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/public/evaluation/"+token+"/respondents", bytes.NewBufferString(`{"name":"A"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var start struct {
		Data struct {
			RespondentID string `json:"respondent_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &start)

	// attempt with out-of-range score
	w = httptest.NewRecorder()
	body := bytes.NewBufferString(`{"answers":[{"item_id":1,"category":"Attractiveness","score":99}]}`)
	req, _ = http.NewRequest(http.MethodPost, "/public/evaluation/"+token+"/respondents/"+start.Data.RespondentID+"/answers", body)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("answers: got %d, want 422. body=%s", w.Code, w.Body.String())
	}
}