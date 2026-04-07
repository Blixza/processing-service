package database

import (
	"context"
	"main/internal/domain"
	"time"

	sq "github.com/Masterminds/squirrel"
)

type JobRepository interface {
	CreateJob(ctx context.Context, job domain.Job) error
	UpdateStatus(ctx context.Context, id string, status string) error
}

type repo struct {
	infra *Infrastructure
	sb    sq.StatementBuilderType
}

func NewJobRepository(infra *Infrastructure) JobRepository {
	return &repo{
		infra: infra,
		sb:    sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *repo) CreateJob(ctx context.Context, job domain.Job) error {
	query, args, err := r.sb.Insert("jobs").
		Columns("id", "type", "status", "callback_url", "filename").
		Values(job.ID, job.Type, "pending", job.CallbackURL, job.Payload).
		ToSql()

	if err != nil {
		return err
	}

	_, err = r.infra.DB.Exec(ctx, query, args...)

	return err
}

func (r *repo) UpdateStatus(ctx context.Context, id string, status string) error {
	query, args, err := r.sb.Update("jobs").Set("status", status).Set("updated_at", time.Now()).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return err
	}
	
	_, err = r.infra.DB.Exec(ctx, query, args...)

	return err
}
