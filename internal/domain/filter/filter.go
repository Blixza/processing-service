package filter

type FilterType string

const (
	FilterGrayscale  FilterType = "grayscale"
	FilterInvert     FilterType = "invert"
	FilterBrightness FilterType = "brightness"
	FilterContrast   FilterType = "contrast"

	FilterGaussianBlur FilterType = "gaussian_blur"
	FilterSharpen      FilterType = "sharpen"
	FilterBoxBlur      FilterType = "box_blur"

	FilterResize FilterType = "resize"
	FilterFit    FilterType = "fit"
)

type Filter struct {
	Type   FilterType     `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}
