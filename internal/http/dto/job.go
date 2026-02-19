package dto

import "github.com/cesargomez89/navidrums/internal/domain"

type JobResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Status    string  `json:"status"`
	SourceID  string  `json:"source_id"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Error     string  `json:"error,omitempty"`
	Progress  float64 `json:"progress"`
}

func NewJobResponse(j *domain.Job) JobResponse {
	resp := JobResponse{
		ID:        j.ID,
		Type:      string(j.Type),
		Status:    string(j.Status),
		Progress:  j.Progress,
		SourceID:  j.SourceID,
		CreatedAt: j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: j.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if j.Error != nil {
		resp.Error = *j.Error
	}
	return resp
}
