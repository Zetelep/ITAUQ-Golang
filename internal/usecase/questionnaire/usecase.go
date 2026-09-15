package questionnaire

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	domain "github.com/itauq-golang/internal/domain/questionnaire"
	repo "github.com/itauq-golang/internal/repository/questionnaire"
)

type Usecase struct {
	repo repo.Repository
}

func NewUsecase(r repo.Repository) *Usecase {
	return &Usecase{repo: r}
}

func (u *Usecase) Create(ctx context.Context, in domain.CreateInput, administratorID string) (*domain.Questionnaire, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.AppName = strings.TrimSpace(in.AppName)
	in.Description = strings.TrimSpace(in.Description)
	in.AppLink = strings.TrimSpace(in.AppLink)
	in.ImgLink = strings.TrimSpace(in.ImgLink)

	if in.Title == "" || in.AppName == "" {
		return nil, errors.New("title and app_name are required")
	}
	if err := validateURL(in.AppLink); err != nil {
		return nil, errors.New("app_link: " + err.Error())
	}
	if err := validateURL(in.ImgLink); err != nil {
		return nil, errors.New("img_link: " + err.Error())
	}

	if in.Status == "" {
		in.Status = domain.StatusDraft
	}
	if in.ItauqVersion == "" {
		in.ItauqVersion = "itauq-v1"
	}

	now := time.Now().UTC()
	q := &domain.Questionnaire{
		ID:              uuid.NewString(),
		AdministratorID: administratorID,
		Title:           in.Title,
		AppName:         in.AppName,
		Description:     in.Description,
		ItauqVersion:    in.ItauqVersion,
		Status:          in.Status,
		AppLink:         in.AppLink,
		ImgLink:         in.ImgLink,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := u.repo.Create(ctx, q); err != nil {
		return nil, err
	}
	return q, nil
}

func (u *Usecase) List(ctx context.Context, callerID, callerRole string, status domain.Status, administratorID string, page, pageSize int) ([]domain.Questionnaire, domain.Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	f := repo.ListFilter{
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	if callerRole == "administrator" {
		f.AdministratorID = callerID
	} else if administratorID != "" {
		f.AdministratorID = administratorID
	}

	items, total, err := u.repo.List(ctx, f)
	if err != nil {
		return nil, domain.Page{}, err
	}

	totalPages := (total + pageSize - 1) / pageSize
	return items, domain.Page{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages}, nil
}

func (u *Usecase) Get(ctx context.Context, id, callerID, callerRole string) (*domain.Questionnaire, error) {
	q, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if callerRole == "administrator" && q.AdministratorID != callerID {
		return nil, repo.ErrForbidden
	}
	return q, nil
}

func (u *Usecase) Update(ctx context.Context, id string, in domain.UpdateInput, callerID string) (*domain.Questionnaire, error) {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing.AdministratorID != callerID {
		return nil, repo.ErrForbidden
	}
	if in.AppLink != nil {
		v := strings.TrimSpace(*in.AppLink)
		if err := validateURL(v); err != nil {
			return nil, errors.New("app_link: " + err.Error())
		}
		in.AppLink = &v
	}
	if in.ImgLink != nil {
		v := strings.TrimSpace(*in.ImgLink)
		if err := validateURL(v); err != nil {
			return nil, errors.New("img_link: " + err.Error())
		}
		in.ImgLink = &v
	}
	return u.repo.Update(ctx, id, in)
}

func (u *Usecase) Delete(ctx context.Context, id, callerID string) error {
	existing, err := u.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if existing.AdministratorID != callerID {
		return repo.ErrForbidden
	}
	return u.repo.Delete(ctx, id)
}

func validateURL(raw string) error {
	if raw == "" {
		return nil
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("must be a valid http/https URL")
	}
	return nil
}
