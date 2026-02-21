package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Redis struct {
	Address  string
	Password string
}

type Config struct {
	Port        string
	DatabaseURL string
	Redis       Redis
}

func LoadConfig() (Config, error) {
	var config Config

	if os.Getenv("ENV") != "production" {
		if err := godotenv.Load(); err != nil {
			return config, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	config.Port = mustGetEnv("PORT")

	config.DatabaseURL = mustGetEnv("DB_URL")

	config.Redis = Redis{
		Address:  os.Getenv("REDIS_ADDRESS"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}

	if _, err := strconv.Atoi(config.Port); err != nil {
		return config, fmt.Errorf("invalid port number: %w", err)
	}

	return config, nil
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}
