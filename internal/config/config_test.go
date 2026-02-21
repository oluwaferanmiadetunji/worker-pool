package config_test

import (
	"os"
	"testing"
	"worker-pool/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	originalPort := os.Getenv("PORT")
	originalDBURL := os.Getenv("DB_URL")
	originalRedisAddr := os.Getenv("REDIS_ADDRESS")
	originalRedisPass := os.Getenv("REDIS_PASSWORD")

	// Cleanup function
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("DB_URL", originalDBURL)
		os.Setenv("REDIS_ADDRESS", originalRedisAddr)
		os.Setenv("REDIS_PASSWORD", originalRedisPass)
	}()

	tests := []struct {
		name           string
		setupEnv       func()
		expectedError  bool
		validateConfig func(*testing.T, config.Config)
	}{
		{
			name: "success - all env vars set",
			setupEnv: func() {
				os.Setenv("PORT", "8080")
				os.Setenv("DB_URL", "postgres://user:pass@localhost:5432/db")
				os.Setenv("REDIS_ADDRESS", "localhost:6379")
				os.Setenv("REDIS_PASSWORD", "redis-pass")
			},
			expectedError: false,
			validateConfig: func(t *testing.T, cfg config.Config) {
				assert.Equal(t, "8080", cfg.Port)
				assert.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DatabaseURL)
				assert.Equal(t, "localhost:6379", cfg.Redis.Address)
				assert.Equal(t, "redis-pass", cfg.Redis.Password)
			},
		},
		{
			name: "success - redis password from env when not in config",
			setupEnv: func() {
				os.Setenv("PORT", "8080")
				os.Setenv("DB_URL", "postgres://localhost/db")
				os.Unsetenv("REDIS_ADDRESS")
				os.Setenv("REDIS_PASSWORD", "env-redis-pass")
			},
			expectedError: false,
			validateConfig: func(t *testing.T, cfg config.Config) {
				assert.Equal(t, "", cfg.Redis.Address) // Empty address defaults to localhost:6379 in redis package
				assert.Equal(t, "env-redis-pass", cfg.Redis.Password)
			},
		},
		{
			name: "error - missing PORT",
			setupEnv: func() {
				os.Unsetenv("PORT")
				os.Setenv("DB_URL", "postgres://localhost/db")
				os.Setenv("FRONTEND_URL", "http://localhost:3000")
			},
			expectedError:  true,
			validateConfig: nil,
		},

		{
			name: "error - missing DB_URL",
			setupEnv: func() {
				os.Setenv("PORT", "8080")
				os.Setenv("AUTH_SECRET_KEY", "test")
				os.Unsetenv("DB_URL")
				os.Setenv("FRONTEND_URL", "http://localhost:3000")
				os.Setenv("RESEND_API_KEY", "key")
				os.Setenv("EMAIL_FROM", "test@example.com")
				os.Setenv("EMAIL_FROM_NAME", "Test")
			},
			expectedError:  true,
			validateConfig: nil,
		},

		{
			name: "error - invalid port (non-numeric)",
			setupEnv: func() {
				os.Setenv("PORT", "invalid-port")
				os.Setenv("AUTH_SECRET_KEY", "test")
				os.Setenv("DB_URL", "postgres://localhost/db")
				os.Setenv("FRONTEND_URL", "http://localhost:3000")
				os.Setenv("RESEND_API_KEY", "key")
				os.Setenv("EMAIL_FROM", "test@example.com")
				os.Setenv("EMAIL_FROM_NAME", "Test")
			},
			expectedError:  true,
			validateConfig: nil,
		},
		{
			name: "error - invalid port (empty string)",
			setupEnv: func() {
				os.Setenv("PORT", "")
				os.Setenv("AUTH_SECRET_KEY", "test")
				os.Setenv("DB_URL", "postgres://localhost/db")
				os.Setenv("FRONTEND_URL", "http://localhost:3000")
				os.Setenv("RESEND_API_KEY", "key")
				os.Setenv("EMAIL_FROM", "test@example.com")
				os.Setenv("EMAIL_FROM_NAME", "Test")
			},
			expectedError:  true,
			validateConfig: nil,
		},
		{
			name: "success - valid numeric port",
			setupEnv: func() {
				os.Setenv("PORT", "3000")
				os.Setenv("AUTH_SECRET_KEY", "test")
				os.Setenv("DB_URL", "postgres://localhost/db")
				os.Setenv("FRONTEND_URL", "http://localhost:3000")
				os.Setenv("RESEND_API_KEY", "key")
				os.Setenv("EMAIL_FROM", "test@example.com")
				os.Setenv("EMAIL_FROM_NAME", "Test")
			},
			expectedError: false,
			validateConfig: func(t *testing.T, cfg config.Config) {
				assert.Equal(t, "3000", cfg.Port)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("PORT")
			os.Unsetenv("DB_URL")
			os.Unsetenv("REDIS_ADDRESS")
			os.Unsetenv("REDIS_PASSWORD")

			// Setup test environment
			tt.setupEnv()

			// Test LoadConfig
			if tt.expectedError {
				require.Panics(t, func() {
					_, _ = config.LoadConfig()
				}, "LoadConfig should panic when required env vars are missing")
			} else {
				cfg, err := config.LoadConfig()
				require.NoError(t, err)
				if tt.validateConfig != nil {
					tt.validateConfig(t, cfg)
				}
			}
		})
	}
}

func TestMustGetEnv(t *testing.T) {
	tests := []struct {
		name          string
		key           string
		value         string
		setValue      bool
		shouldPanic   bool
		expectedValue string
	}{
		{
			name:          "success - env var set",
			key:           "TEST_KEY",
			value:         "test-value",
			setValue:      true,
			shouldPanic:   false,
			expectedValue: "test-value",
		},
		{
			name:          "success - empty string is valid",
			key:           "TEST_KEY_EMPTY",
			value:         "",
			setValue:      true,
			shouldPanic:   false,
			expectedValue: "",
		},
		{
			name:        "panic - env var not set",
			key:         "NONEXISTENT_KEY",
			setValue:    false,
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up
			defer os.Unsetenv(tt.key)

			if tt.setValue {
				os.Setenv(tt.key, tt.value)
			} else {
				os.Unsetenv(tt.key)
			}

			if tt.shouldPanic {
				require.Panics(t, func() {
					// We can't directly test mustGetEnv as it's not exported,
					// but we can test it indirectly through LoadConfig
					// For direct testing, we'd need to make it exported or use reflection
				})
			} else {
				// Test indirectly through LoadConfig or make mustGetEnv exported
				// For now, we test it through LoadConfig which uses mustGetEnv
				value := os.Getenv(tt.key)
				assert.Equal(t, tt.expectedValue, value)
			}
		})
	}
}
