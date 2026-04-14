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

// const ConsumerTag = "worker-01"

type QueueHandler interface {
	PublishJob(ctx context.Context, job domain.Job) error
}

type RabbitHandler struct {
	Conn        *amqp.Connection
	QueueName   string
	ConsumerTag string
	Dsn         string
}

func NewRabbitHandler(dsn string) (*RabbitHandler, error) {
	conn, err := amqp.Dial(dsn)
	if err != nil {
		return nil, err
	}

	handler := &RabbitHandler{
		Conn:      conn,
		QueueName: "image_jobs",
		Dsn:       dsn,
	}

	tempCh, err := conn.Channel()
	if err != nil {
		return nil, err
	}
	defer tempCh.Close()

	err = handler.SetupQueues(tempCh)
	if err != nil {
		return nil, err
	}

	return handler, nil
}

func (r *RabbitHandler) PublishJob(ctx context.Context, job domain.Job) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open channel for publishing: %w", err)
	}
	defer ch.Close()

	body, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return ch.PublishWithContext(
		ctx,
		"",          // exchange
		r.QueueName, // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}

func (r *RabbitHandler) ConsumeJobsWithRetry(
	ctx context.Context, log *zap.Logger,
	infra *database.Infrastructure, wg *sync.WaitGroup,
	registry *metrics.Registry, restartIntervalSec int,
	isTest bool,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if r.Conn == nil || r.Conn.IsClosed() {
			log.Info("Connection lost. Dialing back..")
			newConn, err := amqp.Dial(r.Dsn)
			if err != nil {
				log.Error("Dialing failed", zap.Error(err))
				time.Sleep(time.Duration(restartIntervalSec) * time.Second)
				continue
			}
			r.Conn = newConn

			tempCh, err := r.Conn.Channel()
			if err != nil {
				log.Error("Failed to setup queues for a new connection", zap.Error(err))
			}

			r.SetupQueues(tempCh)
			tempCh.Close()

			log.Info("Successfully connected back")
		}

		err := r.ConsumeJobs(ctx, log, infra, wg, registry, isTest)
		if err != nil {
			log.Error("Failed to start worker. Restarting in 5s...", zap.Error(err))

			select {
			case <-time.After(time.Duration(restartIntervalSec) * time.Second):
				continue
			case <-ctx.Done():
				return
			}
		}

	}
}

func (r *RabbitHandler) ConsumeJobs(
	ctx context.Context, log *zap.Logger,
	infra *database.Infrastructure, wg *sync.WaitGroup,
	registry *metrics.Registry, isTest bool,
) error {
	ch, err := r.Conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx_exchange",
		"x-dead-letter-routing-key": "failed_key",
	}

	_, err = ch.QueueDeclare(
		r.QueueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		args,        // arguments
	)
	if err != nil {
		return err
	}

	currentTag := fmt.Sprintf("worker--%d", time.Now().Unix())

	msgs, err := ch.Consume(
		r.QueueName,
		currentTag, // consumer tag
		false,      // auto-ack (for now)
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return err
	}

	r.ConsumerTag = currentTag

	log.Info("Worker started", zap.String("tag", currentTag))

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled, stopping consumer loop")
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("rabbit channel closed unexpectedly")
			}
			wg.Add(1)
			go r.handleMessage(ctx, log, d, infra, wg, registry, isTest)
		}
	}
}

func (r *RabbitHandler) handleMessage(
	ctx context.Context, log *zap.Logger,
	d amqp.Delivery, infra *database.Infrastructure,
	wg *sync.WaitGroup, registry *metrics.Registry,
	isTest bool,
) {
	defer wg.Done()

	defer func() {
		r := recover()
		if r != nil {
			log.Error("Worker panicked during processing", zap.Any("panic", r))
			_ = d.Nack(false, false)
		}
	}()

	now := time.Now()
	var job domain.Job

	err := json.Unmarshal(d.Body, &job)
	if err != nil {
		log.Error("Error encoding job", zap.String("job id", job.ID), zap.Error(err))
		return
	}

	finalStatus := "completed"

	if isTest {
		if job.Filename == "killme.jpg" {
			log.Warn("Kill me detected. Panicking")
			_ = d.Nack(false, false)
			r.Conn.Close()
			panic("runtime error: invalid memory address or nil pointer deference")
		}
	}

	err = r.ProcessJob(ctx, log, job, infra, false)
	if err != nil {
		log.Error("Failed to process job", zap.String("job id", job.ID), zap.Error(err))
		finalStatus = "failed"
	}

	registry.JobsProcessed.WithLabelValues(finalStatus, job.Type).Inc()
	registry.ProcessingDuration.Observe(time.Since(now).Seconds())

	log.Info("Job completed", zap.String("job id", job.ID))

	callbackTimeoutSec := 5
	callback.SendWebhook(ctx, log, job.CallbackURL, callbackTimeoutSec, callback.WebhookPayload{
		JobID:      job.ID,
		Status:     finalStatus,
		Filename:   job.Filename,
		FinishedAt: time.Now().Format(time.RFC3339),
	})

	err = d.Ack(false)
	if err != nil {
		log.Error("Failed to acknowledge rabbitmq delivery", zap.Error(err))
	}
}

func (r *RabbitHandler) ProcessJob(
	ctx context.Context,
	log *zap.Logger,
	job domain.Job,
	infra *database.Infrastructure,
	isTest bool,
) error {
	repo := database.NewJobRepository(infra)
	statusKey := fmt.Sprintf("job:%s:status", job.ID)

	log.Info("Worker processing Job", zap.String("job id", job.ID))

	err := repo.UpdateStatus(ctx, job.ID, "processing")
	if err != nil {
		log.Error("Failed to update job status", zap.String("job id", job.ID), zap.String("new status", "processing"))
	}

	cmd := infra.Redis.Set(ctx, statusKey, "processing", 0)
	if cmd.Err() != nil {
		log.Error(
			"Redis: failed to update job status",
			zap.String("job id", job.ID),
			zap.String("new status", "processing"),
			zap.Error(err),
		)
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

		err = json.Unmarshal([]byte(job.Payload), &filters)
		if err != nil {
			log.Error("Unmarshaling filters failed", zap.String("job id", job.ID), zap.Error(err))

			return err
		}

		processError = processor.ProcessImage(context.Background(), job.SourceURL, filename, filters...)
		if processError != nil {
			log.Error("Processing failed", zap.String("job id", job.ID), zap.Error(processError))
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
			log.Error(
				"Failed to update job status",
				zap.String("job id", job.ID),
				zap.String("new status", "failed"),
				zap.Error(err),
			)
		}

		finalStatus = "failed"
	} else {
		err = repo.UpdateStatus(ctx, job.ID, "completed")
		if err != nil {
			log.Error(
				"Failed to update job status",
				zap.String("job id", job.ID),
				zap.String("new status", "completed"),
			)
		}
	}

	cmd = infra.Redis.Set(ctx, statusKey, finalStatus, 0)
	if cmd.Err() != nil {
		log.Error(
			"Redis: failed to update job status",
			zap.String("job id", job.ID),
			zap.String("new status", finalStatus),
			zap.Error(err),
		)
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

func (r *RabbitHandler) SetupQueues(ch *amqp.Channel) error {
	err := ch.ExchangeDeclare(
		"dlx_exchange", // name
		"direct",       // type
		true,           // durable
		false,          // auto-deleted
		false,          // internal
		false,          // no-wait
		nil,            // arguments
	)
	if err != nil {
		return err
	}

	_, err = ch.QueueDeclare(
		"image_jobs_failed", // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)

	err = ch.QueueBind("image_jobs_failed", "failed_key", "dlx_exchange", false, nil)

	args := amqp.Table{
		"x-dead-letter-exchange":    "dlx_exchange",
		"x-dead-letter-routing-key": "failed_key",
	}

	_, err = ch.QueueDeclare(
		"image_jobs", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		args,         // PASS THE DLX ARGS HERE
	)
	return err
}
