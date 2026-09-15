// Package evaluation orchestrates the read-only "Evaluation Results & Reports"
// feature described in API_SPECIFICATION.md §10. It owns scoring logic for
// ITAUQ (per-category averages, overall x-bar) and SUS, and enforces
// administrator-vs-super-admin scoping via the callerID/callerRole arguments.
package evaluation

import (
	"context"
	"errors"
	"fmt"

	domain "github.com/itauq-golang/internal/domain/evaluation"
	repo "github.com/itauq-golang/internal/repository/evaluation"
	"github.com/itauq-golang/pkg/itauq"
	"github.com/itauq-golang/pkg/sus"
)

// Errors returned to the handler layer. The handler maps these to HTTP codes.
var (
	ErrNotFound  = errors.New("evaluation resource not found")
	ErrForbidden = errors.New("not authorized to access this evaluation result")
)

// Usecase orchestrates scoring and access checks for the evaluation reports.
type Usecase struct {
	repo      repo.Repository
	itauqInst *itauq.Loaded
	susInst   *sus.Loaded
}

func NewUsecase(r repo.Repository, itauqInst *itauq.Loaded, susInst *sus.Loaded) *Usecase {
	return &Usecase{repo: r, itauqInst: itauqInst, susInst: susInst}
}

// ListRespondents returns a page of respondents with their computed scores,
// scoped by role: administrator sees only their own, super_admin can filter
// by administrator_id or questionnaire_id.
func (u *Usecase) ListRespondents(
	ctx context.Context,
	callerID, callerRole, administratorID, questionnaireID string,
	page, pageSize int,
) ([]domain.RespondentSummary, domain.Page, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	f := repo.ListFilter{
		QuestionnaireID: questionnaireID,
		Page:            page,
		PageSize:        pageSize,
	}

	switch callerRole {
	case "administrator":
		// Force scope to the caller; ignore any caller-supplied
		// administrator_id so an admin can't peek at another's data.
		f.AdministratorID = callerID
	case "super_admin":
		f.AdministratorID = administratorID
	default:
		return nil, domain.Page{}, ErrForbidden
	}

	rows, total, err := u.repo.ListRespondents(ctx, f)
	if err != nil {
		return nil, domain.Page{}, fmt.Errorf("list respondents: %w", err)
	}

	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.RespondentID
	}
	allAnswers, err := u.repo.BatchListAnswers(ctx, ids)
	if err != nil {
		return nil, domain.Page{}, fmt.Errorf("batch list answers: %w", err)
	}
	allSUS, err := u.repo.BatchListSUSAnswers(ctx, ids)
	if err != nil {
		return nil, domain.Page{}, fmt.Errorf("batch list sus answers: %w", err)
	}
	allAttempts, err := u.repo.BatchListTaskAttempts(ctx, ids)
	if err != nil {
		return nil, domain.Page{}, fmt.Errorf("batch list task attempts: %w", err)
	}

	out := make([]domain.RespondentSummary, 0, len(rows))
	for _, row := range rows {
		summary := u.buildSummaryFromBatch(row, allAnswers[row.RespondentID], allSUS[row.RespondentID], allAttempts[row.RespondentID])
		out = append(out, summary)
	}

	totalPages := (total + pageSize - 1) / pageSize
	return out, domain.Page{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages}, nil
}

// GetRespondentDetail returns the full per-respondent report for the
// evaluation results detail page. Caller must be the owning administrator or
// super_admin.
func (u *Usecase) GetRespondentDetail(ctx context.Context, id, callerID, callerRole string) (*domain.RespondentDetail, error) {
	row, err := u.repo.GetRespondent(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrRespondentNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get respondent: %w", err)
	}
	if !u.canAccess(row.AdministratorID, callerID, callerRole) {
		return nil, ErrForbidden
	}

	answers, err := u.repo.ListAnswers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list answers: %w", err)
	}
	susAnswers, err := u.repo.ListSUSAnswers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list sus answers: %w", err)
	}
	attempts, err := u.repo.ListTaskAttempts(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("list task attempts: %w", err)
	}

	stats := itauq.ComputeCategoryStats(u.itauqInst, answers)
	catScores := make([]domain.CategoryScore, 0, len(stats))
	perCat := make(map[string]float64, len(stats))
	for _, s := range stats {
		catScores = append(catScores, domain.CategoryScore{
			Category:        s.Category,
			AvgRawScore:     s.AvgRawScore,
			NormalizedScore: s.NormalizedScore,
		})
		perCat[s.Category] = s.AvgRawScore
	}

	var overall *float64
	if _, norm, ok := itauq.OverallScore(perCat, u.itauqInst.Instrument); ok {
		overall = &norm
	}

	taskResults := make([]domain.TaskResult, 0, len(attempts))
	for _, a := range attempts {
		taskResults = append(taskResults, domain.TaskResult{
			TaskScenarioID:  a.TaskScenarioID,
			Title:           a.Title,
			IsSuccess:       a.IsSuccess,
			DurationSeconds: a.DurationSeconds,
		})
	}

	susBlock := domain.SUSBlock{AnsweredItems: len(susAnswers)}
	if score, _, err := sus.ComputeScore(u.susInst, susAnswers); err == nil {
		susBlock.SUSScore = &score
	}

	identity := domain.RespondentIdentity{
		ID:   row.RespondentID,
		Name: row.RespondentName,
	}
	idRow, err := u.repo.GetRespondentIdentity(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get respondent identity: %w", err)
	}
	identity.Age = idRow.Age
	identity.Gender = idRow.Gender
	identity.Occupation = idRow.Occupation

	return &domain.RespondentDetail{
		Respondent:            identity,
		OverallUsabilityScore: overall,
		CategoryScores:        catScores,
		TaskResults:           taskResults,
		EvaluationWebsiteSUS:  susBlock,
	}, nil
}

// GetQuestionnaireReport aggregates across all respondents of one
// questionnaire. Caller must be the owning administrator or super_admin.
func (u *Usecase) GetQuestionnaireReport(ctx context.Context, questionnaireID, callerID, callerRole string) (*domain.QuestionnaireReport, error) {
	q, err := u.repo.GetQuestionnaire(ctx, questionnaireID)
	if err != nil {
		if errors.Is(err, repo.ErrQuestionnaireNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get questionnaire: %w", err)
	}
	if !u.canAccess(q.AdministratorID, callerID, callerRole) {
		return nil, ErrForbidden
	}

	respondentIDs, err := u.repo.ListRespondentIDs(ctx, questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("list respondent ids: %w", err)
	}

	allAnswers, err := u.repo.BatchListAnswers(ctx, respondentIDs)
	if err != nil {
		return nil, fmt.Errorf("batch list answers: %w", err)
	}
	allSUS, err := u.repo.BatchListSUSAnswers(ctx, respondentIDs)
	if err != nil {
		return nil, fmt.Errorf("batch list sus answers: %w", err)
	}
	allAttempts, err := u.repo.BatchListTaskAttempts(ctx, respondentIDs)
	if err != nil {
		return nil, fmt.Errorf("batch list task attempts: %w", err)
	}

	perCatNorm := make(map[string]float64)
	perCatCount := make(map[string]int)
	var overallSum float64
	var overallCount int
	var susSum float64
	var susCount int
	var successSum float64
	var successCountKnown int

	for _, rid := range respondentIDs {
		answers := allAnswers[rid]
		stats := itauq.ComputeCategoryStats(u.itauqInst, answers)
		if len(stats) == 0 {
			continue
		}
		perCat := make(map[string]float64, len(stats))
		for _, s := range stats {
			perCat[s.Category] = s.AvgRawScore
			perCatNorm[s.Category] += s.NormalizedScore
			perCatCount[s.Category]++
		}
		if _, norm, ok := itauq.OverallScore(perCat, u.itauqInst.Instrument); ok {
			overallSum += norm
			overallCount++
		}

		if score, _, err := sus.ComputeScore(u.susInst, allSUS[rid]); err == nil {
			susSum += score
			susCount++
		}

		attempts := allAttempts[rid]
		if len(attempts) > 0 {
			known := 0
			successes := 0
			for _, a := range attempts {
				if a.IsSuccess != nil {
					known++
					if *a.IsSuccess {
						successes++
					}
				}
			}
			if known > 0 {
				successSum += float64(successes) / float64(known) * 100.0
				successCountKnown++
			}
		}
	}

	cats := make([]string, 0, len(perCatNorm))
	for c := range perCatNorm {
		cats = append(cats, c)
	}
	sortStrings(cats)
	catAverages := make([]domain.CategoryAverage, 0, len(cats))
	for _, c := range cats {
		avg := perCatNorm[c] / float64(perCatCount[c])
		catAverages = append(catAverages, domain.CategoryAverage{
			Category:           c,
			NormalizedScoreAvg: round2(avg),
		})
	}

	out := &domain.QuestionnaireReport{
		Questionnaire:    domain.QuestionnaireHeader{ID: q.ID, Title: q.Title, AppName: q.AppName},
		RespondentCount:  len(respondentIDs),
		CategoryAverages: catAverages,
	}
	if overallCount > 0 {
		v := round2(overallSum / float64(overallCount))
		out.OverallUsabilityScoreAvg = &v
	}
	if susCount > 0 {
		v := round2(susSum / float64(susCount))
		out.EvaluationWebsiteSUSScoreAvg = &v
	}
	if successCountKnown > 0 {
		v := round2(successSum / float64(successCountKnown))
		out.TaskSuccessRateAvg = &v
	}
	return out, nil
}

// canAccess centralizes the role check used by both detail endpoints.
func (u *Usecase) canAccess(administratorID, callerID, callerRole string) bool {
	switch callerRole {
	case "super_admin":
		return true
	case "administrator":
		return administratorID == callerID
	default:
		return false
	}
}

// buildSummaryFromBatch builds a RespondentSummary from pre-fetched batch data.
func (u *Usecase) buildSummaryFromBatch(row repo.RespondentRow, answers map[int]int, susAns map[int]int, tasks []repo.TaskAttemptRow) domain.RespondentSummary {
	stats := itauq.ComputeCategoryStats(u.itauqInst, answers)
	perCat := make(map[string]float64, len(stats))
	for _, s := range stats {
		perCat[s.Category] = s.AvgRawScore
	}
	_, overall, ok := itauq.OverallScore(perCat, u.itauqInst.Instrument)

	susScore, _, _ := sus.ComputeScore(u.susInst, susAns)

	var successPct *float64
	if len(tasks) > 0 {
		known := 0
		successes := 0
		for _, t := range tasks {
			if t.IsSuccess != nil {
				known++
				if *t.IsSuccess {
					successes++
				}
			}
		}
		if known > 0 {
			v := round2(float64(successes) / float64(known) * 100.0)
			successPct = &v
		}
	}

	s := domain.RespondentSummary{
		RespondentID:       row.RespondentID,
		RespondentName:     row.RespondentName,
		QuestionnaireTitle: row.QuestionnaireTitle,
		AppName:            row.AppName,
		AdministratorName:  row.AdministratorName,
		SubmittedAt:        row.SubmittedAt,
	}
	if ok {
		s.OverallUsabilityScore = &overall
	}
	if len(susAns) > 0 {
		v := susScore
		s.EvaluationWebsiteSUSScore = &v
	}
	s.TaskSuccessRatePct = successPct
	return s
}

// round2 rounds to two decimals. Duplicated here to avoid importing pkg/itauq
// for an unexported helper.
func round2(v float64) float64 {
	if v != v {
		return 0
	}
	return float64(int64(v*100+0.5)) / 100.0
}

// sortStrings is an insertion sort. Used instead of "sort" to keep the file's
// import surface minimal.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
