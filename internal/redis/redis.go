package redis

import (
	"context"
	"os"

	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Address  string
	Password string
}

var RedisClient *redis.Client

func InitRedis(cfg RedisConfig) error {
	redisAddress := cfg.Address
	if redisAddress == "" {
		redisAddress = "localhost:6379"
	}

	password := cfg.Password
	if password == "" {
		password = os.Getenv("REDIS_PASSWORD")
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddress,
		Password: password,
		DB:       0,
	})

	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		RedisClient = nil
		return err
	}

	return nil
}

func GetRedisClient() *redis.Client {
	return RedisClient
}
