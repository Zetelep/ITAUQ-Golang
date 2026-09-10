package administrator

import "time"

// Administrator is the view shape used by /administrators endpoints. Email
// comes from auth.users (the profile row only carries the user id).
type Administrator struct {
	ID                 string    `json:"id"`
	FullName           string    `json:"full_name"`
	Email              string    `json:"email"`
	Institution        string    `json:"institution,omitempty"`
	Occupation         string    `json:"occupation,omitempty"`
	Role               string    `json:"role"`
	IsActive           bool      `json:"is_active"`
	MustChangePassword bool      `json:"must_change_password"`
	ApplicationID      *string   `json:"application_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
}

type CreateInput struct {
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type UpdateInput struct {
	FullName    *string `json:"full_name,omitempty"`
	Institution *string `json:"institution,omitempty"`
	Occupation  *string `json:"occupation,omitempty"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type ListFilter struct {
	IsActive *bool
	Page     int
	PageSize int
}

type Page struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
