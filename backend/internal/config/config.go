package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                 string
	AppName                string
	AppBasePath            string
	HTTPHost               string
	HTTPPort               string
	TrustedProxies         []string
	CORSAllowedOrigins     []string
	FrontendDistDir        string
	MediaUploadDir         string
	StorageDriver          string
	S3Endpoint             string
	S3Bucket               string
	S3Region               string
	S3CDNBaseURL           string
	S3ForcePathStyle       bool
	SQLitePath             string
	DatabaseDriver         string
	DatabaseURL            string
	DatabaseMaxOpenConns   int
	DatabaseMaxIdleConns   int
	DatabaseConnectTimeout time.Duration
	DatabaseQueryTimeout   time.Duration
	DatabaseHealthTimeout  time.Duration
	HTTPMaxBodyBytes       int64
	MetricsToken           string
	OTLPEndpoint           string
	OTLPServiceName        string
	OTLPInsecure           bool
	OTLPCertificate        string

	JWTSecret         string
	AdminToken        string
	AdminEmail        string
	AdminRole         string
	AdminPassword     string
	AdminPasswordHash string

	AIAPIKey  string
	AIBaseURL string
	AIModel   string

	SMTPHost                               string
	SMTPPort                               int
	SMTPUsername                           string
	SMTPPassword                           string
	SMTPFromEmail                          string
	SMTPReplyTo                            string
	SMTPFromName                           string
	SMTPUseTLS                             bool
	SMTPStartTLS                           bool
	EmailVerificationTTLMinutes            int
	EmailVerificationSubject               string
	EmailVerificationCooldownSeconds       int
	EmailVerificationEmailHourlyLimit      int
	EmailVerificationIPHourlyLimit         int
	EmailVerificationMaxValidationAttempts int
}

func Load() (Config, error) {
	databaseMaxOpenConns, err := getPositiveEnvInt("DATABASE_MAX_OPEN_CONNS", 20)
	if err != nil {
		return Config{}, err
	}
	if databaseMaxOpenConns > 20 {
		return Config{}, errors.New("DATABASE_MAX_OPEN_CONNS cannot exceed 20")
	}
	databaseMaxIdleConns, err := getNonNegativeEnvInt("DATABASE_MAX_IDLE_CONNS", 5)
	if err != nil {
		return Config{}, err
	}
	if databaseMaxIdleConns > databaseMaxOpenConns {
		return Config{}, errors.New("DATABASE_MAX_IDLE_CONNS cannot exceed DATABASE_MAX_OPEN_CONNS")
	}
	databaseConnectTimeout, err := getPositiveEnvDuration("DATABASE_CONNECT_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	databaseQueryTimeout, err := getPositiveEnvDuration("DATABASE_QUERY_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	databaseHealthTimeout, err := getPositiveEnvDuration("DATABASE_HEALTH_TIMEOUT", 2*time.Second)
	if err != nil {
		return Config{}, err
	}
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "465"))
	if err != nil {
		return Config{}, errors.New("SMTP_PORT must be an integer")
	}
	emailVerificationTTLMinutes, err := strconv.Atoi(getEnv("EMAIL_VERIFICATION_TTL_MINUTES", "10"))
	if err != nil {
		return Config{}, errors.New("EMAIL_VERIFICATION_TTL_MINUTES must be an integer")
	}
	emailVerificationCooldownSeconds, err := getPositiveEnvInt("EMAIL_VERIFICATION_COOLDOWN_SECONDS", 60)
	if err != nil {
		return Config{}, err
	}
	emailVerificationEmailHourlyLimit, err := getPositiveEnvInt("EMAIL_VERIFICATION_EMAIL_HOURLY_LIMIT", 5)
	if err != nil {
		return Config{}, err
	}
	emailVerificationIPHourlyLimit, err := getPositiveEnvInt("EMAIL_VERIFICATION_IP_HOURLY_LIMIT", 20)
	if err != nil {
		return Config{}, err
	}
	emailVerificationMaxValidationAttempts, err := getPositiveEnvInt("EMAIL_VERIFICATION_MAX_VALIDATION_ATTEMPTS", 5)
	if err != nil {
		return Config{}, err
	}
	httpMaxBodyBytes, err := getPositiveEnvInt64("HTTP_MAX_BODY_BYTES", 1*1024*1024)
	if err != nil {
		return Config{}, err
	}

	sqlitePath := getEnv("SQLITE_PATH", filepath.Join("data", "soulcourse.db"))
	frontendDistDir := resolveFrontendDistDir(strings.TrimSpace(os.Getenv("FRONTEND_DIST_DIR")))
	mediaUploadDir := getEnv("MEDIA_UPLOAD_DIR", filepath.Join("data", "uploads"))
	storageDriver := strings.ToLower(getEnv("STORAGE_DRIVER", "local"))

	cfg := Config{
		AppEnv:                 getEnv("APP_ENV", "local"),
		AppName:                getEnv("APP_NAME", "选科π"),
		AppBasePath:            normalizeBasePath(os.Getenv("APP_BASE_PATH")),
		HTTPHost:               strings.TrimSpace(os.Getenv("HTTP_HOST")),
		HTTPPort:               getEnv("HTTP_PORT", "1309"),
		TrustedProxies:         splitCSV(os.Getenv("TRUSTED_PROXIES")),
		CORSAllowedOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5712,http://127.0.0.1:5712")),
		FrontendDistDir:        frontendDistDir,
		MediaUploadDir:         mediaUploadDir,
		StorageDriver:          storageDriver,
		S3Endpoint:             strings.TrimRight(strings.TrimSpace(os.Getenv("S3_ENDPOINT")), "/"),
		S3Bucket:               strings.TrimSpace(os.Getenv("S3_BUCKET")),
		S3Region:               strings.TrimSpace(os.Getenv("S3_REGION")),
		S3CDNBaseURL:           strings.TrimRight(strings.TrimSpace(os.Getenv("S3_CDN_BASE_URL")), "/"),
		S3ForcePathStyle:       getEnvBool("S3_FORCE_PATH_STYLE", false),
		SQLitePath:             sqlitePath,
		DatabaseDriver:         strings.ToLower(getEnv("DATABASE_DRIVER", "sqlite")),
		DatabaseURL:            strings.TrimSpace(os.Getenv("DATABASE_URL")),
		DatabaseMaxOpenConns:   databaseMaxOpenConns,
		DatabaseMaxIdleConns:   databaseMaxIdleConns,
		DatabaseConnectTimeout: databaseConnectTimeout,
		DatabaseQueryTimeout:   databaseQueryTimeout,
		DatabaseHealthTimeout:  databaseHealthTimeout,
		HTTPMaxBodyBytes:       httpMaxBodyBytes,
		MetricsToken:           strings.TrimSpace(os.Getenv("METRICS_TOKEN")),
		OTLPEndpoint:           strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		OTLPServiceName:        getEnv("OTEL_SERVICE_NAME", getEnv("APP_NAME", "subject-choice-forum")),
		OTLPInsecure:           getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", false),
		OTLPCertificate:        strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_CERTIFICATE")),

		JWTSecret:         getEnv("JWT_SECRET", "replace-me-before-production"),
		AdminToken:        strings.TrimSpace(os.Getenv("ADMIN_TOKEN")),
		AdminEmail:        strings.ToLower(strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))),
		AdminRole:         strings.ToLower(getEnv("ADMIN_ROLE", "super_admin")),
		AdminPassword:     os.Getenv("ADMIN_PASSWORD"),
		AdminPasswordHash: os.Getenv("ADMIN_PASSWORD_HASH"),

		AIAPIKey:  os.Getenv("AI_API_KEY"),
		AIBaseURL: getEnv("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModel:   getEnv("AI_MODEL", "deepseek-v4-flash"),

		SMTPHost:                               strings.TrimSpace(os.Getenv("SMTP_HOST")),
		SMTPPort:                               smtpPort,
		SMTPUsername:                           strings.TrimSpace(os.Getenv("SMTP_USERNAME")),
		SMTPPassword:                           os.Getenv("SMTP_PASSWORD"),
		SMTPFromEmail:                          strings.TrimSpace(os.Getenv("SMTP_FROM_EMAIL")),
		SMTPReplyTo:                            strings.TrimSpace(os.Getenv("SMTP_REPLY_TO")),
		SMTPFromName:                           getEnv("SMTP_FROM_NAME", getEnv("APP_NAME", "选科π")),
		SMTPUseTLS:                             getEnvBool("SMTP_USE_TLS", true),
		SMTPStartTLS:                           getEnvBool("SMTP_START_TLS", false),
		EmailVerificationTTLMinutes:            emailVerificationTTLMinutes,
		EmailVerificationSubject:               getEnv("EMAIL_VERIFICATION_SUBJECT", "选科π邮箱验证码"),
		EmailVerificationCooldownSeconds:       emailVerificationCooldownSeconds,
		EmailVerificationEmailHourlyLimit:      emailVerificationEmailHourlyLimit,
		EmailVerificationIPHourlyLimit:         emailVerificationIPHourlyLimit,
		EmailVerificationMaxValidationAttempts: emailVerificationMaxValidationAttempts,
	}

	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	if cfg.DatabaseDriver != "sqlite" && cfg.DatabaseDriver != "postgres" {
		return Config{}, errors.New("DATABASE_DRIVER must be sqlite or postgres")
	}
	if cfg.DatabaseDriver == "postgres" && cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required when DATABASE_DRIVER=postgres")
	}
	if cfg.StorageDriver != "local" && cfg.StorageDriver != "s3" {
		return Config{}, errors.New("STORAGE_DRIVER must be local or s3")
	}
	if err := cfg.ValidateProduction(); err != nil {
		return Config{}, err
	}

	cfg.SQLitePath = filepath.Clean(cfg.SQLitePath)
	if cfg.FrontendDistDir != "" {
		cfg.FrontendDistDir = filepath.Clean(cfg.FrontendDistDir)
	}
	cfg.MediaUploadDir = filepath.Clean(cfg.MediaUploadDir)
	return cfg, nil
}

func (c Config) HTTPAddress() string {
	return c.HTTPHost + ":" + c.HTTPPort
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

func (c Config) Production() bool {
	env := strings.ToLower(strings.TrimSpace(c.AppEnv))
	return env == "production" || env == "prod"
}

func (c Config) ValidateProduction() error {
	if !c.Production() {
		return nil
	}
	if c.DatabaseDriver != "postgres" {
		return errors.New("DATABASE_DRIVER must be postgres in production")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required in production")
	}
	if c.StorageDriver != "s3" {
		return errors.New("STORAGE_DRIVER=s3 is required in production")
	}
	if c.S3Endpoint == "" || c.S3Bucket == "" || c.S3Region == "" || c.S3CDNBaseURL == "" {
		return errors.New("S3_ENDPOINT, S3_BUCKET, S3_REGION and S3_CDN_BASE_URL are required in production")
	}
	if c.JWTSecret == "replace-me-before-production" || len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be a production secret with at least 32 characters")
	}
	if c.AdminToken != "" && len(c.AdminToken) < 32 {
		return errors.New("ADMIN_TOKEN must be at least 32 characters in production")
	}
	if c.AdminPassword != "" {
		return errors.New("ADMIN_PASSWORD is not allowed in production; use ADMIN_PASSWORD_HASH")
	}
	if c.AdminEmail != "" && c.AdminPasswordHash == "" {
		return errors.New("ADMIN_PASSWORD_HASH is required when ADMIN_EMAIL is configured in production")
	}
	if c.AdminRole != "" && c.AdminRole != "super_admin" && c.AdminRole != "content_editor" && c.AdminRole != "moderator" {
		return errors.New("ADMIN_ROLE must be super_admin, content_editor or moderator")
	}
	if len(c.MetricsToken) < 32 {
		return errors.New("METRICS_TOKEN must be set to at least 32 characters in production")
	}
	if len(c.TrustedProxies) == 0 {
		return errors.New("TRUSTED_PROXIES must be set in production")
	}
	for _, proxy := range c.TrustedProxies {
		if proxy == "*" || strings.EqualFold(proxy, "0.0.0.0/0") || strings.EqualFold(proxy, "::/0") {
			return errors.New("TRUSTED_PROXIES cannot trust all addresses in production")
		}
		if net.ParseIP(proxy) == nil {
			if _, _, err := net.ParseCIDR(proxy); err != nil {
				return errors.New("TRUSTED_PROXIES must contain only IP addresses or CIDR ranges")
			}
		}
	}
	for _, origin := range c.CORSAllowedOrigins {
		if origin == "*" {
			return errors.New("CORS_ALLOWED_ORIGINS cannot contain * in production")
		}
		if !strings.HasPrefix(origin, "https://") {
			return errors.New("CORS_ALLOWED_ORIGINS must use https:// origins in production")
		}
	}
	if !c.SMTPEnabled() {
		return errors.New("SMTP_HOST, SMTP_USERNAME, SMTP_PASSWORD and SMTP_FROM_EMAIL are required in production")
	}
	if !c.SMTPUseTLS && !c.SMTPStartTLS {
		return errors.New("SMTP must use TLS or STARTTLS in production")
	}
	if err := c.ValidateOTLP(); err != nil {
		return err
	}
	return nil
}

func (c Config) ValidateOTLP() error {
	if strings.TrimSpace(c.OTLPEndpoint) == "" {
		return nil
	}
	u, err := url.Parse(c.OTLPEndpoint)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return errors.New("OTEL_EXPORTER_OTLP_ENDPOINT must be an absolute endpoint URL without credentials, query or fragment")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("OTEL_EXPORTER_OTLP_ENDPOINT must use http or https")
	}
	if !c.OTLPInsecure && u.Scheme != "https" {
		return errors.New("OTEL_EXPORTER_OTLP_INSECURE=true is required for an http OTLP endpoint")
	}
	if strings.TrimSpace(c.OTLPServiceName) == "" {
		return errors.New("OTEL_SERVICE_NAME cannot be empty when OTLP is enabled")
	}
	return nil
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

func getPositiveEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive integer")
	}
	return parsed, nil
}

func getNonNegativeEnvInt(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return parsed, nil
}

func getPositiveEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}

func getPositiveEnvInt64(key string, fallback int64) (int64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, errors.New(key + " must be a positive integer")
	}
	return parsed, nil
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
