package administrator

import (
	"context"
	"errors"
	"strings"
	"sync"

	domain "github.com/itauq-golang/internal/domain/administrator"
)

var ErrNotFound = errors.New("administrator not found")

type Repository interface {
	Create(context.Context, *domain.Administrator) error
	List(context.Context, domain.ListFilter) ([]domain.Administrator, int, error)
	Get(context.Context, string) (*domain.Administrator, error)
	Update(context.Context, string, domain.UpdateInput) (*domain.Administrator, error)
	Delete(context.Context, string) error
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Administrator
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.Administrator)}
}

func (r *MemoryRepository) Create(_ context.Context, a *domain.Administrator) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.items {
		if strings.EqualFold(x.Email, a.Email) {
			return errors.New("administrator with this email already exists")
		}
	}
	cp := *a
	r.items[a.ID] = &cp
	return nil
}

func (r *MemoryRepository) List(_ context.Context, f domain.ListFilter) ([]domain.Administrator, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.Administrator, 0)
	for _, x := range r.items {
		if f.IsActive != nil && x.IsActive != *f.IsActive {
			continue
		}
		all = append(all, *x)
	}
	total := len(all)
	page := f.Page
	size := f.PageSize
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	start := (page - 1) * size
	if start >= total {
		return []domain.Administrator{}, total, nil
	}
	end := start + size
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.Administrator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *x
	return &cp, nil
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.Administrator, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	x, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.FullName != nil {
		x.FullName = *in.FullName
	}
	if in.Institution != nil {
		x.Institution = *in.Institution
	}
	if in.Occupation != nil {
		x.Occupation = *in.Occupation
	}
	if in.IsActive != nil {
		x.IsActive = *in.IsActive
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
