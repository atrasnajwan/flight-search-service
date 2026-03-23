package config

import (
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server configuration
	ServerPort     string

	// Redis configuration
	RedisAddress  string
	RedisPollSize int
}

// Global application configuration
var AppConfig Config


func LoadConfig() {
	// Find .env file
	envPath := ".env"
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		// Try to find .env in parent directories
		envPath = filepath.Join("..", ".env")
		if _, err := os.Stat(envPath); os.IsNotExist(err) {
			envPath = filepath.Join("..", "..", ".env")
		}
	}

	// Load .env file if it exists
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(envPath); err != nil {
			log.Println("Error loading .env file")
		}
	}

	AppConfig = Config{
		ServerPort:                getEnv("PORT", "3000"),
		RedisAddress:              getEnv("REDIS_ADDRESS", "localhost:6379"),
		RedisPollSize:             getEnv("REDIS_POOL_SIZE", 10),
	}
}

// gets an environment variable or returns a default value
func getEnv[T any](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	var result any
	switch any(defaultValue).(type) {
	case string:
		result = value
	case int:
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		result = i
	default:
		return defaultValue
	}

	return result.(T)
}
