package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Infrastructure struct {
	DB    *pgxpool.Pool
	Redis *redis.Client
}

func InitInfrastructure(ctx context.Context, dsn string) (*Infrastructure, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})

	err = rdb.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	// TODO log

	return &Infrastructure{
		DB: db,
		Redis: rdb,
	}, nil
}
