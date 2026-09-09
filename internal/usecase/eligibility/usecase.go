// Package eligibility hosts the admin-facing CRUD for the
// "Terms & Conditions checklist" attached to a questionnaire. It mirrors
// the task-scenario usecase: every operation goes through
// checkOwnership on the parent questionnaire first, so non-owning
// administrators get 403 and unknown questionnaires get 404. See
// API_SPECIFICATION.md §6b.
package eligibility

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/eligibility"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	repo "github.com/itauq-golang/internal/repository/eligibility"
)

type Usecase struct {
	repo      repo.Repository
	questRepo qRepo.Repository
}

func NewUsecase(r repo.Repository, qr qRepo.Repository) *Usecase {
	return &Usecase{repo: r, questRepo: qr}
}

func (u *Usecase) checkOwnership(ctx context.Context, questionnaireID, callerID, callerRole string) error {
	q, err := u.questRepo.Get(ctx, questionnaireID)
	if err != nil {
		return err
	}
	if callerRole == "administrator" && q.AdministratorID != callerID {
		return qRepo.ErrForbidden
	}
	return nil
}

func (u *Usecase) Create(ctx context.Context, questionnaireID string, in domain.CreateInput, callerID, callerRole string) (*domain.EligibilityCriterion, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}

	in.Statement = strings.TrimSpace(in.Statement)
	if in.Statement == "" {
		return nil, errors.New("statement is required")
	}

	now := time.Now().UTC()
	c := &domain.EligibilityCriterion{
		ID:              uuid.NewString(),
		QuestionnaireID: questionnaireID,
		Statement:       in.Statement,
		CriteriaOrder:   in.CriteriaOrder,
		CreatedAt:       now,
	}

	if err := u.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *Usecase) ListByQuestionnaire(ctx context.Context, questionnaireID, callerID, callerRole string) ([]domain.EligibilityCriterion, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return u.repo.ListByQuestionnaire(ctx, questionnaireID)
}

func (u *Usecase) Get(ctx context.Context, id, callerID, callerRole string) (*domain.EligibilityCriterion, error) {
	c, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.checkOwnership(ctx, c.QuestionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return c, nil
}

func (u *Usecase) Update(ctx context.Context, id string, in domain.UpdateInput, callerID string) (*domain.EligibilityCriterion, error) {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	q, err := u.questRepo.Get(ctx, existing.QuestionnaireID)
	if err != nil {
		return nil, err
	}
	if q.AdministratorID != callerID {
		return nil, qRepo.ErrForbidden
	}
	if in.Statement != nil {
		trimmed := strings.TrimSpace(*in.Statement)
		if trimmed == "" {
			return nil, errors.New("statement must not be blank")
		}
		in.Statement = &trimmed
	}
	return u.repo.Update(ctx, id, in)
}

func (u *Usecase) Delete(ctx context.Context, id, callerID string) error {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	q, err := u.questRepo.Get(ctx, existing.QuestionnaireID)
	if err != nil {
		return err
	}
	if q.AdministratorID != callerID {
		return qRepo.ErrForbidden
	}
	return u.repo.Delete(ctx, id)
}
