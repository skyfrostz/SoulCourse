package httpserver

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/logx"
	"subject-choice-forum/backend/internal/repository"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestPublicBetaPostgresHTTPJourney(t *testing.T) {
	postgresURL := os.Getenv("POSTGRES_HTTP_TEST_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_HTTP_TEST_URL is not set")
	}
	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv: "local", DatabaseDriver: "postgres", DatabaseURL: postgresURL,
		MediaUploadDir: filepath.Join(tempDir, "uploads"), StorageDriver: "local",
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		AdminEmail:         "admin@example.com", AdminPassword: "admin-password",
		EmailVerificationTTLMinutes: 10, EmailVerificationCooldownSeconds: 1,
		EmailVerificationEmailHourlyLimit: 20, EmailVerificationIPHourlyLimit: 40,
		EmailVerificationMaxValidationAttempts: 5, HTTPMaxBodyBytes: 1024 * 1024,
		DatabaseMaxOpenConns: 20, DatabaseMaxIdleConns: 5,
		DatabaseConnectTimeout: 5 * time.Second, DatabaseQueryTimeout: 5 * time.Second,
	}
	database, err := storage.NewDatabase(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	db := database.DB
	defer db.Close()
	forumRepository, err := repository.NewForumRepository(database)
	if err != nil {
		t.Fatal(err)
	}
	forum := service.NewForumService(forumRepository, cfg, nil)
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)
	if server == nil {
		t.Fatal("server initialization failed")
	}
	verifyPostgresRealDataEndpoints(t, server, db)
	app := httptest.NewServer(server.Handler)
	defer app.Close()
	runPublicBetaSmokeUserAndModerationJourney(t, app, db)
}

func verifyPostgresRealDataEndpoints(t *testing.T, server *http.Server, db *sql.DB) {
	t.Helper()
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("postgres-http-%d", time.Now().UnixNano()))))
	var provinceID int64
	err := db.QueryRow(`
		INSERT INTO provinces (code, name, region, coverage_status, data_year, records_count, captured_at, methodology)
		VALUES ('GD', '广东', '华南', 'verified', 2026, 2, now(), '广东官方附件逐行复核')
		ON CONFLICT (code) DO UPDATE SET coverage_status = EXCLUDED.coverage_status, data_year = EXCLUDED.data_year,
			records_count = EXCLUDED.records_count, methodology = EXCLUDED.methodology
		RETURNING id`).Scan(&provinceID)
	if err != nil {
		t.Fatal(err)
	}
	var sourceID string
	err = db.QueryRow(`
		INSERT INTO sources (province_id, name, url, asset_key, file_hash, data_year, scope, captured_at, methodology, coverage_status)
		VALUES ($1, '广东省教育考试院', 'https://eea.gd.gov.cn/source.pdf', 'sources/gd/source.pdf', $2, 2026, '广东', now(), '保留原文件并校验哈希', 'verified')
		RETURNING id::text`, provinceID, hash).Scan(&sourceID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM requirements WHERE source_id = $1::uuid`, sourceID)
		_, _ = db.Exec(`DELETE FROM policies WHERE source_id = $1::uuid`, sourceID)
		_, _ = db.Exec(`DELETE FROM sources WHERE id = $1::uuid`, sourceID)
	})
	if _, err := db.Exec(`
		INSERT INTO policies (province_id, source_id, title, type, scope, coverage_status, data_year, captured_at, summary, methodology, tags, url)
		VALUES ($1, $2::uuid, '广东选科政策测试', '政策', '广东', 'verified', 2026, now(), '测试摘要', '官方文件逐条复核', '["广东","选科"]', 'https://eea.gd.gov.cn/policy')`, provinceID, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO requirements (province_id, source_id, title, major_code, type, scope, required_subjects, coverage_status, data_year, captured_at, summary, methodology, tags, url)
		VALUES ($1, $2::uuid, '临床医学选科要求测试', '100201K', '专业要求', '广东', '["physics","chemistry"]', 'verified', 2026, now(), '测试摘要', '官方文件逐条复核', '["医学"]', 'https://eea.gd.gov.cn/requirements')`, provinceID, sourceID); err != nil {
		t.Fatal(err)
	}

	provinces := getJSON[struct {
		Data struct {
			Provinces []struct {
				Province       string `json:"province"`
				CoverageStatus string `json:"coverageStatus"`
				DataYear       int    `json:"dataYear"`
			} `json:"provinces"`
		} `json:"data"`
	}](t, server, "/api/v1/provinces")
	if len(provinces.Data.Provinces) == 0 || provinces.Data.Provinces[0].Province != "广东" || provinces.Data.Provinces[0].CoverageStatus != "verified" || provinces.Data.Provinces[0].DataYear != 2026 {
		t.Fatalf("typed province data missing: %#v", provinces.Data.Provinces)
	}
	policies := getJSON[struct {
		Data struct {
			Policies []struct {
				FileHash string `json:"fileHash"`
				DataYear int    `json:"dataYear"`
			} `json:"policies"`
		} `json:"data"`
	}](t, server, "/api/v1/policies")
	if len(policies.Data.Policies) == 0 || policies.Data.Policies[0].FileHash != hash || policies.Data.Policies[0].DataYear != 2026 {
		t.Fatalf("typed policy data missing: %#v", policies.Data.Policies)
	}
	requirements := getJSON[struct {
		Data struct {
			Requirements []struct {
				FileHash         string   `json:"fileHash"`
				RequiredSubjects []string `json:"requiredSubjects"`
			} `json:"requirements"`
		} `json:"data"`
	}](t, server, "/api/v1/requirements")
	if len(requirements.Data.Requirements) == 0 || requirements.Data.Requirements[0].FileHash != hash || len(requirements.Data.Requirements[0].RequiredSubjects) != 2 {
		t.Fatalf("typed requirement data missing: %#v", requirements.Data.Requirements)
	}
	source := getJSON[struct {
		Data struct {
			ID       string `json:"id"`
			FileHash string `json:"fileHash"`
		} `json:"data"`
	}](t, server, "/api/v1/sources/"+sourceID)
	if source.Data.ID != sourceID || source.Data.FileHash != hash {
		t.Fatalf("typed source data missing: %#v", source.Data)
	}
}
