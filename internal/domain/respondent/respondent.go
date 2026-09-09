package respondent

import "time"

// Respondent is an anonymous user filling out a public evaluation link.
type Respondent struct {
	ID               string     `json:"id"`
	EvaluationLinkID string     `json:"evaluation_link_id"`
	Name             string     `json:"name"`
	Email            string     `json:"email,omitempty"`
	Age              *int       `json:"age,omitempty"`
	Gender           *string    `json:"gender,omitempty"`
	Occupation       string     `json:"occupation,omitempty"`
	StartedAt        time.Time  `json:"started_at"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// StartInput is the payload for POST /public/evaluation/:token/respondents.
// CheckedCriteriaIDs is the respondent's tick-confirmation for the
// questionnaire's eligibility criteria; if any active criterion is missing
// here, the backend rejects the request (see API_SPECIFICATION.md §8).
type StartInput struct {
	Name              string   `json:"name" binding:"required"`
	Email             string   `json:"email"`
	Age               *int     `json:"age"`
	Gender            *string  `json:"gender"`
	Occupation        string   `json:"occupation"`
	CheckedCriteriaIDs []string `json:"checked_criteria_ids"`
}

// Attempt is a single task-scenario attempt submitted by a respondent.
type Attempt struct {
	ID              string    `json:"id"`
	RespondentID    string    `json:"respondent_id"`
	TaskScenarioID  string    `json:"task_scenario_id"`
	IsSuccess       *bool     `json:"is_success,omitempty"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	Notes           string    `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
}

// AttemptInput is the payload for a single attempt inside the task-attempts
// batch endpoint.
type AttemptInput struct {
	TaskScenarioID  string `json:"task_scenario_id" binding:"required"`
	IsSuccess       *bool  `json:"is_success"`
	DurationSeconds *int   `json:"duration_seconds"`
	Notes           string `json:"notes"`
}

// AttemptsInput is the wrapper around a batch of attempt inputs.
type AttemptsInput struct {
	Attempts []AttemptInput `json:"attempts" binding:"required"`
}

// Answer is a single Likert answer in the questionnaire_answers table.
type Answer struct {
	ID           string    `json:"id"`
	RespondentID string    `json:"respondent_id"`
	ItemID       int       `json:"item_id"`
	Category     string    `json:"category"`
	Score        int       `json:"score"`
	CreatedAt    time.Time `json:"created_at"`
}

// AnswerInput is the payload for a single answer inside the answers endpoint.
type AnswerInput struct {
	ItemID   int    `json:"item_id" binding:"required"`
	Category string `json:"category" binding:"required"`
	Score    int    `json:"score" binding:"required"`
}

// AnswersInput is the wrapper around a batch of answer inputs.
type AnswersInput struct {
	Answers []AnswerInput `json:"answers" binding:"required"`
}

// SUSAnswer is a single Likert answer in the sus_answers table — respondent's
// rating of the evaluation website itself (not the app under evaluation).
type SUSAnswer struct {
	ID           string    `json:"id"`
	RespondentID string    `json:"respondent_id"`
	ItemID       int       `json:"item_id"`
	Score        int       `json:"score"`
	CreatedAt    time.Time `json:"created_at"`
}

// SUSAnswerInput is the payload for a single SUS answer.
type SUSAnswerInput struct {
	ItemID int `json:"item_id" binding:"required"`
	Score  int `json:"score"  binding:"required"`
}

// SUSAnswersInput is the wrapper around a batch of SUS answer inputs.
type SUSAnswersInput struct {
	Answers []SUSAnswerInput `json:"answers" binding:"required"`
}