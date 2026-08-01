package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateProductionRejectsUnsafeDefaults(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{
			name: "default jwt secret",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "replace-me-before-production",
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "wildcard cors",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				CORSAllowedOrigins: []string{"*"},
			},
		},
		{
			name: "plaintext admin password",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				AdminPassword:      "secret",
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "http cors origin",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				MetricsToken:       "abcdef0123456789abcdef0123456789",
				CORSAllowedOrigins: []string{"http://example.com"},
			},
		},
		{
			name: "missing metrics token",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "missing trusted proxies",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				MetricsToken:       "abcdef0123456789abcdef0123456789",
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "trust all proxy range",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				MetricsToken:       "abcdef0123456789abcdef0123456789",
				TrustedProxies:     []string{"0.0.0.0/0"},
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "invalid trusted proxy",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				MetricsToken:       "abcdef0123456789abcdef0123456789",
				TrustedProxies:     []string{"not-a-proxy"},
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
		{
			name: "missing smtp",
			cfg: Config{
				AppEnv:             "production",
				JWTSecret:          "0123456789abcdef0123456789abcdef",
				MetricsToken:       "abcdef0123456789abcdef0123456789",
				TrustedProxies:     []string{"127.0.0.1"},
				CORSAllowedOrigins: []string{"https://example.com"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.ValidateProduction(); err == nil {
				t.Fatal("expected production validation error")
			}
		})
	}
}

func TestValidateProductionAcceptsHardenedConfig(t *testing.T) {
	cfg := Config{
		AppEnv:            "prod",
		DatabaseDriver:    "postgres",
		DatabaseURL:       "postgres://app:secret@db.example.com:5432/soulcourse?sslmode=require",
		StorageDriver:     "s3",
		S3Endpoint:        "https://s3.example.com",
		S3Bucket:          "soulcourse",
		S3Region:          "cn-south-1",
		S3CDNBaseURL:      "https://cdn.example.com",
		JWTSecret:         "0123456789abcdef0123456789abcdef",
		AdminToken:        "abcdef0123456789abcdef0123456789",
		AdminEmail:        "admin@example.com",
		AdminPasswordHash: "$2a$12$0123456789abcdef0123456789012345678901234567890123",
		MetricsToken:      "metrics0123456789abcdef0123456789",
		TrustedProxies:    []string{"127.0.0.1", "10.0.0.0/8"},
		CORSAllowedOrigins: []string{
			"https://example.com",
			"https://admin.example.com",
		},
		SMTPHost:      "smtp.example.com",
		SMTPUsername:  "mailer@example.com",
		SMTPPassword:  "smtp-secret",
		SMTPFromEmail: "no-reply@example.com",
		SMTPUseTLS:    true,
	}

	if err := cfg.ValidateProduction(); err != nil {
		t.Fatalf("expected hardened config to pass: %v", err)
	}
}

func TestValidateProductionRejectsSQLiteDatabase(t *testing.T) {
	cfg := Config{AppEnv: "production", DatabaseDriver: "sqlite"}
	if err := cfg.ValidateProduction(); err == nil || err.Error() != "DATABASE_DRIVER must be postgres in production unless ALLOW_SQLITE_PRODUCTION=true" {
		t.Fatalf("error = %v, want PostgreSQL production requirement", err)
	}
}

func TestValidateProductionAllowsExplicitSQLiteMode(t *testing.T) {
	cfg := Config{
		AppEnv:                "production",
		DatabaseDriver:        "sqlite",
		AllowSQLiteProduction: true,
		StorageDriver:         "s3",
		S3Endpoint:            "https://s3.example.com",
		S3Bucket:              "bucket",
		S3Region:              "region",
		S3CDNBaseURL:          "https://cdn.example.com",
		JWTSecret:             "01234567890123456789012345678901",
		MetricsToken:          "01234567890123456789012345678901",
		TrustedProxies:        []string{"127.0.0.1"},
		CORSAllowedOrigins:    []string{"https://soulcourse.cn"},
		SMTPHost:              "smtp.example.com",
		SMTPUsername:          "mailer@example.com",
		SMTPPassword:          "password",
		SMTPFromEmail:         "mailer@example.com",
		SMTPUseTLS:            true,
	}
	if err := cfg.ValidateProduction(); err != nil {
		t.Fatalf("explicit SQLite production mode should pass: %v", err)
	}
}

func TestLoadRequiresPostgresURL(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_DRIVER", "postgres")
	t.Setenv("DATABASE_URL", "")

	if _, err := Load(); err == nil || err.Error() != "DATABASE_URL is required when DATABASE_DRIVER=postgres" {
		t.Fatalf("error = %v, want missing DATABASE_URL error", err)
	}
}

func TestLoadRejectsDatabasePoolAboveProductionCap(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("DATABASE_MAX_OPEN_CONNS", "21")

	if _, err := Load(); err == nil || err.Error() != "DATABASE_MAX_OPEN_CONNS cannot exceed 20" {
		t.Fatalf("error = %v, want pool cap error", err)
	}
}

func TestValidateProductionAllowsLocalDefaultsOutsideProduction(t *testing.T) {
	cfg := Config{
		AppEnv:             "local",
		JWTSecret:          "replace-me-before-production",
		CORSAllowedOrigins: []string{"http://localhost:5712"},
	}

	if err := cfg.ValidateProduction(); err != nil {
		t.Fatalf("expected local config to pass: %v", err)
	}
}

func TestLoadParsesHTTPMaxBodyBytes(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("HTTP_MAX_BODY_BYTES", "2048")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.HTTPMaxBodyBytes != 2048 {
		t.Fatalf("HTTPMaxBodyBytes = %d, want 2048", cfg.HTTPMaxBodyBytes)
	}
}

func TestLoadRejectsInvalidHTTPMaxBodyBytes(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("HTTP_MAX_BODY_BYTES", "0")

	if _, err := Load(); err == nil {
		t.Fatal("expected Load to reject invalid HTTP_MAX_BODY_BYTES")
	}
}

func hardenedProductionConfig() Config {
	return Config{
		AppEnv: "production", DatabaseDriver: "postgres", DatabaseURL: "postgres://db/app",
		StorageDriver: "s3", S3Endpoint: "https://s3.example.com", S3Bucket: "bucket", S3Region: "region", S3CDNBaseURL: "https://cdn.example.com",
		JWTSecret: strings.Repeat("j", 32), AdminRole: "super_admin", MetricsToken: strings.Repeat("m", 32),
		TrustedProxies: []string{"127.0.0.1"}, CORSAllowedOrigins: []string{"https://app.example.com"},
		SMTPHost: "smtp.example.com", SMTPUsername: "mailer", SMTPPassword: "secret", SMTPFromEmail: "mail@example.com", SMTPUseTLS: true,
	}
}

func TestValidateProductionFailFastMatrix(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{name: "missing database URL", edit: func(c *Config) { c.DatabaseURL = " " }, want: "DATABASE_URL is required"},
		{name: "local storage", edit: func(c *Config) { c.StorageDriver = "local" }, want: "STORAGE_DRIVER=s3"},
		{name: "incomplete S3", edit: func(c *Config) { c.S3CDNBaseURL = "" }, want: "S3_ENDPOINT, S3_BUCKET, S3_REGION and S3_CDN_BASE_URL"},
		{name: "short JWT", edit: func(c *Config) { c.JWTSecret = strings.Repeat("j", 31) }, want: "JWT_SECRET"},
		{name: "short admin token", edit: func(c *Config) { c.AdminToken = "short" }, want: "ADMIN_TOKEN"},
		{name: "admin hash missing", edit: func(c *Config) { c.AdminEmail = "admin@example.com" }, want: "ADMIN_PASSWORD_HASH"},
		{name: "unknown admin role", edit: func(c *Config) { c.AdminRole = "owner" }, want: "ADMIN_ROLE"},
		{name: "IPv6 trust all", edit: func(c *Config) { c.TrustedProxies = []string{"::/0"} }, want: "cannot trust all"},
		{name: "SMTP transport insecure", edit: func(c *Config) { c.SMTPUseTLS = false }, want: "SMTP must use TLS or STARTTLS"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := hardenedProductionConfig()
			test.edit(&cfg)
			if err := cfg.ValidateProduction(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsInvalidOperationalSettings(t *testing.T) {
	tests := []struct{ key, value, want string }{
		{key: "DATABASE_MAX_OPEN_CONNS", value: "zero", want: "positive integer"},
		{key: "DATABASE_MAX_IDLE_CONNS", value: "-1", want: "non-negative integer"},
		{key: "DATABASE_CONNECT_TIMEOUT", value: "0s", want: "positive duration"},
		{key: "DATABASE_QUERY_TIMEOUT", value: "invalid", want: "positive duration"},
		{key: "DATABASE_HEALTH_TIMEOUT", value: "-1s", want: "positive duration"},
		{key: "SMTP_PORT", value: "smtp", want: "SMTP_PORT must be an integer"},
		{key: "EMAIL_VERIFICATION_TTL_MINUTES", value: "ten", want: "must be an integer"},
		{key: "EMAIL_VERIFICATION_COOLDOWN_SECONDS", value: "0", want: "positive integer"},
		{key: "HTTP_MAX_BODY_BYTES", value: "overflow-overflow", want: "positive integer"},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			t.Setenv("APP_ENV", "local")
			t.Setenv("JWT_SECRET", "test-secret")
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestLoadRejectsIdlePoolAboveOpenPoolAndBadDrivers(t *testing.T) {
	t.Run("idle above open", func(t *testing.T) {
		t.Setenv("APP_ENV", "local")
		t.Setenv("JWT_SECRET", "test-secret")
		t.Setenv("DATABASE_MAX_OPEN_CONNS", "5")
		t.Setenv("DATABASE_MAX_IDLE_CONNS", "6")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "cannot exceed DATABASE_MAX_OPEN_CONNS") {
			t.Fatalf("error = %v, want idle pool limit error", err)
		}
	})

	tests := []struct{ name, key, value, want string }{
		{name: "database driver", key: "DATABASE_DRIVER", value: "mysql", want: "DATABASE_DRIVER must be sqlite or postgres"},
		{name: "storage driver", key: "STORAGE_DRIVER", value: "filesystem", want: "STORAGE_DRIVER must be local or s3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("APP_ENV", "local")
			t.Setenv("JWT_SECRET", "test-secret")
			t.Setenv(test.key, test.value)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestConfigRoutingAndParsingHelpers(t *testing.T) {
	cfg := Config{HTTPHost: "127.0.0.1", HTTPPort: "1309", AppBasePath: "/app"}
	if cfg.HTTPAddress() != "127.0.0.1:1309" {
		t.Fatalf("HTTPAddress = %q", cfg.HTTPAddress())
	}
	for input, want := range map[string]string{"": "/app", "/": "/app", "api/v1/../health": "/app/api/health"} {
		if got := cfg.RoutePath(input); got != want {
			t.Errorf("RoutePath(%q) = %q, want %q", input, got, want)
		}
	}
	if got := (Config{}).RoutePath("health"); got != "/health" {
		t.Fatalf("root RoutePath = %q", got)
	}
	if normalizeBasePath(" /app/ ") != "/app" || normalizeBasePath("/") != "" {
		t.Fatal("base path normalization failed")
	}
	if value, err := getPositiveEnvDuration("UNSET_TEST_DURATION", 3*time.Second); err != nil || value != 3*time.Second {
		t.Fatalf("duration fallback = %s, %v", value, err)
	}
}

func TestValidateOTLP(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantError bool
	}{
		{"disabled", Config{}, false},
		{"tls", Config{OTLPEndpoint: "https://collector.example.com/v1/traces", OTLPServiceName: "soulcourse"}, false},
		{"insecure", Config{OTLPEndpoint: "http://127.0.0.1:4318", OTLPServiceName: "soulcourse", OTLPInsecure: true}, false},
		{"http without insecure", Config{OTLPEndpoint: "http://collector:4318", OTLPServiceName: "soulcourse"}, true},
		{"credentials", Config{OTLPEndpoint: "https://user:pass@collector", OTLPServiceName: "soulcourse"}, true},
		{"query", Config{OTLPEndpoint: "https://collector?v=1", OTLPServiceName: "soulcourse"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.ValidateOTLP() != nil; got != tt.wantError {
				t.Fatalf("ValidateOTLP error=%v, want error=%v", tt.cfg.ValidateOTLP(), tt.wantError)
			}
		})
	}
}
