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

	// FilterEdgeDetect FilterType = "edge_detect" not found
	// FilterEmboss     FilterType = "emboss" not found
	// FilterPixelate   FilterType = "pixelate" not found or idk how

	FilterResize FilterType = "resize"
	FilterFit    FilterType = "fit"
	// FilterThumbnail FilterType = "thumbnail" what is that
)

type Filter struct {
	Type   FilterType     `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}
