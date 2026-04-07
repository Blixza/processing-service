package request

import "main/internal/domain/filter"

type ProcessRequest struct {
	URL      string          `json:"source_url" example:"https://example.com/image.jpg"`
	JobType  string          `json:"job_type" example:"FILTER_IMAGE"`
	Callback string          `json:"callback_url" example:"http://localhost:8081/ping"`
	Filename string          `json:"filename" example:"my_image.jpg"`
	Filters  []filter.Filter `json:"filters"`
}
