package cache

import (
	"context"

	"github.com/raufendro/novacore/internal/config"
	"github.com/redis/go-redis/v9"
)

func Connect(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	client := redis.NewClient(&redis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB})
	return client, client.Ping(ctx).Err()
}
