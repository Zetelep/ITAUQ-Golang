package questionnaire

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	domain "github.com/itauq-golang/internal/domain/questionnaire"
)

var ErrNotFound = errors.New("questionnaire not found")
var ErrForbidden = errors.New("not authorized to access this questionnaire")

type ListFilter struct {
	AdministratorID string
	Status          domain.Status
	Page            int
	PageSize        int
}

type Repository interface {
	Create(context.Context, *domain.Questionnaire) error
	List(context.Context, ListFilter) ([]domain.Questionnaire, int, error)
	Get(context.Context, string) (*domain.Questionnaire, error)
	Update(context.Context, string, domain.UpdateInput) (*domain.Questionnaire, error)
	Delete(context.Context, string) error
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Questionnaire
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.Questionnaire)}
}

func (r *MemoryRepository) Create(_ context.Context, q *domain.Questionnaire) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *q
	r.items[q.ID] = &cp
	return nil
}

func (r *MemoryRepository) List(_ context.Context, f ListFilter) ([]domain.Questionnaire, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.Questionnaire, 0)
	for _, x := range r.items {
		if f.AdministratorID != "" && x.AdministratorID != f.AdministratorID {
			continue
		}
		if f.Status != "" && x.Status != f.Status {
			continue
		}
		all = append(all, *x)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	total := len(all)
	start := (f.Page - 1) * f.PageSize
	if start >= total {
		return []domain.Questionnaire{}, total, nil
	}
	end := start + f.PageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.Questionnaire, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.Questionnaire, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.Title != nil {
		x.Title = *in.Title
	}
	if in.AppName != nil {
		x.AppName = *in.AppName
	}
	if in.Description != nil {
		x.Description = *in.Description
	}
	if in.Status != nil {
		x.Status = *in.Status
	}
	if in.AppLink != nil {
		x.AppLink = *in.AppLink
	}
	if in.ImgLink != nil {
		x.ImgLink = *in.ImgLink
	}
	x.UpdatedAt = time.Now().UTC()
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
