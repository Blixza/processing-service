package processor

import (
	"fmt"
	"image"
	"image/jpeg"
	"main/internal/domain/filter"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func ProcessImage(sourceURL string, filename string, filters ...filter.Filter) error {
	resp, err := http.Get(sourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	for _, f := range filters {
		switch f.Type {
		case filter.FilterGrayscale:
			img = imaging.Grayscale(img)
		case filter.FilterInvert:
			img = imaging.Invert(img)
		case filter.FilterBrightness:
			percentage, _ := f.Params["percentage"].(float64)

			img = imaging.AdjustBrightness(img, percentage)
		case filter.FilterContrast:
			percentage, _ := f.Params["percentage"].(float64)

			img = imaging.AdjustContrast(img, percentage)
		case filter.FilterGaussianBlur:
			x := img.Bounds().Max.X
			y := img.Bounds().Max.Y

			img = imaging.Resize(img, x, y, imaging.Gaussian)
		case filter.FilterSharpen:
			sigma, _ := f.Params["sigma"].(float64)

			img = imaging.Sharpen(img, sigma)
		case filter.FilterBoxBlur:
			x := img.Bounds().Max.X
			y := img.Bounds().Max.Y

			img = imaging.Resize(img, x, y, imaging.Box)
		case filter.FilterResize:
			w, _ := f.Params["width"].(float64)
			h, _ := f.Params["height"].(float64)

			if w == 0 || h == 0 {
				return fmt.Errorf("resize dimensions are incorrect. Must be 'width' and 'height'")
			}
			img = imaging.Resize(img, int(w), int(h), imaging.Linear) // TODO
		case filter.FilterFit:
			w, _ := f.Params["width"].(int)
			h, _ := f.Params["height"].(int)

			if w == 0 || h == 0 {
				return fmt.Errorf("resize dimensions are incorrect. Must be 'width' and 'height'")
			}

			img = imaging.Fit(img, w, h, imaging.Linear)
		}

		outDir := "storage"
		outPath := filepath.Join(outDir, filename)

		err = os.MkdirAll(outDir, 0755)
		if err != nil {
			return err
		}

		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()

		err = jpeg.Encode(f, img, nil)
		if err != nil {
			return err
		}
	}

	return nil
}
