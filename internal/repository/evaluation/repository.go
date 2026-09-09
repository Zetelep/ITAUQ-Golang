package evaluation

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// Errors shared by both backends. Mapped to HTTP codes by the handler layer.
var (
	ErrRespondentNotFound = errors.New("respondent not found")
	ErrQuestionnaireNotFound = errors.New("questionnaire not found")
)

// RespondentRow is the flat row used to build RespondentSummary. Scores are
// pointers so "not yet computed / no answers submitted" is distinguishable
// from a literal 0.
type RespondentRow struct {
	RespondentID        string
	RespondentName      string
	QuestionnaireID     string
	QuestionnaireTitle  string
	AppName             string
	AdministratorID     string
	AdministratorName   string
	SubmittedAt         *time.Time
}

// ListFilter scopes the respondent list query.
type ListFilter struct {
	// AdministratorID restricts results to one owning administrator. Pass an
	// empty string for super-admin (no restriction).
	AdministratorID string
	// QuestionnaireID optionally narrows by questionnaire.
	QuestionnaireID string
	Page            int
	PageSize        int
}

// RespondentAnswers holds the per-respondent inputs needed to compute scores.
type RespondentAnswers struct {
	Answers    map[int]int // item_id -> score
	SUSAnswers map[int]int // item_id -> score
}

// TaskAttemptRow is a single task attempt as returned by the repository.
type TaskAttemptRow struct {
	TaskScenarioID  string
	Title           string
	IsSuccess       *bool
	DurationSeconds *int
}

// Repository is the storage contract used by the evaluation usecase.
type Repository interface {
	// ListRespondents returns a page of respondents with the data needed to
	// build RespondentSummary. The second return is the total row count.
	ListRespondents(context.Context, ListFilter) ([]RespondentRow, int, error)
	// GetRespondent returns the identity row or ErrRespondentNotFound.
	GetRespondent(context.Context, string) (*RespondentRow, error)
	// ListAnswers returns all 30 (or fewer) item_id -> score answers.
	ListAnswers(context.Context, string) (map[int]int, error)
	// ListSUSAnswers returns all 10 item_id -> score answers (or fewer).
	ListSUSAnswers(context.Context, string) (map[int]int, error)
	// ListTaskAttempts returns the attempts joined with their scenario titles.
	ListTaskAttempts(context.Context, string) ([]TaskAttemptRow, error)
	// GetQuestionnaire returns the questionnaire identity or
	// ErrQuestionnaireNotFound.
	GetQuestionnaire(context.Context, string) (*QuestionnaireInfo, error)
	// ListRespondentIDs returns all respondent IDs for a questionnaire. Used by
	// the report aggregator.
	ListRespondentIDs(context.Context, string) ([]string, error)
	// GetRespondentIdentity returns age/gender/occupation for the detail
	// endpoint. ErrRespondentNotFound if the id is unknown.
	GetRespondentIdentity(context.Context, string) (*RespondentIdentity, error)
}

// RespondentIdentity is the small identity slice needed for the detail page.
type RespondentIdentity struct {
	Age        *int
	Gender     *string
	Occupation string
}

// QuestionnaireInfo is the minimal questionnaire payload needed for headers.
type QuestionnaireInfo struct {
	ID              string
	Title           string
	AppName         string
	AdministratorID string
}

// --- in-memory implementation (used when DATABASE_URL is not set) ---

// MemoryDataSource exposes the underlying tables so the main wiring can hand
// the same in-memory respondents/answers/etc. over to this repository.
type MemoryDataSource struct {
	mu           sync.RWMutex
	Respondents  map[string]*memRespondent
	Answers      map[string]map[int]int // respondent_id -> item_id -> score
	SUS          map[string]map[int]int
	Attempts     map[string][]memAttempt
	Scenarios    map[string]*memScenario // id -> scenario
	ScenariosByQ map[string][]string     // questionnaire_id -> []scenario_id
	Quests       map[string]*memQuest
	Profiles     map[string]*memProfile
	Links        map[string]*memLink // link_id -> questionnaire_id
}

type memRespondent struct {
	ID              string
	EvaluationLinkID string
	Name            string
	Age             *int
	Gender          *string
	Occupation      string
	SubmittedAt     *time.Time
}

type memAttempt struct {
	TaskScenarioID  string
	IsSuccess       *bool
	DurationSeconds *int
}

type memScenario struct {
	ID             string
	QuestionnaireID string
	Title          string
}

type memQuest struct {
	ID              string
	AdministratorID string
	Title           string
	AppName         string
}

type memProfile struct {
	ID       string
	FullName string
}

type memLink struct {
	ID              string
	QuestionnaireID string
}

// NewMemoryDataSource builds an empty data source. main.go wires it up after
// instantiating the other memory repositories.
func NewMemoryDataSource() *MemoryDataSource {
	return &MemoryDataSource{
		Respondents:  make(map[string]*memRespondent),
		Answers:      make(map[string]map[int]int),
		SUS:          make(map[string]map[int]int),
		Attempts:     make(map[string][]memAttempt),
		Scenarios:    make(map[string]*memScenario),
		ScenariosByQ: make(map[string][]string),
		Quests:       make(map[string]*memQuest),
		Profiles:     make(map[string]*memProfile),
		Links:        make(map[string]*memLink),
	}
}

// MemoryRepository implements Repository against a MemoryDataSource.
type MemoryRepository struct{ src *MemoryDataSource }

func NewMemoryRepository(src *MemoryDataSource) *MemoryRepository {
	return &MemoryRepository{src: src}
}

func (r *MemoryRepository) ListRespondents(_ context.Context, f ListFilter) ([]RespondentRow, int, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()

	rows := make([]RespondentRow, 0)
	for _, p := range r.src.Respondents {
		link, ok := r.src.findLinkByID(p.EvaluationLinkID)
		if !ok {
			continue
		}
		q, ok := r.src.Quests[link.QuestionnaireID]
		if !ok {
			continue
		}
		if f.AdministratorID != "" && q.AdministratorID != f.AdministratorID {
			continue
		}
		if f.QuestionnaireID != "" && q.ID != f.QuestionnaireID {
			continue
		}
		adminName := ""
		if prof, ok := r.src.Profiles[q.AdministratorID]; ok {
			adminName = prof.FullName
		}
		rows = append(rows, RespondentRow{
			RespondentID:       p.ID,
			RespondentName:     p.Name,
			QuestionnaireID:    q.ID,
			QuestionnaireTitle: q.Title,
			AppName:            q.AppName,
			AdministratorID:    q.AdministratorID,
			AdministratorName:  adminName,
			SubmittedAt:        p.SubmittedAt,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		ti := time.Time{}
		tj := time.Time{}
		if rows[i].SubmittedAt != nil {
			ti = *rows[i].SubmittedAt
		}
		if rows[j].SubmittedAt != nil {
			tj = *rows[j].SubmittedAt
		}
		return ti.After(tj)
	})

	total := len(rows)
	start := (f.Page - 1) * f.PageSize
	if start >= total {
		return []RespondentRow{}, total, nil
	}
	end := start + f.PageSize
	if end > total {
		end = total
	}
	return rows[start:end], total, nil
}

func (r *MemoryRepository) GetRespondent(_ context.Context, id string) (*RespondentRow, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	p, ok := r.src.Respondents[id]
	if !ok {
		return nil, ErrRespondentNotFound
	}
	link, ok := r.src.findLinkByID(p.EvaluationLinkID)
	if !ok {
		return nil, ErrRespondentNotFound
	}
	q, ok := r.src.Quests[link.QuestionnaireID]
	if !ok {
		return nil, ErrRespondentNotFound
	}
	adminName := ""
	if prof, ok := r.src.Profiles[q.AdministratorID]; ok {
		adminName = prof.FullName
	}
	return &RespondentRow{
		RespondentID:       p.ID,
		RespondentName:     p.Name,
		QuestionnaireID:    q.ID,
		QuestionnaireTitle: q.Title,
		AppName:            q.AppName,
		AdministratorID:    q.AdministratorID,
		AdministratorName:  adminName,
		SubmittedAt:        p.SubmittedAt,
	}, nil
}

func (r *MemoryRepository) ListAnswers(_ context.Context, respondentID string) (map[int]int, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	out := make(map[int]int, len(r.src.Answers[respondentID]))
	for k, v := range r.src.Answers[respondentID] {
		out[k] = v
	}
	return out, nil
}

func (r *MemoryRepository) ListSUSAnswers(_ context.Context, respondentID string) (map[int]int, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	out := make(map[int]int, len(r.src.SUS[respondentID]))
	for k, v := range r.src.SUS[respondentID] {
		out[k] = v
	}
	return out, nil
}

func (r *MemoryRepository) ListTaskAttempts(_ context.Context, respondentID string) ([]TaskAttemptRow, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	src := r.src.Attempts[respondentID]
	out := make([]TaskAttemptRow, 0, len(src))
	for _, a := range src {
		title := ""
		if s, ok := r.src.Scenarios[a.TaskScenarioID]; ok {
			title = s.Title
		}
		out = append(out, TaskAttemptRow{
			TaskScenarioID:  a.TaskScenarioID,
			Title:           title,
			IsSuccess:       a.IsSuccess,
			DurationSeconds: a.DurationSeconds,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TaskScenarioID < out[j].TaskScenarioID })
	return out, nil
}

func (r *MemoryRepository) GetQuestionnaire(_ context.Context, id string) (*QuestionnaireInfo, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	q, ok := r.src.Quests[id]
	if !ok {
		return nil, ErrQuestionnaireNotFound
	}
	return &QuestionnaireInfo{ID: q.ID, Title: q.Title, AppName: q.AppName, AdministratorID: q.AdministratorID}, nil
}

func (r *MemoryRepository) ListRespondentIDs(_ context.Context, questionnaireID string) ([]string, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	ids := make([]string, 0)
	for _, p := range r.src.Respondents {
		link, ok := r.src.findLinkByID(p.EvaluationLinkID)
		if !ok || link.QuestionnaireID != questionnaireID {
			continue
		}
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	return ids, nil
}

func (r *MemoryRepository) GetRespondentIdentity(_ context.Context, id string) (*RespondentIdentity, error) {
	r.src.mu.RLock()
	defer r.src.mu.RUnlock()
	p, ok := r.src.Respondents[id]
	if !ok {
		return nil, ErrRespondentNotFound
	}
	return &RespondentIdentity{
		Age:        p.Age,
		Gender:     p.Gender,
		Occupation: p.Occupation,
	}, nil
}

// LinkRecord is the in-memory representation of an evaluation link, kept in
// the data source only to navigate respondent -> questionnaire.
type LinkRecord struct {
	ID              string
	QuestionnaireID string
}

func (s *MemoryDataSource) findLinkByID(id string) (LinkRecord, bool) {
	l, ok := s.Links[id]
	if !ok {
		return LinkRecord{}, false
	}
	return LinkRecord{ID: l.ID, QuestionnaireID: l.QuestionnaireID}, true
}
