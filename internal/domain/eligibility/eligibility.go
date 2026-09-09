// Package eligibility defines the "Terms & Conditions checklist" attached to
// a questionnaire. Respondents must tick every active criterion before they
// are allowed to start filling the evaluation. See API_SPECIFICATION.md §6b
// (admin CRUD) and §8 (public-flow gate).
package eligibility

import "time"

// EligibilityCriterion is one statement a respondent must check (e.g.
// "Saya berdomisili di Kota Banjarbaru"). Each criterion is owned by a
// single questionnaire, ordered by CriteriaOrder ascending.
type EligibilityCriterion struct {
	ID              string    `json:"id"`
	QuestionnaireID string    `json:"questionnaire_id"`
	Statement       string    `json:"statement"`
	CriteriaOrder   int       `json:"criteria_order"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateInput is the payload for POST /questionnaires/{id}/eligibility-criteria.
type CreateInput struct {
	Statement     string `json:"statement" binding:"required"`
	CriteriaOrder int    `json:"criteria_order"`
}

// UpdateInput is the payload for PATCH /eligibility-criteria/{id}. Only the
// fields that are set will be applied; nil pointers mean "leave unchanged".
type UpdateInput struct {
	Statement     *string `json:"statement"`
	CriteriaOrder *int    `json:"criteria_order"`
}

// Confirmation is the audit row written when a respondent ticks a criterion
// during POST /public/.../respondents. Maps directly to the
// respondent_eligibility_confirmations table.
type Confirmation struct {
	ID           string    `json:"id"`
	RespondentID string    `json:"respondent_id"`
	CriteriaID   string    `json:"criteria_id"`
	IsChecked    bool      `json:"is_checked"`
	CreatedAt    time.Time `json:"created_at"`
}
