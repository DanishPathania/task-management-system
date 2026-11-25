package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// defaultJWTSecret is the well-known insecure default value.
const defaultJWTSecret = "super-secret-jwt-key-change-in-production-must-be-32-bytes!"

type Config struct {
	ServerPort         string
	ServerEnv          string
	LogLevel           string
	DatabaseURL        string
	DBMaxOpenConns     int
	DBMaxIdleConns     int
	DBConnMaxLifetime  time.Duration
	RedisURL           string
	RedisPassword      string
	JWTSecret          string
	JWTExpirationHours int
	CookieSecure       bool
	CookieSameSite     string
	CookieDomain       string
	CORSAllowedOrigins []string
}

// Load reads configuration from environment variables.
// It first tries to load a .env file from the current directory so that
// local manual development works without manually exporting every variable.
// Real environment variables (e.g. from Docker) always take precedence.
func Load() (*Config, error) {
	// Fix #2: Auto-load .env so `go run` works out-of-the-box locally.
	loadDotEnv(".env")

	cfg := &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		ServerEnv:          getEnv("SERVER_ENV", "development"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://taskuser:taskpassword@localhost:5432/taskdb?sslmode=disable"),
		DBMaxOpenConns:     getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:     getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime:  getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379/0"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		JWTSecret:          getEnv("JWT_SECRET", defaultJWTSecret),
		JWTExpirationHours: getEnvAsInt("JWT_EXPIRATION_HOURS", 24),
		CookieSecure:       getEnvAsBool("COOKIE_SECURE", false),
		CookieSameSite:     getEnv("COOKIE_SAME_SITE", "lax"),
		CookieDomain:       os.Getenv("COOKIE_DOMAIN"),
	}

	origins := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	cfg.CORSAllowedOrigins = strings.Split(origins, ",")
	for i := range cfg.CORSAllowedOrigins {
		cfg.CORSAllowedOrigins[i] = strings.TrimSpace(cfg.CORSAllowedOrigins[i])
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be configured")
	}

	// Fix #13: Alert when the public default JWT secret is in use.
	if cfg.JWTSecret == defaultJWTSecret {
		if cfg.ServerEnv == "production" {
			return nil, fmt.Errorf(
				"JWT_SECRET is set to the default insecure value — set a strong random secret before running in production",
			)
		}
		log.Println("[WARNING] JWT_SECRET is using the default insecure value. Set a strong random secret for non-local environments.")
	}

	if cfg.ServerEnv == "production" && len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters in production")
	}

	return cfg, nil
}

// loadDotEnv reads key=value pairs from filename and sets any variable
// that is not already present in the process environment (real env vars win).
func loadDotEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return // .env is optional
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Strip surrounding quotes (single or double)
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Only set if not already in the environment
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	valStr := os.Getenv(key)
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsBool(key string, defaultVal bool) bool {
	valStr := os.Getenv(key)
	if val, err := strconv.ParseBool(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvAsDuration(key string, defaultVal time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if d, err := time.ParseDuration(valStr); err == nil {
		return d
	}
	return defaultVal
}
