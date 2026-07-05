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
	MediaUploadDir     string

	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret         string
	AdminToken        string
	AdminEmail        string
	AdminPassword     string
	AdminPasswordHash string

	AIAPIKey  string
	AIBaseURL string
	AIModel   string

	SMTPHost                    string
	SMTPPort                    int
	SMTPUsername                string
	SMTPPassword                string
	SMTPFromEmail               string
	SMTPFromName                string
	SMTPUseTLS                  bool
	SMTPStartTLS                bool
	EmailVerificationTTLMinutes int
	EmailVerificationSubject    string
}

func Load() (Config, error) {
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("REDIS_DB must be an integer: %w", err)
	}
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "465"))
	if err != nil {
		return Config{}, fmt.Errorf("SMTP_PORT must be an integer: %w", err)
	}
	emailVerificationTTLMinutes, err := strconv.Atoi(getEnv("EMAIL_VERIFICATION_TTL_MINUTES", "10"))
	if err != nil {
		return Config{}, fmt.Errorf("EMAIL_VERIFICATION_TTL_MINUTES must be an integer: %w", err)
	}

	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		AppName:            getEnv("APP_NAME", "选科π"),
		HTTPPort:           getEnv("HTTP_PORT", "8080"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174,http://localhost:5175,http://127.0.0.1:5175")),
		MediaUploadDir:     getEnv("MEDIA_UPLOAD_DIR", "uploads"),

		PostgresHost:     getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:     getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:     getEnv("POSTGRES_USER", "forum"),
		PostgresPassword: getEnv("POSTGRES_PASSWORD", "forum_dev_password"),
		PostgresDB:       getEnv("POSTGRES_DB", "forum"),
		PostgresSSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       redisDB,

		JWTSecret:         getEnv("JWT_SECRET", "replace-me-before-production"),
		AdminToken:        os.Getenv("ADMIN_TOKEN"),
		AdminEmail:        strings.ToLower(os.Getenv("ADMIN_EMAIL")),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),

		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIBaseURL: getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModel:   getEnv("AI_MODEL", "deepseek-chat"),

		SMTPHost:                    os.Getenv("SMTP_HOST"),
		SMTPPort:                    smtpPort,
		SMTPUsername:                os.Getenv("SMTP_USERNAME"),
		SMTPPassword:                os.Getenv("SMTP_PASSWORD"),
		SMTPFromEmail:               os.Getenv("SMTP_FROM_EMAIL"),
		SMTPFromName:                getEnv("SMTP_FROM_NAME", getEnv("APP_NAME", "选科π")),
		SMTPUseTLS:                  getEnvBool("SMTP_USE_TLS", true),
		SMTPStartTLS:                getEnvBool("SMTP_START_TLS", false),
		EmailVerificationTTLMinutes: emailVerificationTTLMinutes,
		EmailVerificationSubject:    getEnv("EMAIL_VERIFICATION_SUBJECT", "选科π邮箱验证码"),
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

func (c Config) SMTPEnabled() bool {
	return c.SMTPHost != "" && c.SMTPUsername != "" && c.SMTPPassword != "" && c.SMTPFromEmail != ""
}

func getEnv(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
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
