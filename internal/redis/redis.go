package redis

import (
	"context"
	"encoding/json"
	"flight-search-service/internal/config"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{Client: client}
}

func NewRedisClient() (*redis.Client, error) {
    client := redis.NewClient(&redis.Options{
        Addr:         config.AppConfig.RedisAddress,
        PoolSize:     config.AppConfig.RedisPollSize,              // Connection pooling 
        MinIdleConns: 3,
        DialTimeout:  5 * time.Second,
    })

    // Use a short timeout for the initial ping
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }

    return client, nil
}

// Stores any object as JSON
func (c *Cache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if c.Client == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.Client.Set(ctx, key, data, ttl).Err()
}

// Retrieves JSON and unmarshals it to variable
func (c *Cache) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	if c.Client == nil {
		return false, nil
	}

	val, err := c.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *Cache) Invalidate(ctx context.Context, key string) error {
	if c.Client == nil {
		return nil
	}
	return c.Client.Del(ctx, key).Err()
}