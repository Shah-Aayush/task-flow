package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables.
// All required fields are validated at startup — the application fails fast if
// any required variable is missing, rather than failing at runtime.
type Config struct {
	// Database connection settings
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// JWT signing secret — MUST come from environment, never hardcoded
	JWTSecret string

	// HTTP server port
	ServerPort string

	// bcrypt cost factor (12 minimum per spec)
	BcryptCost int
}

// Load reads configuration from environment variables and validates required fields.
// Exits the process immediately if any required variable is missing.
func Load(logger *slog.Logger) *Config {
	cfg := &Config{
		DBHost:     getEnvRequired("DB_HOST", logger),
		DBPort:     getEnvOrDefault("DB_PORT", "5432"),
		DBUser:     getEnvRequired("DB_USER", logger),
		DBPassword: getEnvRequired("DB_PASSWORD", logger),
		DBName:     getEnvRequired("DB_NAME", logger),
		DBSSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		JWTSecret:  getEnvRequired("JWT_SECRET", logger),
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
	}

	costStr := getEnvOrDefault("BCRYPT_COST", "12")
	cost, err := strconv.Atoi(costStr)
	if err != nil || cost < 12 {
		logger.Error("BCRYPT_COST must be a valid integer >= 12", "value", costStr)
		os.Exit(1)
	}
	cfg.BcryptCost = cost

	return cfg
}

// DatabaseURL constructs the PostgreSQL connection string (DSN) from config fields.
// Used by both pgxpool and golang-migrate.
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
	)
}

// ServerAddr returns the address string for net.Listen (":<port>").
func (c *Config) ServerAddr() string {
	return ":" + c.ServerPort
}

func getEnvRequired(key string, logger *slog.Logger) string {
	val := os.Getenv(key)
	if val == "" {
		logger.Error("required environment variable is not set", "key", key)
		os.Exit(1)
	}
	return val
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
