package application

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	domain "github.com/itauq-golang/internal/domain/application"
)

var ErrNotFound = errors.New("application not found")
var ErrDuplicatePending = errors.New("pending application already exists for this email")

type Repository interface {
	Create(context.Context, *domain.Application) error
	List(context.Context, domain.Status, int, int) ([]domain.Application, int, error)
	Get(context.Context, string) (*domain.Application, error)
	UpdateReview(context.Context, string, domain.Status, string, string) (*domain.Application, error)
}
type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Application
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.Application)}
}
func (r *MemoryRepository) Create(_ context.Context, a *domain.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.items {
		if strings.EqualFold(x.Email, a.Email) && x.Status == domain.StatusPending {
			return ErrDuplicatePending
		}
	}
	cp := *a
	r.items[a.ID] = &cp
	return nil
}
func (r *MemoryRepository) List(_ context.Context, status domain.Status, page, size int) ([]domain.Application, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.Application, 0)
	for _, x := range r.items {
		if status == "" || x.Status == status {
			all = append(all, *x)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	total := len(all)
	start := (page - 1) * size
	if start >= total {
		return []domain.Application{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}
func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *x
	return &cp, nil
}
func (r *MemoryRepository) UpdateReview(_ context.Context, id string, status domain.Status, note, _ string) (*domain.Application, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if x.Status != domain.StatusPending {
		return nil, ErrDuplicatePending
	}
	now := time.Now().UTC()
	x.Status = status
	x.ReviewNote = note
	x.ReviewedAt = &now
	cp := *x
	return &cp, nil
}
