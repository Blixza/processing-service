package domain

import "time"

const (
	JobTypeFilterImage = "FILTER_IMAGE"
	JobTypeNotifyUser  = "NOTIFY_USER"
	JobTypeExportData  = "EXPORT_DATA"
	JobTypePing        = "PING"
)

type Job struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	SourceURL   string    `json:"source_url"`
	Filename    string    `json:"filename"`
	CallbackURL string    `json:"callback_url"`
	Payload     string    `json:"payload"`
	CreatedAt   time.Time `json:"created_at"`
}
