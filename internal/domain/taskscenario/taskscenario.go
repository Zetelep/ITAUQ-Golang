package taskscenario

import "time"

type TaskScenario struct {
	ID              string    `json:"id"`
	QuestionnaireID string    `json:"questionnaire_id"`
	Title           string    `json:"title"`
	Instruction     string    `json:"instruction"`
	TaskOrder       int       `json:"task_order"`
	CreatedAt       time.Time `json:"created_at"`
}

type CreateInput struct {
	Title       string `json:"title" binding:"required"`
	Instruction string `json:"instruction" binding:"required"`
	TaskOrder   int    `json:"task_order"`
}

type UpdateInput struct {
	Title       *string `json:"title"`
	Instruction *string `json:"instruction"`
	TaskOrder   *int    `json:"task_order"`
}

// TaskScenarioStats holds aggregated statistics for a single task scenario.
type TaskScenarioStats struct {
	TaskScenarioID    string   `json:"task_scenario_id"`
	Title             string   `json:"title"`
	TaskOrder         int      `json:"task_order"`
	TotalAttempts     int      `json:"total_attempts"`
	SuccessfulAttempts int    `json:"successful_attempts"`
	CompletionRate    float64  `json:"completion_rate"`
	AvgCompletionTime *float64 `json:"avg_completion_time"`
}

// QuestionnaireTaskScenarioStats is the top-level response for the stats endpoint.
type QuestionnaireTaskScenarioStats struct {
	QuestionnaireID string               `json:"questionnaire_id"`
	TaskScenarios   []TaskScenarioStats  `json:"task_scenarios"`
}
