package rabbitmq

import (
	"main/config"
	"main/internal/database"
	"main/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func setEnv(t *testing.T) {
	t.Setenv("RABBITMQ_USER", "app")
	t.Setenv("RABBITMQ_PASSWORD", "strongpassword")
	t.Setenv("RABBITMQ_HOST", "0.0.0.0")
	t.Setenv("RABBITMQ_PORT", "5672")
}

func TestProcessJob_Ping(t *testing.T) {
	setEnv(t)

	handler := &RabbitHandler{}
	job := domain.Job{
		ID:   uuid.New().String(),
		Type: domain.JobTypePing,
	}

	dbCfg, err := config.NewDBConfig("test.env")
	if err != nil {
		t.Fatalf("Failed to load DB config: %v", err)
	}

	infra, err := database.InitInfrastructure(t.Context(), dbCfg.Dsn())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	err = handler.ProcessJob(t.Context(), zap.NewNop(), job, infra, true)
	if err != nil {
		t.Errorf("expected success for ping, got %v", err)
	}
}

func TestProcessJob_FilterImage(t *testing.T) {
	setEnv(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../../testdata/sample.jpg")
	}))
	defer ts.Close()

	handler := &RabbitHandler{}

	tests := []struct {
		name string
		job  domain.Job
	}{
		{
			name: "Grayscale Filter",
			job: domain.Job{
				ID:          uuid.New().String(),
				Type:        domain.JobTypeFilterImage,
				SourceURL:   ts.URL,
				CallbackURL: "http://localhost:8081/ping",
				Filename:    "test_gray.jpg",
				Payload:     `[{"type": "grayscale"}]`,
			},
		},
		{
			name: "Fail",
			job: domain.Job{
				ID:          uuid.New().String(),
				Type:        domain.JobTypeFilterImage,
				SourceURL:   ts.URL,
				CallbackURL: "http://localhost:8081/ping",
				Filename:    "killme.jpg",
				Payload:     `[{"type": "grayscale"}]`,
			},
		},
	}

	dbCfg, err := config.NewDBConfig("test.env")
	if err != nil {
		t.Fatalf("Failed to load DB config: %v", err)
	}

	infra, err := database.InitInfrastructure(t.Context(), dbCfg.Dsn())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if tt.job.Filename == "killme.jpg" {
					if r == nil {
						t.Errorf("The code did not panic, but it should have for killme.jpg")
					}
				} else if r != nil {
					t.Errorf("The code panicked unexpectedly: %v", r)
				}
			}()
			tt.job.SourceURL = ts.URL

			err = handler.ProcessJob(t.Context(), zap.NewNop(), tt.job, infra, true)
			if err != nil {
				t.Errorf("expected success for ping, got %v", err)
			}
		})
	}
}
