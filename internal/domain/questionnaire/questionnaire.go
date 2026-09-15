package questionnaire

import "time"

type Status string

const (
	StatusDraft  Status = "draft"
	StatusActive Status = "active"
	StatusClosed Status = "closed"
)

type Questionnaire struct {
	ID              string    `json:"id"`
	AdministratorID string    `json:"administrator_id"`
	Title           string    `json:"title"`
	AppName         string    `json:"app_name"`
	Description     string    `json:"description,omitempty"`
	ItauqVersion    string    `json:"itauq_version"`
	Status          Status    `json:"status"`
	AppLink         string    `json:"app_link,omitempty"`
	ImgLink         string    `json:"img_link,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateInput struct {
	Title        string `json:"title" binding:"required"`
	AppName      string `json:"app_name" binding:"required"`
	Description  string `json:"description"`
	Status       Status `json:"status"`
	ItauqVersion string `json:"itauq_version"`
	AppLink      string `json:"app_link"`
	ImgLink      string `json:"img_link"`
}

type UpdateInput struct {
	Title       *string `json:"title"`
	AppName     *string `json:"app_name"`
	Description *string `json:"description"`
	Status      *Status `json:"status"`
	AppLink     *string `json:"app_link"`
	ImgLink     *string `json:"img_link"`
}

type Page struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}
