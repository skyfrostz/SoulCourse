package httpserver

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/logx"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

type provinceCoverage struct {
	Province       string `json:"province"`
	CoverageStatus string `json:"coverageStatus"`
}

func TestRealDataEndpointsExposeCoverageAndSources(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cfg := config.Config{
		AppEnv:             "test",
		SQLitePath:         filepath.Join(tempDir, "real-data.db"),
		MediaUploadDir:     filepath.Join(tempDir, "uploads"),
		CORSAllowedOrigins: []string{"http://localhost:5173"},
	}
	db, err := storage.NewSQLiteDB(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	forum := service.NewForumService(sqliterepo.NewForumRepository(db), cfg, nil)
	server := NewServer(cfg, logx.New(io.Discard, logx.LevelError), db, forum)

	provinces := getJSON[struct {
		Data struct {
			Provinces []provinceCoverage `json:"provinces"`
		} `json:"data"`
	}](t, server, "/api/v1/provinces")
	if !hasProvinceCoverage(provinces.Data.Provinces, "广东", "verified") {
		t.Fatalf("广东 reviewed coverage missing: %#v", provinces.Data.Provinces)
	}

	policies := getJSON[struct {
		Data struct {
			Policies []struct {
				ID             string `json:"id"`
				CoverageStatus string `json:"coverageStatus"`
				Source         struct {
					Name string `json:"name"`
					URL  string `json:"url"`
				} `json:"source"`
			} `json:"policies"`
		} `json:"data"`
	}](t, server, "/api/v1/policies")
	if len(policies.Data.Policies) == 0 || policies.Data.Policies[0].Source.Name == "" {
		t.Fatalf("policy source metadata missing: %#v", policies.Data.Policies)
	}

	requirements := getJSON[struct {
		Data struct {
			Requirements []struct {
				ID             string `json:"id"`
				CoverageStatus string `json:"coverageStatus"`
				Methodology    string `json:"methodology"`
			} `json:"requirements"`
		} `json:"data"`
	}](t, server, "/api/v1/requirements")
	if len(requirements.Data.Requirements) == 0 || requirements.Data.Requirements[0].Methodology == "" {
		t.Fatalf("requirement methodology missing: %#v", requirements.Data.Requirements)
	}

	sourceID := seedContentSource(t, db)
	source := getJSON[struct {
		Data struct {
			ID             int64  `json:"id"`
			SourceURL      string `json:"sourceUrl"`
			CoverageStatus string `json:"coverageStatus"`
		} `json:"data"`
	}](t, server, "/api/v1/sources/"+stringInt64(sourceID))
	if source.Data.SourceURL == "" || source.Data.CoverageStatus != "unverified" {
		t.Fatalf("source metadata invalid: %#v", source.Data)
	}
}

func seedContentSource(t *testing.T, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) int64 {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := db.ExecContext(context.Background(), `
		INSERT INTO posts (author_name, author_role, title, content, track, electives, category, grade, province, created_at, updated_at)
		VALUES ('来源测试', 'counselor', '来源追溯测试', '来源追溯测试内容', 'physics', '["chemistry","biology"]', 'data', '高一', '广东', ?, ?)
	`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	postID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	result, err = db.ExecContext(context.Background(), `
		INSERT INTO content_sources (post_id, source_platform, source_url, source_note_id, source_title, source_author, transformation_note, captured_at)
		VALUES (?, 'official', 'https://example.edu/source.pdf', 'source-1', '官方来源', '考试院', '逐行复算', ?)
	`, postID, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func getJSON[T any](t *testing.T, server *http.Server, path string) T {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	server.Handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
	}
	var payload T
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func hasProvinceCoverage(items []provinceCoverage, province string, status string) bool {
	for _, item := range items {
		if item.Province == province && item.CoverageStatus == status {
			return true
		}
	}
	return false
}
