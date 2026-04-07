package processor

import (
	"context"
	"errors"
	"image"
	"image/jpeg"
	"main/internal/domain/filter"
	"net/http"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
)

func saveImage(img *image.Image, filename string) error {
	outDir := "storage"
	outPath := filepath.Join(outDir, filename)

	err := os.MkdirAll(outDir, 0750)
	if err != nil {
		return err
	}

	var f *os.File
	f, err = os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	err = jpeg.Encode(f, *img, nil)
	if err != nil {
		return err
	}

	return nil
}

func processFilters(imgPtr *image.Image, filters ...filter.Filter) error {
	img := *imgPtr
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
				return errors.New("resize dimensions are incorrect. Must be 'width' and 'height'")
			}

			img = imaging.Resize(img, int(w), int(h), imaging.Linear) // TODO
		case filter.FilterFit:
			w, _ := f.Params["width"].(int)
			h, _ := f.Params["height"].(int)

			if w == 0 || h == 0 {
				return errors.New("resize dimensions are incorrect. Must be 'width' and 'height'")
			}

			img = imaging.Fit(img, w, h, imaging.Linear)
		}

		*imgPtr = img
	}

	return nil
}

func ProcessImage(ctx context.Context, sourceURL string, filename string, filters ...filter.Filter) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return err
	}

	err = processFilters(&img, filters...)
	if err != nil {
		return err
	}

	err = saveImage(&img, filename)
	if err != nil {
		return err
	}

	return nil
}
