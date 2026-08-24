package evaluationlink

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/evaluationlink"
	repo "github.com/itauq-golang/internal/repository/evaluationlink"
	qRepo "github.com/itauq-golang/internal/repository/questionnaire"
)

const tokenAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
const tokenLength = 8

var ErrInvalidExpiry = errors.New("expires_at must be in the future")

type Usecase struct {
	repo      repo.Repository
	questRepo qRepo.Repository
	baseURL   string
}

func NewUsecase(r repo.Repository, qr qRepo.Repository, baseURL string) *Usecase {
	if baseURL == "" {
		baseURL = "https://itauq.site/e/"
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	return &Usecase{repo: r, questRepo: qr, baseURL: baseURL}
}

func generateToken() (string, error) {
	b := make([]byte, tokenLength)
	max := big.NewInt(int64(len(tokenAlphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = tokenAlphabet[n.Int64()]
	}
	return string(b), nil
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

// requireOwner enforces ownership strictly (no super admin override) for
// mutating operations, matching the API spec.
func (u *Usecase) requireOwner(ctx context.Context, questionnaireID, callerID string) error {
	q, err := u.questRepo.Get(ctx, questionnaireID)
	if err != nil {
		return err
	}
	if q.AdministratorID != callerID {
		return qRepo.ErrForbidden
	}
	return nil
}

func (u *Usecase) Create(ctx context.Context, questionnaireID string, in domain.CreateInput, callerID, callerRole string) (*domain.EvaluationLink, error) {
	if err := u.requireOwner(ctx, questionnaireID, callerID); err != nil {
		return nil, err
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidExpiry
	}

	token, err := generateToken()
	if err != nil {
		return nil, err
	}

	l := &domain.EvaluationLink{
		ID:              uuid.NewString(),
		QuestionnaireID: questionnaireID,
		Token:           token,
		URL:             u.baseURL + token,
		CreatedBy:       callerID,
		IsActive:        true,
		ExpiresAt:       in.ExpiresAt,
		CreatedAt:       time.Now().UTC(),
	}
	if err := u.repo.Create(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (u *Usecase) ListByQuestionnaire(ctx context.Context, questionnaireID, callerID, callerRole string) ([]domain.EvaluationLink, error) {
	if err := u.checkOwnership(ctx, questionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return u.repo.ListByQuestionnaire(ctx, questionnaireID)
}

func (u *Usecase) Get(ctx context.Context, id, callerID, callerRole string) (*domain.EvaluationLink, error) {
	l, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.checkOwnership(ctx, l.QuestionnaireID, callerID, callerRole); err != nil {
		return nil, err
	}
	return l, nil
}

func (u *Usecase) Update(ctx context.Context, id string, in domain.UpdateInput, callerID string) (*domain.EvaluationLink, error) {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := u.requireOwner(ctx, existing.QuestionnaireID, callerID); err != nil {
		return nil, err
	}
	if in.ExpiresAt != nil && !in.ExpiresAt.After(time.Now().UTC()) {
		return nil, ErrInvalidExpiry
	}
	return u.repo.Update(ctx, id, in)
}

func (u *Usecase) Delete(ctx context.Context, id, callerID string) error {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := u.requireOwner(ctx, existing.QuestionnaireID, callerID); err != nil {
		return err
	}
	return u.repo.Delete(ctx, id)
}
