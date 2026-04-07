package server

import (
	"encoding/json"
	"main/internal/database"
	"main/internal/domain"
	"main/internal/domain/request"
	"main/internal/transport/rabbitmq"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Server struct {
	Queue rabbitmq.QueueHandler
	Repo  database.JobRepository
	Log   *zap.Logger
}

// HandleProcess
// @Summary Create a new image processing job
// @Description Accepts a URL and a callback, then queues an image for grayscale processing
// @Tags jobs
// @Accept json
// @Produce json
// @Param request body process.ProcessRequest true "Job details"
// @Success 201 {object} domain.Job
// @Failure 400 {string} string "Invalid input"
// @Router /process [post].
func (s *Server) HandleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST to send a payload", http.StatusMethodNotAllowed)

		return
	}

	var data request.ProcessRequest

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)

		return
	}

	if (data.URL == "" || data.Callback == "") && data.JobType != "PING" {
		http.Error(w, "URL and Callback are required in payload", http.StatusBadRequest)

		return
	}

	payload, err := json.Marshal(data.Filters)
	if err != nil {
		http.Error(w, "Filters are incorrect", http.StatusBadRequest)
	}

	newJob := domain.Job{
		ID:          uuid.New().String(),
		Type:        data.JobType,
		SourceURL:   data.URL,
		CallbackURL: data.Callback,
		Filename:    data.Filename,
		Payload:     string(payload),
		CreatedAt:   time.Now(),
	}

	err = s.Repo.CreateJob(r.Context(), newJob)
	if err != nil {
		http.Error(w, "Failed to save job", http.StatusInternalServerError)

		return
	}

	err = s.Queue.PublishJob(r.Context(), newJob)
	if err != nil {
		http.Error(w, "Failed to queue job", http.StatusInternalServerError)

		return
	}

	w.Write([]byte("Job Queued: " + newJob.ID))
}

// HandlePing
// @Summary Healthcheck
// @Description Returns "pong"
// @Router /ping [get].
func (s *Server) HandlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Write([]byte("pong"))
	}
}
