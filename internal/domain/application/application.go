package application

import "time"

type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
	StatusRejected Status = "rejected"
)

type Application struct {
	ID          string     `json:"id"`
	FullName    string     `json:"full_name"`
	Email       string     `json:"email"`
	Institution string     `json:"institution,omitempty"`
	Occupation  string     `json:"occupation,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	Status      Status     `json:"status"`
	ReviewNote  string     `json:"review_note,omitempty"`
	ReviewedBy  *string    `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateInput struct {
	FullName    string `json:"full_name" binding:"required"`
	Email       string `json:"email" binding:"required,email"`
	Institution string `json:"institution"`
	Occupation  string `json:"occupation"`
	Reason      string `json:"reason"`
}

type ReviewInput struct {
	ReviewNote string `json:"review_note"`
}

type Page struct {
	Page       int
	PageSize   int
	Total      int
	TotalPages int
}
