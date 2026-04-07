package server

import (
	"context"
	"main/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

type MockQueue struct {
	Jobs []domain.Job
}

func (m *MockQueue) PublishJob(ctx context.Context, job domain.Job) error {
	m.Jobs = append(m.Jobs, job)
	return nil
}

type MockRepo struct {
	CreatedJobs []domain.Job
}

func (m *MockRepo) CreateJob(ctx context.Context, job domain.Job) error {
	m.CreatedJobs = append(m.CreatedJobs, job)
	return nil
}

func (m *MockRepo) UpdateStatus(ctx context.Context, s1 string, s2 string) error {
	return nil
}

func TestHandlePing(t *testing.T) {
	mockQueue := &MockQueue{}
	srv := &Server{
		Queue: mockQueue,
		Log:   zap.NewNop(),
	}

	body := `{"source_url": "https://i.pinimg.com/1200x/21/9c/e3/219ce3c0bcbd51efe9879d765ca8d715.jpg", "job_type": "PING"}`
	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.HandlePing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHandleProcess(t *testing.T) {
	mockQueue := &MockQueue{}
	mockRepo := MockRepo{}

	srv := &Server{
		Queue: mockQueue,
		Repo:  &mockRepo,
		Log:   zap.NewNop(),
	}

	body := `{"source_url": "https://i.pinimg.com/1200x/21/9c/e3/219ce3c0bcbd51efe9879d765ca8d715.jpg", "callback_url": "http://localhost:8081/ping", "job_type": "FILTER_IMAGE"}`
	req := httptest.NewRequest(http.MethodPost, "/process", strings.NewReader(body))
	w := httptest.NewRecorder()

	srv.HandleProcess(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	if len(mockQueue.Jobs) != 1 {
		t.Errorf("Expected 1 job in queue, got %d", len(mockQueue.Jobs))
	}
}
