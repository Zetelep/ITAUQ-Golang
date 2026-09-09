package profile

import "time"

type Profile struct {
	ID                 string    `json:"id"`
	FullName           string    `json:"full_name"`
	Occupation         string    `json:"occupation,omitempty"`
	Institution        string    `json:"institution,omitempty"`
	Role               string    `json:"role"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
}

type UpdateInput struct {
	FullName *string `json:"full_name,omitempty"`
}
