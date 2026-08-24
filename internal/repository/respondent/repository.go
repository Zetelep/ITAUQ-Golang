package respondent

import (
	"context"
	"errors"
	"sync"
	"time"

	domain "github.com/itauq-golang/internal/domain/respondent"
)

var ErrNotFound = errors.New("respondent not found")

type Repository interface {
	CreateRespondent(context.Context, *domain.Respondent) error
	GetRespondent(context.Context, string) (*domain.Respondent, error)
	CreateAttempts(context.Context, []domain.Attempt) error
	CreateAnswers(context.Context, []domain.Answer) error
	CreateSUSAnswers(context.Context, []domain.SUSAnswer) error
	CountAnswers(context.Context, string) (int, error)
	CountAttempts(context.Context, string) (int, error)
	CountSUSAnswers(context.Context, string) (int, error)
	MarkSubmitted(context.Context, string, time.Time) error
}

type MemoryRepository struct {
	mu          sync.RWMutex
	respondents map[string]*domain.Respondent
	attempts    map[string]*domain.Attempt
	answers     map[string]*domain.Answer
	sus         map[string]*domain.SUSAnswer
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		respondents: make(map[string]*domain.Respondent),
		attempts:    make(map[string]*domain.Attempt),
		answers:     make(map[string]*domain.Answer),
		sus:         make(map[string]*domain.SUSAnswer),
	}
}

func (r *MemoryRepository) CreateRespondent(_ context.Context, p *domain.Respondent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.respondents[p.ID] = &cp
	return nil
}

func (r *MemoryRepository) GetRespondent(_ context.Context, id string) (*domain.Respondent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.respondents[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *MemoryRepository) CreateAttempts(_ context.Context, attempts []domain.Attempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range attempts {
		cp := attempts[i]
		r.attempts[cp.ID] = &cp
	}
	return nil
}

func (r *MemoryRepository) CreateAnswers(_ context.Context, answers []domain.Answer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range answers {
		cp := answers[i]
		r.answers[cp.ID] = &cp
	}
	return nil
}

func (r *MemoryRepository) CreateSUSAnswers(_ context.Context, answers []domain.SUSAnswer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range answers {
		cp := answers[i]
		r.sus[cp.ID] = &cp
	}
	return nil
}

func (r *MemoryRepository) CountAnswers(_ context.Context, respondentID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, a := range r.answers {
		if a.RespondentID == respondentID {
			n++
		}
	}
	return n, nil
}

func (r *MemoryRepository) CountAttempts(_ context.Context, respondentID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, a := range r.attempts {
		if a.RespondentID == respondentID {
			n++
		}
	}
	return n, nil
}

func (r *MemoryRepository) CountSUSAnswers(_ context.Context, respondentID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, a := range r.sus {
		if a.RespondentID == respondentID {
			n++
		}
	}
	return n, nil
}

func (r *MemoryRepository) MarkSubmitted(_ context.Context, id string, at time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.respondents[id]
	if !ok {
		return ErrNotFound
	}
	cp := at
	p.SubmittedAt = &cp
	return nil
}