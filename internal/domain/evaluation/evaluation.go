// Package evaluation defines the read-model types returned by the
// "Evaluation Results & Reports" feature. See API_SPECIFICATION.md §10.
package evaluation

import "time"

// Page is the pagination meta block included in list responses.
type Page struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// RespondentSummary is a single row in GET /respondents.
type RespondentSummary struct {
	RespondentID             string     `json:"respondent_id"`
	RespondentName           string     `json:"respondent_name"`
	QuestionnaireTitle       string     `json:"questionnaire_title"`
	AppName                  string     `json:"app_name"`
	AdministratorName        string     `json:"administrator_name"`
	OverallUsabilityScore    *float64   `json:"overall_usability_score"`
	TaskSuccessRatePct       *float64   `json:"task_success_rate_pct"`
	EvaluationWebsiteSUSScore *float64   `json:"evaluation_website_sus_score"`
	SubmittedAt              *time.Time `json:"submitted_at"`
}

// CategoryScore is one row inside the per-respondent category breakdown.
type CategoryScore struct {
	Category        string  `json:"category"`
	AvgRawScore     float64 `json:"avg_raw_score"`
	NormalizedScore float64 `json:"normalized_score"`
}

// TaskResult is one row inside the per-respondent task breakdown.
type TaskResult struct {
	TaskScenarioID  string `json:"task_scenario_id"`
	Title           string `json:"title"`
	IsSuccess       *bool  `json:"is_success"`
	DurationSeconds *int   `json:"duration_seconds"`
}

// SUSBlock is the SUS-of-the-evaluation-website block on a respondent detail.
type SUSBlock struct {
	AnsweredItems int     `json:"answered_items"`
	SUSScore      *float64 `json:"sus_score"`
}

// RespondentIdentity is the embedded respondent identity in the detail response.
type RespondentIdentity struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Age        *int    `json:"age"`
	Gender     *string `json:"gender"`
	Occupation string  `json:"occupation"`
}

// RespondentDetail is the response of GET /respondents/{id}.
type RespondentDetail struct {
	Respondent             RespondentIdentity `json:"respondent"`
	OverallUsabilityScore  *float64           `json:"overall_usability_score"`
	CategoryScores         []CategoryScore    `json:"category_scores"`
	TaskResults            []TaskResult       `json:"task_results"`
	EvaluationWebsiteSUS   SUSBlock           `json:"evaluation_website_sus"`
}

// QuestionnaireHeader is the embedded questionnaire info in the report.
type QuestionnaireHeader struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	AppName string `json:"app_name"`
}

// CategoryAverage is one row inside the report's category_averages.
type CategoryAverage struct {
	Category            string  `json:"category"`
	NormalizedScoreAvg  float64 `json:"normalized_score_avg"`
}

// QuestionnaireReport is the response of GET /questionnaires/{id}/report.
type QuestionnaireReport struct {
	Questionnaire             QuestionnaireHeader `json:"questionnaire"`
	RespondentCount           int                 `json:"respondent_count"`
	OverallUsabilityScoreAvg  *float64            `json:"overall_usability_score_avg"`
	CategoryAverages          []CategoryAverage   `json:"category_averages"`
	TaskSuccessRateAvg        *float64            `json:"task_success_rate_avg"`
	EvaluationWebsiteSUSScoreAvg *float64         `json:"evaluation_website_sus_score_avg"`
}
