package callback

import (
	"bytes"
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

func SendWebhook(log *zap.Logger, url string, data WebhookPayload) {
	if url == "" {
		return
	}

	body, _ := json.Marshal(data)

	client := &http.Client{Timeout: 5 * time.Second} //nolint:mnd

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Error("Failed to send webhook", zap.String("url", url), zap.Error(err))
	}
	defer resp.Body.Close()

	log.Info("Webhook sent successfully", zap.String("url", url), zap.Error(err))
}
