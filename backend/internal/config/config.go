package config

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	AppEnv             string
	AppName            string
	AppBasePath        string
	HTTPPort           string
	CORSAllowedOrigins []string
	FrontendDistDir    string
	MediaUploadDir     string
	SQLitePath         string

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
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "465"))
	if err != nil {
		return Config{}, errors.New("SMTP_PORT must be an integer")
	}
	emailVerificationTTLMinutes, err := strconv.Atoi(getEnv("EMAIL_VERIFICATION_TTL_MINUTES", "10"))
	if err != nil {
		return Config{}, errors.New("EMAIL_VERIFICATION_TTL_MINUTES must be an integer")
	}

	sqlitePath := getEnv("SQLITE_PATH", filepath.Join("data", "soulcourse.db"))
	frontendDistDir := resolveFrontendDistDir(strings.TrimSpace(os.Getenv("FRONTEND_DIST_DIR")))
	mediaUploadDir := getEnv("MEDIA_UPLOAD_DIR", filepath.Join("data", "uploads"))

	cfg := Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		AppName:            getEnv("APP_NAME", "选科π"),
		AppBasePath:        normalizeBasePath(os.Getenv("APP_BASE_PATH")),
		HTTPPort:           getEnv("HTTP_PORT", "1309"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5712,http://127.0.0.1:5712")),
		FrontendDistDir:    frontendDistDir,
		MediaUploadDir:     mediaUploadDir,
		SQLitePath:         sqlitePath,

		JWTSecret:         getEnv("JWT_SECRET", "replace-me-before-production"),
		AdminToken:        strings.TrimSpace(os.Getenv("ADMIN_TOKEN")),
		AdminEmail:        strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),

		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIBaseURL: getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModel:   getEnv("AI_MODEL", "deepseek-chat"),

		SMTPHost:                    strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                    smtpPort,
		SMTPUsername:                strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:                os.Getenv("SMTP_PASSWORD"),
		SMTPFromEmail:               strings.TrimSpace(os.Getenv("SMTP_FROM_EMAIL")),
		SMTPFromName:                getEnv("SMTP_FROM_NAME", getEnv("APP_NAME", "选科π")),
		SMTPUseTLS:                  getEnvBool("SMTP_USE_TLS", true),
		SMTPStartTLS:                getEnvBool("SMTP_START_TLS", false),
		EmailVerificationTTLMinutes: emailVerificationTTLMinutes,
		EmailVerificationSubject:    getEnv("EMAIL_VERIFICATION_SUBJECT", "选科π邮箱验证码"),
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	cfg.SQLitePath = filepath.Clean(cfg.SQLitePath)
	if cfg.FrontendDistDir != "" {
		cfg.FrontendDistDir = filepath.Clean(cfg.FrontendDistDir)
	}
	cfg.MediaUploadDir = filepath.Clean(cfg.MediaUploadDir)
	return cfg, nil
}

func (c Config) HTTPAddress() string {
	return ":" + c.HTTPPort
}

func (c Config) RoutePath(relativePath string) string {
	normalized := normalizeRoutePath(relativePath)
	if c.AppBasePath == "" {
		return normalized
	}
	if normalized == "/" {
		return c.AppBasePath
	}
	return c.AppBasePath + normalized
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

func resolveFrontendDistDir(explicit string) string {
	if explicit != "" {
		return explicit
	}
	candidates := []string{
		filepath.Join("frontend", "dist"),
		filepath.Join("..", "frontend", "dist"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func normalizeBasePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || trimmed == "/" {
		return ""
	}
	return "/" + strings.Trim(strings.TrimSpace(trimmed), "/")
}

func normalizeRoutePath(value string) string {
	cleaned := path.Clean("/" + strings.TrimSpace(value))
	if cleaned == "." {
		return "/"
	}
	return cleaned
}
