package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/application"
	repo "github.com/itauq-golang/internal/repository/application"
)

type Administrator struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}
type Approval struct {
	Application   *domain.Application `json:"application"`
	Administrator Administrator       `json:"administrator"`
}
type Provisioner interface {
	InviteAdministrator(context.Context, *domain.Application) (Administrator, error)
}
type MemoryProvisioner struct{}

func (MemoryProvisioner) InviteAdministrator(_ context.Context, a *domain.Application) (Administrator, error) {
	return Administrator{ID: uuid.NewString(), Email: a.Email, Role: "administrator"}, nil
}

type Usecase struct {
	repo        repo.Repository
	provisioner Provisioner
}

func NewUsecase(r repo.Repository, p Provisioner) *Usecase { return &Usecase{repo: r, provisioner: p} }
func (u *Usecase) Create(ctx context.Context, in domain.CreateInput) (*domain.Application, error) {
	in.FullName = strings.TrimSpace(in.FullName)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Institution = strings.TrimSpace(in.Institution)
	in.Occupation = strings.TrimSpace(in.Occupation)
	in.Reason = strings.TrimSpace(in.Reason)
	if in.FullName == "" {
		return nil, errors.New("all fields are required")
	}
	a := &domain.Application{ID: uuid.NewString(), FullName: in.FullName, Email: in.Email, Institution: in.Institution, Occupation: in.Occupation, Reason: in.Reason, Status: domain.StatusPending, CreatedAt: time.Now().UTC()}
	if err := u.repo.Create(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}
func (u *Usecase) List(ctx context.Context, status domain.Status, page, size int) ([]domain.Application, domain.Page, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := u.repo.List(ctx, status, page, size)
	pages := (total + size - 1) / size
	return items, totalPage(page, size, total, pages), err
}
func totalPage(page, size, total, pages int) domain.Page {
	return domain.Page{Page: page, PageSize: size, Total: total, TotalPages: pages}
}
func (u *Usecase) Get(ctx context.Context, id string) (*domain.Application, error) {
	return u.repo.Get(ctx, id)
}
func (u *Usecase) Approve(ctx context.Context, id, note, reviewerID string) (*Approval, error) {
	a, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Status != domain.StatusPending {
		return nil, repo.ErrDuplicatePending
	}
	admin, err := u.provisioner.InviteAdministrator(ctx, a)
	if err != nil {
		return nil, err
	}
	updated, err := u.repo.UpdateReview(ctx, id, domain.StatusApproved, note, reviewerID)
	if err != nil {
		return nil, err
	}
	return &Approval{Application: updated, Administrator: admin}, nil
}
func (u *Usecase) Reject(ctx context.Context, id, note, reviewerID string) (*domain.Application, error) {
	a, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if a.Status != domain.StatusPending {
		return nil, repo.ErrDuplicatePending
	}
	return u.repo.UpdateReview(ctx, id, domain.StatusRejected, note, reviewerID)
}
