package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	API      APIConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret      string
	ExpiryHours int
}

type APIConfig struct {
	KeyHeader              string
	RateLimitMessagesPerSec int
}

type CORSConfig struct {
	AllowedOrigins []string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		redisDB = 0
	}

	jwtExpiry, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "168"))
	if err != nil {
		jwtExpiry = 168
	}

	rateLimit, err := strconv.Atoi(getEnv("RATE_LIMIT_MESSAGES_PER_SECOND", "10"))
	if err != nil {
		rateLimit = 10
	}

	origins := strings.Split(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ",")

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "tullo"),
			Password: getEnv("DB_PASSWORD", "tullo_password"),
			DBName:   getEnv("DB_NAME", "tullo_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		JWT: JWTConfig{
			Secret:      getEnv("JWT_SECRET", "change-this-secret-key"),
			ExpiryHours: jwtExpiry,
		},
		API: APIConfig{
			KeyHeader:              getEnv("API_KEY_HEADER", "X-API-Key"),
			RateLimitMessagesPerSec: rateLimit,
		},
		CORS: CORSConfig{
			AllowedOrigins: origins,
		},
	}

	// Validate required fields
	if cfg.JWT.Secret == "change-this-secret-key" && cfg.Server.Env == "production" {
		return nil, fmt.Errorf("JWT_SECRET must be set in production")
	}

	return cfg, nil
}

// GetDSN returns the database connection string
func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.DBName,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
