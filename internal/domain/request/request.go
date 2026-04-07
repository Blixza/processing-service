package request

import "main/internal/domain/filter"

type ProcessRequest struct {
	URL      string          `json:"source_url" example:"https://example.com/image.jpg"` //nolint:tagalign
	JobType  string          `json:"job_type" example:"FILTER_IMAGE"`                    //nolint:tagalign
	Callback string          `json:"callback_url" example:"http://localhost:8081/ping"`  //nolint:tagalign
	Filename string          `json:"filename" example:"my_image.jpg"`                    //nolint:tagalign
	Filters  []filter.Filter `json:"filters"`                                            //nolint:tagalign
}
