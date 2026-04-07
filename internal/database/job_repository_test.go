package database

import (
	"main/config"
	"main/internal/domain"
	"testing"

	"github.com/google/uuid"
)

func setEnv(t *testing.T) {
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "password")
	t.Setenv("DB_PORT", "5433")
	t.Setenv("DB_NAME", "assets_db")
	t.Setenv("DB_HOST", "localhost")
}

func TestJobRepository_CreateJob(t *testing.T) {
	setEnv(t)

	dbCfg := config.NewDBConfig()
	infra, err := InitInfrastructure(t.Context(), dbCfg.Dsn())
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	repo := NewJobRepository(infra)
	ctx := t.Context()

	job := domain.Job{
		ID:   uuid.New().String(),
		Type: domain.JobTypePing,
	}

	err = repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("CreateJob() expected no error, got %v", err)
	}
}

func TestJobRepository_UpdateStatus(t *testing.T) {
	setEnv(t)

	dbCfg := config.NewDBConfig()
	infra, err := InitInfrastructure(t.Context(), dbCfg.Dsn())
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	repo := NewJobRepository(infra)
	ctx := t.Context()

	job := domain.Job{
		ID:   uuid.New().String(),
		Type: domain.JobTypePing,
	}

	err = repo.CreateJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create test job: %v", err)
	}

	tests := []struct {
		name      string
		jobID     string
		newStatus string
		wantErr   bool
	}{
		{
			name:      "Update to processing",
			jobID:     job.ID,
			newStatus: "processing",
			wantErr:   false,
		},
		{
			name:      "Update to completed",
			jobID:     job.ID,
			newStatus: "completed",
			wantErr:   false,
		},
		{
			name:      "Update non-existent job",
			jobID:     "00000000-0000-0000-0000-000000000000",
			newStatus: "failed",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(ctx, tt.jobID, tt.newStatus)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.jobID == job.ID {
				var status string
				err = infra.DB.QueryRow(ctx, "SELECT status FROM jobs WHERE id = $1", tt.jobID).Scan(&status)
				if err != nil {
					t.Fatalf("Failed to verify status: %v", err)
				}
				if status != tt.newStatus {
					t.Errorf("Expected status %s, got %s", tt.newStatus, status)
				}
			}
		})
	}
}
