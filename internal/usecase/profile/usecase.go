package profile

import (
	"context"
	"errors"
	"strings"

	domain "github.com/itauq-golang/internal/domain/profile"
	repo "github.com/itauq-golang/internal/repository/profile"
)

var ErrFullNameRequired = errors.New("full_name is required")

type PasswordUpdater interface {
	UpdatePassword(ctx context.Context, userID, newPassword string) error
}

type Usecase struct {
	repo            repo.Repository
	passwordUpdater PasswordUpdater
}

func NewUsecase(r repo.Repository, p PasswordUpdater) *Usecase {
	return &Usecase{repo: r, passwordUpdater: p}
}

func (u *Usecase) GetMe(ctx context.Context, userID string) (*domain.Profile, error) {
	return u.repo.GetByID(ctx, userID)
}

func (u *Usecase) UpdateMe(ctx context.Context, userID string, in domain.UpdateInput) (*domain.Profile, error) {
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
	return u.repo.Update(ctx, userID, in)
}

func (u *Usecase) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	if u.passwordUpdater == nil {
		return errors.New("password updates are not available in dev/in-memory mode")
	}
	if err := u.passwordUpdater.UpdatePassword(ctx, userID, newPassword); err != nil {
		return err
	}
	return u.repo.ClearMustChangePassword(ctx, userID)
}
