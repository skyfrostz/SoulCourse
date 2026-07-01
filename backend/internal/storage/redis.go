package storage

import (
	"context"

	"subject-choice-forum/backend/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
