package cache

import (
	"ai-gateway/common/config"
	"ai-gateway/common/logger"
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var RDB *redis.Client

func InitRedis() {
	cfg := config.GlobalConfig.Redis
	RDB = redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		logger.Log.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	logger.Log.Info("Redis connection established")
}

func SetToken(ctx context.Context, token string, userID uint, expiration time.Duration) error {
	return RDB.Set(ctx, "token:"+token, userID, expiration).Err()
}

func GetToken(ctx context.Context, token string) (string, error) {
	return RDB.Get(ctx, "token:"+token).Result()
}

func ValidateToken(ctx context.Context, token string) (bool, error) {
	_, err := RDB.Get(ctx, "token:"+token).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
