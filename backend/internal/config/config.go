package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv             string
	AppName            string
	HTTPPort           string
	CORSAllowedOrigins []string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret string

	AIAPIKey  string
	AIBaseURL string
	AIModel   string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("REDIS_DB must be an integer: %w", err)
	}

	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		AppName:            getEnv("APP_NAME", "subject-choice-forum"),
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174")),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "forum"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "forum_dev_password"),
		PostgresDB:       getEnv("POSTGRES_DB", "forum"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,

		JWTSecret: getEnv("JWT_SECRET", "replace-me-before-production"),

		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIBaseURL: getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModel:   getEnv("AI_MODEL", "deepseek-chat"),
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func (c Config) HTTPAddress() string {
	return ":" + c.HTTPPort
}

func (c Config) PostgresDSN() string {
	values := url.Values{}
	values.Set("sslmode", c.PostgresSSLMode)

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.PostgresUser, c.PostgresPassword),
		Host:     c.PostgresHost + ":" + c.PostgresPort,
		Path:     c.PostgresDB,
		RawQuery: values.Encode(),
	}
	return u.String()
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
