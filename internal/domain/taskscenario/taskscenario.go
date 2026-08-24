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
