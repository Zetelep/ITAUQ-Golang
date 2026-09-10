package profile

import (
	"context"
	"errors"
	"sync"

	domain "github.com/itauq-golang/internal/domain/profile"
)

var ErrNotFound = errors.New("profile not found")

type Repository interface {
	GetByID(ctx context.Context, id string) (*domain.Profile, error)
	Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.Profile, error)
	ClearMustChangePassword(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu    sync.RWMutex
	items map[string]*domain.Profile
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{items: make(map[string]*domain.Profile)}
}

func (r *MemoryRepository) GetByID(_ context.Context, id string) (*domain.Profile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.FullName != nil {
		p.FullName = *in.FullName
	}
	if in.Institution != nil {
		p.Institution = *in.Institution
	}
	if in.Occupation != nil {
		p.Occupation = *in.Occupation
	}
	cp := *p
	return &cp, nil
}

func (r *MemoryRepository) ClearMustChangePassword(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.items[id]
	if !ok {
		return ErrNotFound
	}
	p.MustChangePassword = false
	return nil
}
