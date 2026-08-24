package evaluationlink

import (
	"context"
	"errors"
	"sort"
	"sync"

	domain "github.com/itauq-golang/internal/domain/evaluationlink"
)

var ErrNotFound = errors.New("evaluation link not found")

type Repository interface {
	Create(context.Context, *domain.EvaluationLink) error
	ListByQuestionnaire(context.Context, string) ([]domain.EvaluationLink, error)
	Get(context.Context, string) (*domain.EvaluationLink, error)
	GetByToken(context.Context, string) (*domain.EvaluationLink, error)
	Update(context.Context, string, domain.UpdateInput) (*domain.EvaluationLink, error)
	Delete(context.Context, string) error
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.EvaluationLink
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.EvaluationLink)}
}

func (r *MemoryRepository) Create(_ context.Context, l *domain.EvaluationLink) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *l
	r.items[l.ID] = &cp
	return nil
}

func (r *MemoryRepository) ListByQuestionnaire(_ context.Context, questionnaireID string) ([]domain.EvaluationLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.EvaluationLink, 0)
	for _, x := range r.items {
		if x.QuestionnaireID == questionnaireID {
			all = append(all, *x)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})
	return all, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.EvaluationLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (r *MemoryRepository) GetByToken(_ context.Context, token string) (*domain.EvaluationLink, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, x := range r.items {
		if x.Token == token {
			cp := *x
			return &cp, nil
		}
	}
	return nil, ErrNotFound
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.EvaluationLink, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.IsActive != nil {
		x.IsActive = *in.IsActive
	}
	if in.ExpiresAt != nil {
		x.ExpiresAt = in.ExpiresAt
	}
	cp := *x
	return &cp, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)
	return nil
}
