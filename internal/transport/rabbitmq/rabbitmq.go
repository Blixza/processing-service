package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"main/internal/callback"
	"main/internal/database"
	"main/internal/domain"
	"main/internal/domain/filter"
	"main/internal/metrics"
	"main/internal/processor"
	"strings"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const ConsumerTag = "worker-01"

type QueueHandler interface {
	PublishJob(ctx context.Context, job domain.Job) error
}

type RabbitHandler struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Queue   amqp.Queue
}

func NewRabbitHandler(dsn string) (*RabbitHandler, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		"job_queue", // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return nil, err
	}

	return &RabbitHandler{
		Conn:    conn,
		Channel: ch,
		Queue:   q,
	}, nil
}

func (r *RabbitHandler) PublishJob(ctx context.Context, job domain.Job) error {
	body, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return r.Channel.PublishWithContext(
		ctx,
		"",           // exchange
		r.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (r *RabbitHandler) ConsumeJobs(ctx context.Context, log *zap.Logger, infra *database.Infrastructure, wg *sync.WaitGroup) error {
	msgs, err := r.Channel.Consume(
		r.Queue.Name,
		ConsumerTag, // consumer tag
		false,       // auto-ack (for now)
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info("Context cancelled, stopping consumer loop")

				return
			case d, ok := <-msgs:
				if !ok {
					return
				}

				wg.Add(1)

				go func(d amqp.Delivery) {
					defer wg.Done()

					now := time.Now()

					var job domain.Job

					err := json.Unmarshal(d.Body, &job)
					if err != nil {
						log.Error("Error encoding job", zap.String("job id", job.ID), zap.Error(err))
						d.Ack(false)
					}

					finalStatus := "completed"

					err = r.ProcessJob(context.Background(), log, job, infra)
					if err != nil {
						log.Error("Failed to process job", zap.String("job id", job.ID), zap.Error(err))

						finalStatus = "failed"
					}

					metrics.JobsProcessed.WithLabelValues(finalStatus, job.Type).Inc()
					metrics.ProcessingDuration.Observe(time.Since(now).Seconds())

					log.Info("Job completed", zap.String("job id", job.ID))

					callback.SendWebhook(log, job.CallbackURL, callback.WebhookPayload{
						JobID:      job.ID,
						Status:     "completed",
						Filename:   job.Filename,
						FinishedAt: time.Now().Format(time.RFC3339),
					})

					d.Ack(false)
				}(d)
			}
		}
	}()

	return nil
}

func (r *RabbitHandler) ProcessJob(ctx context.Context, log *zap.Logger, job domain.Job, infra *database.Infrastructure) error {
	repo := database.NewJobRepository(infra)
	statusKey := fmt.Sprintf("job:%s:status", job.ID)

	log.Info("Worker processing Job", zap.String("job id", job.ID))

	err := repo.UpdateStatus(ctx, job.ID, "processing")
	if err != nil {
		log.Error("Failed to update job status", zap.String("job id", job.ID), zap.String("new status", "processing"))
	}

	cmd := infra.Redis.Set(ctx, statusKey, "processing", 0)
	if cmd.Err() != nil {
		log.Error("Redis: failed to update job status", zap.String("job id", job.ID), zap.String("new status", "processing"), zap.Error(err))
	}

	finalStatus := "completed"

	var processError error

	switch job.Type {
	case domain.JobTypePing:
		log.Info("Got ping job", zap.String("id", job.ID))
	case domain.JobTypeFilterImage:
		log.Info("Applying filter to image", zap.String("id", job.ID))
		filename := generateFilename(job)

		var filters []filter.Filter

		err := json.Unmarshal([]byte(job.Payload), &filters)
		if err != nil {
			log.Error("Unmarshaling filters failed", zap.String("job id", job.ID), zap.Error(err))

			return err
		}

		processError = processor.ProcessImage(job.SourceURL, filename, filters...)
		if processError != nil {
			log.Error("Processing failed", zap.String("job id", job.ID), zap.Error(err))
		} else {
			log.Info("Image saved", zap.String("path", fmt.Sprintf("storage/%s", job.Filename)))
		}
	case domain.JobTypeNotifyUser:
		log.Info("Sending notification", zap.String("id", job.ID))
		// TODO
	default:
		log.Warn("unknown job type received", zap.String("type", job.Type))
	}

	if processError != nil {
		err = repo.UpdateStatus(ctx, job.ID, "failed")
		if err != nil {
			log.Error("Failed to update job status", zap.String("job id", job.ID), zap.String("new status", "failed"), zap.Error(err))
		}

		finalStatus = "failed"
	} else {
		err = repo.UpdateStatus(ctx, job.ID, "completed")
		if err != nil {
			log.Error("Failed to update job status", zap.String("job id", job.ID), zap.String("new status", "completed"))
		}
	}

	cmd = infra.Redis.Set(ctx, statusKey, finalStatus, 0)
	if cmd.Err() != nil {
		log.Error("Redis: failed to update job status", zap.String("job id", job.ID), zap.String("new status", finalStatus), zap.Error(err))
	}

	return nil
}

func generateFilename(job domain.Job) string {
	filename := job.ID

	if job.Filename != "" {
		b := strings.Builder{}
		b.WriteString(filename)
		b.WriteString("_")
		b.WriteString(job.Filename)
		filename = b.String()
	}

	return filename
}
