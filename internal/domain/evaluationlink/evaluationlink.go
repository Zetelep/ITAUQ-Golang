package evaluationlink

import "time"

type EvaluationLink struct {
	ID              string     `json:"id"`
	QuestionnaireID string     `json:"questionnaire_id"`
	Token           string     `json:"token"`
	URL             string     `json:"url"`
	CreatedBy       string     `json:"created_by"`
	IsActive        bool       `json:"is_active"`
	ExpiresAt       *time.Time `json:"expires_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreateInput struct {
	ExpiresAt *time.Time `json:"expires_at"`
}

type UpdateInput struct {
	IsActive  *bool      `json:"is_active"`
	ExpiresAt *time.Time `json:"expires_at"`
}
