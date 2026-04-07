package filter

type Type string

const (
	FilterGrayscale  Type = "grayscale"
	FilterInvert     Type = "invert"
	FilterBrightness Type = "brightness"
	FilterContrast   Type = "contrast"

	FilterGaussianBlur Type = "gaussian_blur"
	FilterSharpen      Type = "sharpen"
	FilterBoxBlur      Type = "box_blur"

	FilterResize Type = "resize"
	FilterFit    Type = "fit"
)

type Filter struct {
	Type   Type           `json:"type"`
	Params map[string]any `json:"params,omitempty"`
}
