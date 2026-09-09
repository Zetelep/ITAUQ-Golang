package eligibility

import (
	"context"
	"errors"
	"sort"
	"sync"

	domain "github.com/itauq-golang/internal/domain/eligibility"
)

// ErrNotFound is returned when a criterion id does not exist. Maps to HTTP 404.
var ErrNotFound = errors.New("eligibility criterion not found")

// Repository is the persistence boundary for the eligibility feature.
// Criteria ops are admin-facing; ListIDsForQuestionnaire and
// CreateConfirmations are used by the public-flow gate.
type Repository interface {
	Create(ctx context.Context, c *domain.EligibilityCriterion) error
	ListByQuestionnaire(ctx context.Context, questionnaireID string) ([]domain.EligibilityCriterion, error)
	ListIDsForQuestionnaire(ctx context.Context, questionnaireID string) ([]string, error)
	Get(ctx context.Context, id string) (*domain.EligibilityCriterion, error)
	Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.EligibilityCriterion, error)
	Delete(ctx context.Context, id string) error
	CreateConfirmations(ctx context.Context, confs []domain.Confirmation) error
}

// MemoryRepository is the in-memory implementation used in dev mode
// (no DATABASE_URL set). Confirmation rows are stored separately so the
// count matches what the public flow's gate would validate against.
type MemoryRepository struct {
	mu           sync.RWMutex
	criteria     map[string]*domain.EligibilityCriterion
	confirmations map[string]*domain.Confirmation
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		criteria:      make(map[string]*domain.EligibilityCriterion),
		confirmations: make(map[string]*domain.Confirmation),
	}
}

func (r *MemoryRepository) Create(_ context.Context, c *domain.EligibilityCriterion) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.criteria[c.ID] = &cp
	return nil
}

func (r *MemoryRepository) ListByQuestionnaire(_ context.Context, questionnaireID string) ([]domain.EligibilityCriterion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]domain.EligibilityCriterion, 0)
	for _, c := range r.criteria {
		if c.QuestionnaireID == questionnaireID {
			all = append(all, *c)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CriteriaOrder == all[j].CriteriaOrder {
			return all[i].CreatedAt.Before(all[j].CreatedAt)
		}
		return all[i].CriteriaOrder < all[j].CriteriaOrder
	})
	return all, nil
}

func (r *MemoryRepository) ListIDsForQuestionnaire(_ context.Context, questionnaireID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := make([]string, 0)
	for _, c := range r.criteria {
		if c.QuestionnaireID == questionnaireID {
			ids = append(ids, c.ID)
		}
	}
	return ids, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (*domain.EligibilityCriterion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.criteria[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *MemoryRepository) Update(_ context.Context, id string, in domain.UpdateInput) (*domain.EligibilityCriterion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.criteria[id]
	if !ok {
		return nil, ErrNotFound
	}
	if in.Statement != nil {
		c.Statement = *in.Statement
	}
	if in.CriteriaOrder != nil {
		c.CriteriaOrder = *in.CriteriaOrder
	}
	cp := *c
	return &cp, nil
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.criteria[id]; !ok {
		return ErrNotFound
	}
	delete(r.criteria, id)
	return nil
}

func (r *MemoryRepository) CreateConfirmations(_ context.Context, confs []domain.Confirmation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range confs {
		cp := confs[i]
		r.confirmations[cp.ID] = &cp
	}
	return nil
}
