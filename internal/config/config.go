package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
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
