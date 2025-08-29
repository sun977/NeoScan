package database

import (
	"context"
	"fmt"
	"time"

	"neomaster/internal/config"

	"github.com/go-redis/redis/v8"
)

// NewRedisConnection 创建Redis连接
func NewRedisConnection(cfg *config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端
	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  time.Duration(cfg.DialTimeout),
		ReadTimeout:  time.Duration(cfg.ReadTimeout),
		WriteTimeout: time.Duration(cfg.WriteTimeout),
		PoolTimeout:  time.Duration(cfg.PoolTimeout),
		IdleTimeout:  time.Duration(cfg.IdleTimeout),
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	return client, nil
}