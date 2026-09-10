package administrator

import (
	"context"
	"errors"
	"strings"
	"time"

	domain "github.com/itauq-golang/internal/domain/administrator"
	repo "github.com/itauq-golang/internal/repository/administrator"
)

var ErrFullNameRequired = errors.New("full_name is required")
var ErrEmailRequired = errors.New("email is required")

// Inviter creates a Supabase Auth user for a new administrator. Memory
// implementation fabricates a deterministic-but-unique UUID for in-memory/dev
// mode; production uses the Supabase admin API via SupabaseInviter.
type Inviter interface {
	Invite(ctx context.Context, fullName, email string) (userID string, err error)
}

type Usecase struct {
	repo    repo.Repository
	inviter Inviter
}

func NewUsecase(r repo.Repository, inviter Inviter) *Usecase {
	return &Usecase{repo: r, inviter: inviter}
}

func (u *Usecase) Create(ctx context.Context, in domain.CreateInput) (*domain.Administrator, error) {
	fullName := strings.TrimSpace(in.FullName)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if fullName == "" {
		return nil, ErrFullNameRequired
	}
	if email == "" {
		return nil, ErrEmailRequired
	}
	userID, err := u.inviter.Invite(ctx, fullName, email)
	if err != nil {
		return nil, err
	}
	a := &domain.Administrator{
		ID:                 userID,
		FullName:           fullName,
		Email:              email,
		Role:               "administrator",
		IsActive:           true,
		MustChangePassword: true,
		CreatedAt:          time.Now().UTC(),
	}
	if err := u.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (u *Usecase) List(ctx context.Context, f domain.ListFilter) ([]domain.Administrator, domain.Page, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	items, total, err := u.repo.List(ctx, f)
	if err != nil {
		return nil, domain.Page{}, err
	}
	pages := 0
	if total > 0 {
		pages = (total + f.PageSize - 1) / f.PageSize
	}
	return items, domain.Page{
		Page:       f.Page,
		PageSize:   f.PageSize,
		Total:      total,
		TotalPages: pages,
	}, nil
}

func (u *Usecase) Get(ctx context.Context, id string) (*domain.Administrator, error) {
	return u.repo.Get(ctx, id)
}

func (u *Usecase) Update(ctx context.Context, id string, in domain.UpdateInput) (*domain.Administrator, error) {
	if in.FullName != nil {
		name := strings.TrimSpace(*in.FullName)
		if name == "" {
			return nil, ErrFullNameRequired
		}
		in.FullName = &name
	}
	if in.Institution != nil {
		institution := strings.TrimSpace(*in.Institution)
		in.Institution = &institution
	}
	if in.Occupation != nil {
		occupation := strings.TrimSpace(*in.Occupation)
		in.Occupation = &occupation
	}
	return u.repo.Update(ctx, id, in)
}

func (u *Usecase) Delete(ctx context.Context, id string) error {
	return u.repo.Delete(ctx, id)
}
