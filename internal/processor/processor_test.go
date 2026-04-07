package processor

import (
	"main/internal/domain/filter"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestProcessImage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../testdata/sample.jpg")
	}))
	defer ts.Close()

	tests := []struct {
		name     string
		filename string
		filters  []filter.Filter
		wantErr  bool
	}{
		{
			name:     "Grayscale Filter",
			filename: "test_gray.jpg",
			filters:  []filter.Filter{{Type: filter.FilterGrayscale}},
			wantErr:  false,
		},
		{
			name:     "Resize Filter",
			filename: "test_resize.jpg",
			filters:  []filter.Filter{{Type: filter.FilterResize, Params: map[string]any{"width": 100.0, "height": 100.0}}},
			wantErr:  false,
		},
		{
			name:     "Invalid URL",
			filename: "test_fail.jpg",
			filters:  []filter.Filter{},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := ts.URL
			if tt.wantErr && tt.name == "Invalid URL" {
				url = "http://invalid-url-12345.com"
			}

			err := ProcessImage(url, tt.filename, tt.filters...)

			if (err != nil) != tt.wantErr {
				t.Errorf("ProcessImage() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				os.Remove("storage/" + tt.filename)
			}
		})
	}
}
