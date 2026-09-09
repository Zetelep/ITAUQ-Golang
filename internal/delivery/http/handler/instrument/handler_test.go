package instrument

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"

	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	inst, err := itauq.Load()
	if err != nil {
		t.Fatalf("itauq: %v", err)
	}
	susInst, err := sus.Load()
	if err != nil {
		t.Fatalf("sus: %v", err)
	}

	r := gin.New()
	h := NewHandler(inst, susInst)
	r.GET("/instruments/itauq", h.GetITAUQ)
	r.GET("/instruments/sus", h.GetSUS)
	return r
}

func TestGetITAUQHappy(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/instruments/itauq", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Version   string `json:"version"`
			Questions []any  `json:"questions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || resp.Data.Version == "" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(resp.Data.Questions) != 30 {
		t.Fatalf("expected 30 questions, got %d", len(resp.Data.Questions))
	}
}

func TestGetSUSHappy(t *testing.T) {
	r := newTestRouter(t)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/instruments/sus", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200. body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Questions []any `json:"questions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success || len(resp.Data.Questions) != 10 {
		t.Fatalf("expected 10 SUS questions, got %d", len(resp.Data.Questions))
	}
}
