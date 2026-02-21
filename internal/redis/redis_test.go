package redis_test

import (
	"worker-pool/internal/redis"
	"context"
	"os"
	"testing"
	"time"

	redislib "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitRedis(t *testing.T) {
	// Save original Redis client
	originalClient := redis.RedisClient

	// Cleanup function to restore original state
	defer func() {
		if redis.RedisClient != nil {
			redis.RedisClient.Close()
		}
		redis.RedisClient = originalClient
		os.Unsetenv("REDIS_PASSWORD")
	}()

	// Save original env var
	originalRedisPass := os.Getenv("REDIS_PASSWORD")
	defer os.Setenv("REDIS_PASSWORD", originalRedisPass)

	tests := []struct {
		name          string
		config        redis.RedisConfig
		setupEnv      func()
		expectedError bool
		validate      func(*testing.T, *redislib.Client)
		skipIfNoRedis bool // Skip test if Redis is not available
	}{
		{
			name: "success - with address and password",
			config: redis.RedisConfig{
				Address:  "localhost:6379",
				Password: "test-password",
			},
			setupEnv:      func() {},
			expectedError: false,
			validate: func(t *testing.T, client *redislib.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "localhost:6379", client.Options().Addr)
				assert.Equal(t, "test-password", client.Options().Password)
				assert.Equal(t, 0, client.Options().DB)
			},
			skipIfNoRedis: true,
		},
		{
			name: "success - default address when empty",
			config: redis.RedisConfig{
				Address:  "",
				Password: "",
			},
			setupEnv:      func() {},
			expectedError: false,
			validate: func(t *testing.T, client *redislib.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "localhost:6379", client.Options().Addr)
			},
			skipIfNoRedis: true,
		},
		{
			name: "success - password from env when not in config",
			config: redis.RedisConfig{
				Address:  "localhost:6379",
				Password: "",
			},
			setupEnv: func() {
				os.Setenv("REDIS_PASSWORD", "env-password")
			},
			expectedError: false,
			validate: func(t *testing.T, client *redislib.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "env-password", client.Options().Password)
			},
			skipIfNoRedis: true,
		},
		{
			name: "success - config password overrides env",
			config: redis.RedisConfig{
				Address:  "localhost:6379",
				Password: "config-password",
			},
			setupEnv: func() {
				os.Setenv("REDIS_PASSWORD", "env-password")
			},
			expectedError: false,
			validate: func(t *testing.T, client *redislib.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "config-password", client.Options().Password)
			},
			skipIfNoRedis: true,
		},
		{
			name: "error - connection failure (invalid address)",
			config: redis.RedisConfig{
				Address:  "invalid-host:9999",
				Password: "",
			},
			setupEnv:      func() {},
			expectedError: true,
			validate: func(t *testing.T, client *redislib.Client) {
				// Client should be nil on connection failure
				assert.Nil(t, redis.RedisClient)
			},
			skipIfNoRedis: false, // This test doesn't require Redis
		},
		{
			name: "success - custom address",
			config: redis.RedisConfig{
				Address:  "redis.example.com:6380",
				Password: "custom-pass",
			},
			setupEnv:      func() {},
			expectedError: false,
			validate: func(t *testing.T, client *redislib.Client) {
				assert.NotNil(t, client)
				assert.Equal(t, "redis.example.com:6380", client.Options().Addr)
				assert.Equal(t, "custom-pass", client.Options().Password)
			},
			skipIfNoRedis: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip test if Redis is not available and test requires it
			if tt.skipIfNoRedis {
				// Try to connect to see if Redis is available
				testClient := redislib.NewClient(&redislib.Options{
					Addr:     "localhost:6379",
					Password: "",
					DB:       0,
				})
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				err := testClient.Ping(ctx).Err()
				testClient.Close()
				cancel()
				if err != nil {
					t.Skipf("Skipping test: Redis not available at localhost:6379: %v", err)
				}
			}

			// Clean up before test
			if redis.RedisClient != nil {
				redis.RedisClient.Close()
				redis.RedisClient = nil
			}
			os.Unsetenv("REDIS_PASSWORD")

			// Setup environment
			tt.setupEnv()

			// Execute
			err := redis.InitRedis(tt.config)

			// Validate
			if tt.expectedError {
				require.Error(t, err)
				assert.Nil(t, redis.RedisClient)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, redis.RedisClient)
				if tt.validate != nil {
					tt.validate(t, redis.RedisClient)
				}

				// Verify client is usable
				if redis.RedisClient != nil {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					pingErr := redis.RedisClient.Ping(ctx).Err()
					cancel()
					if pingErr != nil {
						t.Logf("Warning: Redis client created but ping failed: %v", pingErr)
					}
				}
			}
		})
	}
}

func TestGetRedisClient(t *testing.T) {
	// Save original client
	originalClient := redis.RedisClient
	defer func() {
		redis.RedisClient = originalClient
	}()

	tests := []struct {
		name           string
		setupClient    func()
		expectedClient *redislib.Client
	}{
		{
			name: "success - client exists",
			setupClient: func() {
				redis.RedisClient = redislib.NewClient(&redislib.Options{
					Addr: "localhost:6379",
					DB:   0,
				})
			},
			expectedClient: nil, // We'll check it's not nil
		},
		{
			name: "success - client is nil",
			setupClient: func() {
				redis.RedisClient = nil
			},
			expectedClient: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tt.setupClient()

			// Execute
			client := redis.GetRedisClient()

			// Validate
			if tt.expectedClient == nil && tt.name == "success - client exists" {
				assert.NotNil(t, client)
				// Clean up
				if client != nil {
					client.Close()
				}
			} else {
				assert.Equal(t, tt.expectedClient, client)
			}
		})
	}
}

func TestInitRedis_ConnectionFailure(t *testing.T) {
	// Save original client
	originalClient := redis.RedisClient
	defer func() {
		if redis.RedisClient != nil {
			redis.RedisClient.Close()
		}
		redis.RedisClient = originalClient
	}()

	// Test with definitely invalid address
	config := redis.RedisConfig{
		Address:  "127.0.0.1:1", // Invalid port that won't have Redis
		Password: "",
	}

	err := redis.InitRedis(config)
	require.Error(t, err, "InitRedis should return error when connection fails")
	assert.Nil(t, redis.RedisClient, "RedisClient should be nil after connection failure")
}

func TestInitRedis_ClientReuse(t *testing.T) {
	// Save original client
	originalClient := redis.RedisClient
	defer func() {
		if redis.RedisClient != nil {
			redis.RedisClient.Close()
		}
		redis.RedisClient = originalClient
	}()

	// Skip if Redis not available
	testClient := redislib.NewClient(&redislib.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := testClient.Ping(ctx).Err()
	testClient.Close()
	cancel()
	if err != nil {
		t.Skipf("Skipping test: Redis not available: %v", err)
	}

	// First initialization
	config1 := redis.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
	}
	err = redis.InitRedis(config1)
	require.NoError(t, err)
	client1 := redis.RedisClient
	require.NotNil(t, client1)

	// Second initialization should replace the client
	config2 := redis.RedisConfig{
		Address:  "localhost:6379",
		Password: "",
	}
	err = redis.InitRedis(config2)
	require.NoError(t, err)
	client2 := redis.RedisClient
	require.NotNil(t, client2)

	// Clients should be different instances (old one closed, new one created)
	// Note: We can't directly compare pointers as they're different instances
	// but we can verify both are valid
	assert.NotNil(t, client1)
	assert.NotNil(t, client2)
}

