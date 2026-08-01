package handler

import (
	"bytes"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/middleware"
)

func TestTypedRealDataHandlersPostgres(t *testing.T) {
	url := os.Getenv("POSTGRES_ADMIN_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_ADMIN_TEST_URL is not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	h := NewAdminHandler(config.Config{DatabaseDriver: "postgres"}, nil, db, middleware.NewAdminSessionStore(0))
	provinceCode := "handler-test-gd"
	for _, statement := range []string{`DELETE FROM requirements WHERE title = 'handler typed requirement'`, `DELETE FROM policies WHERE title IN ('handler typed policy','handler typed requirement')`, `DELETE FROM sources WHERE name = 'handler typed source'`, `DELETE FROM provinces WHERE code = $1`} {
		var err error
		if strings.Contains(statement, "$1") {
			_, err = db.Exec(statement, provinceCode)
		} else {
			_, err = db.Exec(statement)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	var provinceID int64
	err = db.QueryRow(`SELECT id FROM provinces WHERE name = '广东' ORDER BY id LIMIT 1`).Scan(&provinceID)
	if errors.Is(err, sql.ErrNoRows) {
		if err := db.QueryRow(`INSERT INTO provinces (code,name,region,coverage_status,data_year,records_count,captured_at,methodology) VALUES ($1,'广东','华南','verified',2025,2,now(),'official fixture') RETURNING id`, provinceCode).Scan(&provinceID); err != nil {
			t.Fatal(err)
		}
	} else if err != nil {
		t.Fatal(err)
	}
	var sourceID string
	if err := db.QueryRow(`INSERT INTO sources (province_id,name,url,file_hash,data_year,scope,captured_at,methodology,coverage_status) VALUES ($1,'handler typed source','https://example.com/source','handler-test-hash',2025,'广东',now(),'official fixture','verified') RETURNING id::text`, provinceID).Scan(&sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO policies (province_id,source_id,title,type,scope,coverage_status,data_year,captured_at,methodology,summary,tags,url) VALUES ($1,$2,'handler typed policy','招生政策','广东','verified',2025,now(),'official fixture','summary','["physics"]','https://example.com/policy'), ($1,$2,'handler typed requirement','招生要求','广东','verified',2025,now(),'official fixture','summary','["physics"]','https://example.com/requirement')`, provinceID, sourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO requirements (province_id,source_id,title,major_code,type,scope,required_subjects,coverage_status,data_year,captured_at,methodology,summary,tags,url) VALUES ($1,$2,'handler typed requirement','A01','选科要求','广东','["physics","chemistry"]','verified',2025,now(),'official fixture','summary','["stem"]','https://example.com/requirement')`, provinceID, sourceID); err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(middleware.RequestID())
	r.GET("/provinces", h.ListProvinces)
	r.GET("/policies", h.ListPolicies)
	r.GET("/requirements", h.ListRequirements)
	r.GET("/sources/:id", h.GetSource)
	for _, tc := range []struct{ path, want string }{{"/provinces", `"province":"广东"`}, {"/policies", `"title":"handler typed policy"`}, {"/requirements", `"requiredSubjects":["physics","chemistry"]`}, {"/sources/" + sourceID, `"fileHash":"handler-test-hash"`}} {
		res := performHandlerRequest(r, http.MethodGet, tc.path, "")
		if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), tc.want) {
			t.Fatalf("%s status=%d body=%s", tc.path, res.Code, res.Body.String())
		}
	}
}

func TestInspectStoredImageBranches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	data := handlerTestPNG(t)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	size, typ, width, height, err := inspectStoredImage(path)
	if err != nil || size != int64(len(data)) || typ != "image/png" || width != 2 || height != 2 {
		t.Fatalf("inspect success: size=%d type=%s dimensions=%dx%d err=%v", size, typ, width, height, err)
	}
	if _, _, _, _, err := inspectStoredImage(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing image should fail")
	}
	if _, _, _, _, err := inspectStoredImageReader(bytes.NewReader([]byte("bad")), 3); err == nil {
		t.Fatal("bad image should fail")
	}
	if _, _, _, _, err := inspectStoredImageReader(bytes.NewReader(data), int64(len(data)+1)); err == nil {
		t.Fatal("size mismatch should fail")
	}
	if _, _, _, _, err := inspectStoredImageReader(bytes.NewReader(data), int64(len(data))); err != nil {
		t.Fatalf("valid reader failed: %v", err)
	}
}

func TestAdminPayloadAndNormalizationBranches(t *testing.T) {
	record := AdminContentRecord{Type: "提问", Title: "史政地选科困惑", Summary: "历史方向", Owner: "家长", Scope: "广东"}
	if got := decodePayloadMap([]byte(`{"bad"`)); len(got) != 0 {
		t.Fatalf("invalid payload=%v", got)
	}
	if got := payloadString(map[string]any{"n": float64(12), "f": 1.5, "x": 7}, "n"); got != "12" {
		t.Fatalf("integer payload=%q", got)
	}
	if got := payloadString(map[string]any{"f": 1.5}, "f"); got != "1.5" {
		t.Fatalf("float payload=%q", got)
	}
	if payloadInt64(map[string]any{"id": "bad"}, "id") != 0 || payloadInt64(map[string]any{"id": float64(42)}, "id") != 42 {
		t.Fatal("payload int conversion failed")
	}
	if got := payloadStringSlice(map[string]any{"v": []string{" a ", "a", ""}}, "v"); len(got) != 1 || got[0] != "a" {
		t.Fatalf("string slice=%v", got)
	}
	if got := payloadStringSlice(map[string]any{"v": []any{" a ", 2, ""}}, "v"); len(got) != 2 {
		t.Fatalf("any slice=%v", got)
	}
	if got := payloadStringSlice(map[string]any{"v": "a, b,a"}, "v"); len(got) != 2 {
		t.Fatalf("csv slice=%v", got)
	}
	if inferAuthorRole(record) != "parent" || normalizeTrack("", record) != "history" || normalizeCategory("", AdminContentRecord{Type: "政策"}) != "data" {
		t.Fatalf("record normalization failed: role=%s track=%s category=%s", inferAuthorRole(record), normalizeTrack("", record), normalizeCategory("", AdminContentRecord{Type: "政策"}))
	}
	for _, value := range []string{"experience", "question", "data"} {
		if normalizeCategory(value, record) != value {
			t.Fatal("category passthrough failed")
		}
	}
	for _, value := range []string{"physics", "history"} {
		if normalizeTrack(value, record) != value {
			t.Fatal("track passthrough failed")
		}
	}
	for _, tag := range []string{"物化政", "物化地", "物生地", "史政地", "史化生", "unknown"} {
		if len(normalizeElectives(nil, AdminContentRecord{Tags: []string{tag}})) != 2 {
			t.Fatalf("electives %s", tag)
		}
	}
	if got := normalizeElectives([]string{"chemistry", "biology", "chemistry"}, record); len(got) != 2 {
		t.Fatalf("electives passthrough=%v", got)
	}
	for _, tc := range []struct {
		record AdminContentRecord
		want   string
	}{
		{AdminContentRecord{Type: "教师分享"}, "teacher"},
		{AdminContentRecord{Type: "研究数据"}, "counselor"},
		{AdminContentRecord{Type: "学生记录"}, "student"},
	} {
		if got := inferAuthorRole(tc.record); got != tc.want {
			t.Fatalf("author role=%s want=%s", got, tc.want)
		}
	}
	for _, tc := range []struct {
		record AdminContentRecord
		want   string
	}{
		{AdminContentRecord{Type: "提问"}, "question"},
		{AdminContentRecord{Type: "普通分享"}, "experience"},
	} {
		if got := normalizeCategory("", tc.record); got != tc.want {
			t.Fatalf("category=%s want=%s", got, tc.want)
		}
	}
	if defaultString("", "fallback") != "fallback" || defaultString(" value ", "fallback") != " value " {
		t.Fatal("default string branch failed")
	}
	if normalizeJSON(nil) != "{}" || normalizeJSON([]byte("bad")) != "{}" || normalizeJSON([]byte(`{"ok":true}`)) != `{"ok":true}` {
		t.Fatal("normalize json branch failed")
	}
	if payloadStringSlice(map[string]any{"v": 3}, "v") != nil || payloadStringSlice(map[string]any{"v": ""}, "v") != nil {
		t.Fatal("empty slice branches failed")
	}
	if got := buildSyncedPostPayload(AdminContentRecord{Summary: "摘要", Scope: "广东"}, map[string]any{}); got.Content != "摘要" || got.Grade != "高一" || got.Province != "广东" {
		t.Fatalf("synced defaults=%+v", got)
	}
	if got := buildSyncedPostPayload(AdminContentRecord{}, map[string]any{"content": "正文", "postId": "9", "track": "history", "electives": []any{"chemistry", "biology"}, "category": "data", "grade": "高二", "province": "全国", "imageUrls": []any{"/a.png"}}); got.PostID != 9 || got.Content != "正文" || got.Grade != "高二" || len(got.ImageURLs) != 1 {
		t.Fatalf("synced explicit=%+v", got)
	}
	for _, tc := range []struct {
		query string
		want  int
	}{{"", 30}, {"?limit=0", 30}, {"?limit=999", 100}, {"?limit=12", 12}, {"?limit=bad", 30}} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/x"+tc.query, nil)
		if got := parseLimit(c, 30, 100); got != tc.want {
			t.Fatalf("parse limit %q=%d want=%d", tc.query, got, tc.want)
		}
	}
	if (&ForumHandler{}).routePath("/x") != "/x" || (&ForumHandler{appBasePath: "/app"}).routePath("/x") != "/app/x" {
		t.Fatal("route path branch failed")
	}
	if normalizeProvince("") != "全国" || normalizeProvince("首页") != "全国" || normalizeProvince(" 广东 ") != "广东" {
		t.Fatal("province normalization failed")
	}
	plain := &AdminHandler{cfg: config.Config{AdminPassword: "secret"}}
	if !plain.validAdminPassword("secret") || plain.validAdminPassword("wrong") {
		t.Fatal("plain admin password branch failed")
	}
	if got := parseJSONStringSlice(`["a","b"]`); len(got) != 2 || len(parseJSONStringSlice("bad")) != 0 || len(parseJSONStringSlice("")) != 0 {
		t.Fatal("json slice parsing failed")
	}
	if marshalJSON(map[string]any{"ok": true}) == "" || marshalJSON(make(chan int)) != "{}" {
		t.Fatal("json marshaling branches failed")
	}
}

func handlerTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
