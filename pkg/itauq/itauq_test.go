package itauq

import "testing"

func TestLoadReturnsInstrument(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if l.Version == "" {
		t.Fatal("expected version to be populated")
	}
	if l.ScaleMin != 1 || l.ScaleMax != 7 {
		t.Fatalf("scale range: got %d-%d, want 1-7", l.ScaleMin, l.ScaleMax)
	}
	if got := len(l.Questions); got != 30 {
		t.Fatalf("expected 30 questions, got %d", got)
	}
}

func TestItemCategoryLookup(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cat, ok := l.ItemCategory(1)
	if !ok || cat != "Attractiveness" {
		t.Fatalf("item 1: got (%q, %v), want (Attractiveness, true)", cat, ok)
	}
	if _, ok := l.ItemCategory(999); ok {
		t.Fatal("expected item 999 to be missing")
	}
}

func TestRenderQuestionSubstitutesAppName(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	q := l.Questions[0]
	rendered := l.RenderQuestion(q, "WisataKu")
	if rendered.Text == q.Text {
		t.Fatal("expected placeholder to be substituted")
	}
	if !contains(rendered.Text, "WisataKu") {
		t.Fatalf("rendered text missing app name: %q", rendered.Text)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}