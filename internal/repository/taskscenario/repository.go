package taskscenario

import (
	"context"
	"errors"
	"sort"
	"sync"

	domain "github.com/itauq-golang/internal/domain/taskscenario"
)

var ErrNotFound = errors.New("task scenario not found")

type Repository interface {
	Create(context.Context, *domain.TaskScenario) error
	ListByQuestionnaire(context.Context, string) ([]domain.TaskScenario, error)
	Get(context.Context, string) (*domain.TaskScenario, error)
	Update(context.Context, string, domain.UpdateInput) (*domain.TaskScenario, error)
	Delete(context.Context, string) error
	GetStats(context.Context, string) ([]domain.TaskScenarioStats, error)
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.TaskScenario
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.TaskScenario)}
}

func (r *MemoryRepository) Create(_ context.Context, t *domain.TaskScenario) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *t
	r.items[t.ID] = &cp
	return nil
}

func (r *MemoryRepository) ListByQuestionnaire(_ context.Context, questionnaireID string) ([]domain.TaskScenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.TaskScenario, 0)
	for _, x := range r.items {
		if x.QuestionnaireID == questionnaireID {
			all = append(all, *x)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].TaskOrder == all[j].TaskOrder {
			return all[i].CreatedAt.Before(all[j].CreatedAt)
		}
		return all[i].TaskOrder < all[j].TaskOrder
	})
	return all, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.TaskScenario, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.TaskScenario, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.Title != nil {
		x.Title = *in.Title
	}
	if in.Instruction != nil {
		x.Instruction = *in.Instruction
	}
	if in.TaskOrder != nil {
		x.TaskOrder = *in.TaskOrder
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

func (r *MemoryRepository) GetStats(_ context.Context, questionnaireID string) ([]domain.TaskScenarioStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.TaskScenario, 0)
	for _, x := range r.items {
		if x.QuestionnaireID == questionnaireID {
			all = append(all, *x)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].TaskOrder == all[j].TaskOrder {
			return all[i].CreatedAt.Before(all[j].CreatedAt)
		}
		return all[i].TaskOrder < all[j].TaskOrder
	})
	out := make([]domain.TaskScenarioStats, 0, len(all))
	for _, t := range all {
		out = append(out, domain.TaskScenarioStats{
			TaskScenarioID:     t.ID,
			Title:              t.Title,
			TaskOrder:          t.TaskOrder,
			TotalAttempts:      0,
			SuccessfulAttempts: 0,
			CompletionRate:     0,
			AvgCompletionTime:  nil,
		})
	}
	return out, nil
}
