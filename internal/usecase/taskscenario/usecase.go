package taskscenario

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/taskscenario"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
	repo "github.com/itauq-golang/internal/repository/taskscenario"
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

func (u *Usecase) Create(ctx context.Context, questionnaireID string, in domain.CreateInput, callerID, callerRole string) (*domain.TaskScenario, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}

	in.Title = strings.TrimSpace(in.Title)
	in.Instruction = strings.TrimSpace(in.Instruction)

	if in.Title == "" || in.Instruction == "" {
		return nil, errors.New("title and instruction are required")
	}

	now := time.Now().UTC()
	t := &domain.TaskScenario{
		ID:              uuid.NewString(),
		QuestionnaireID: questionnaireID,
		Title:           in.Title,
		Instruction:     in.Instruction,
		TaskOrder:       in.TaskOrder,
		CreatedAt:       now,
	}

	if err := u.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (u *Usecase) ListByQuestionnaire(ctx context.Context, questionnaireID, callerID, callerRole string) ([]domain.TaskScenario, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return u.repo.ListByQuestionnaire(ctx, questionnaireID)
}

func (u *Usecase) Get(ctx context.Context, id, callerID, callerRole string) (*domain.TaskScenario, error) {
	t, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.checkOwnership(ctx, t.QuestionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return t, nil
}

func (u *Usecase) Update(ctx context.Context, id string, in domain.UpdateInput, callerID string) (*domain.TaskScenario, error) {
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

func (u *Usecase) GetStats(ctx context.Context, questionnaireID, callerID, callerRole string) (*domain.QuestionnaireTaskScenarioStats, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	stats, err := u.repo.GetStats(ctx, questionnaireID)
	if err != nil {
		return nil, err
	}
	return &domain.QuestionnaireTaskScenarioStats{
		QuestionnaireID: questionnaireID,
		TaskScenarios:   stats,
	}, nil
}
