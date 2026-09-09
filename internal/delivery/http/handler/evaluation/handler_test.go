package evaluation

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	evalRepo "github.com/itauq-golang/internal/repository/evaluation"
	evalUc "github.com/itauq-golang/internal/usecase/evaluation"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	src := evalRepo.NewMemoryDataSource()
	repo := evalRepo.NewMemoryRepository(src)

	inst, err := itauq.Load()
	if err != nil {
		t.Fatalf("itauq: %v", err)
	}
	susInst, err := sus.Load()
	if err != nil {
		t.Fatalf("sus: %v", err)
	}
	uc := evalUc.NewUsecase(repo, inst, susInst)

	r := gin.New()
	h := NewHandler(uc)

	r.GET("/respondents", h.ListRespondents)
	r.GET("/respondents/:id", h.GetRespondent)
	r.GET("/questionnaires/:id/report", h.GetQuestionnaireReport)

	return r
}

func withCaller(req *http.Request) {
	req.Header.Set("X-User-ID", "admin-1")
	req.Header.Set("X-User-Role", "super_admin")
}

func TestListRespondentsEmpty(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/respondents", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    []any `json:"data"`
	}
	if err := jsonUnmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success=true")
	}
}

func TestGetRespondentNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/respondents/nonexistent", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestGetQuestionnaireReportNotFound(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/questionnaires/nonexistent/report", nil)
	withCaller(req)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status: got %d, want 404. body=%s", w.Code, w.Body.String())
	}
}

func TestListRespondentsForbiddenForUnknownRole(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/respondents", nil)
	req.Header.Set("X-User-ID", "user-1")
	req.Header.Set("X-User-Role", "guest")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status: got %d, want 403. body=%s", w.Code, w.Body.String())
	}
}

func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
