package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type WebhookPayload struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	Filename   string `json:"filename"`
	FinishedAt string `json:"finished_at"`
}

func SendWebhook(ctx context.Context, log *zap.Logger, url string, timeoutSec int, data WebhookPayload) {
	if url == "" {
		return
	}

	body, _ := json.Marshal(data)

	client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(body))
	if err != nil {
		log.Error("Failed to create a request", zap.String("url", url), zap.Error(err))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("Failed to send webhook", zap.String("url", url), zap.Error(err))
	}
	defer resp.Body.Close()

	log.Info("Webhook sent successfully", zap.String("url", url), zap.Error(err))
}
