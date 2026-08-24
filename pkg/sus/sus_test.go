package sus

import "testing"

func TestLoadReturnsInstrument(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if l.Version != "sus-v1" {
		t.Fatalf("expected version sus-v1, got %q", l.Version)
	}
	if l.ScaleMin != 1 || l.ScaleMax != 5 {
		t.Fatalf("scale range: got %d-%d, want 1-5", l.ScaleMin, l.ScaleMax)
	}
	if got := len(l.Questions); got != 10 {
		t.Fatalf("expected 10 questions, got %d", got)
	}
}

func TestItemLookup(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	q, ok := l.Item(1)
	if !ok {
		t.Fatal("expected item 1 to be present")
	}
	if q.Polarity != PolarityPositive {
		t.Fatalf("item 1 polarity: got %q, want positive", q.Polarity)
	}
	q, ok = l.Item(2)
	if !ok {
		t.Fatal("expected item 2 to be present")
	}
	if q.Polarity != PolarityNegative {
		t.Fatalf("item 2 polarity: got %q, want negative", q.Polarity)
	}
	if _, ok := l.Item(11); ok {
		t.Fatal("expected item 11 to be missing")
	}
}