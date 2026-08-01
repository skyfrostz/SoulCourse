package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/http/middleware"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestImageUploadPresignPutAndComplete(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	gin.SetMode(gin.TestMode)

	tempDir := t.TempDir()
	uploadDir := filepath.Join(tempDir, "uploads")
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum.db"),
		MediaUploadDir: uploadDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	repository := sqliterepo.NewForumRepository(db)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("upload-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: "upload@example.com", Nickname: "上传用户", Role: "student", Province: "广东", Grade: "高一",
	}, string(passwordHash))
	if err != nil {
		t.Fatal(err)
	}
	forum := service.NewForumService(repository, config.Config{JWTSecret: "test-secret"}, nil)
	session, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "upload-password"})
	if err != nil {
		t.Fatal(err)
	}
	forumHandler := NewForumHandler(forum, nil, false, uploadDir, "")
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.OptionalAuth(forum))
	router.POST("/api/v1/uploads/images/presign", middleware.RequireAuth(forum), forumHandler.PresignImageUpload)
	router.PUT("/api/v1/uploads/images/:id/object", middleware.RequireAuth(forum), forumHandler.PutImageUploadObject)
	router.POST("/api/v1/uploads/images/:id/complete", middleware.RequireAuth(forum), forumHandler.CompleteImageUpload)

	imageBytes := testPNG(t)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/presign", bytes.NewBufferString(`{"fileName":"choice.png","contentType":"image/png","sizeBytes":10,"width":2,"height":2}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated presign status = %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/presign", bytes.NewBufferString(`{"fileName":"choice.exe","contentType":"application/octet-stream","sizeBytes":10,"width":2,"height":2}`))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_upload") {
		t.Fatalf("invalid presign status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	assertRowCount(t, db, "SELECT COUNT(*) FROM upload_assets", 0)

	presignBody := bytes.NewBufferString(`{"fileName":"choice.png","contentType":"image/png","sizeBytes":` + stringInt(len(imageBytes)) + `,"width":2,"height":2}`)
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/presign", presignBody)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("presign status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var presign struct {
		Data domain.PresignedImageUpload `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &presign); err != nil {
		t.Fatal(err)
	}
	if presign.Data.AssetKey == "" || strings.Contains(presign.Data.AssetKey, "http") {
		t.Fatalf("unexpected asset key: %+v", presign.Data)
	}
	var storedUserID int64
	var storedAssetKey, storedStatus string
	if err := db.QueryRow(`SELECT user_id, asset_key, status FROM upload_assets WHERE id = ?`, presign.Data.ID).Scan(&storedUserID, &storedAssetKey, &storedStatus); err != nil {
		t.Fatal(err)
	}
	if storedUserID != user.ID || storedAssetKey != presign.Data.AssetKey || storedStatus != "pending" {
		t.Fatalf("unexpected pending upload user=%d asset=%q status=%q", storedUserID, storedAssetKey, storedStatus)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, presign.Data.UploadURL, bytes.NewReader(imageBytes))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "image/jpeg")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_content_type") {
		t.Fatalf("content type mismatch status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/api/v1/uploads/images/missing/object", bytes.NewReader(imageBytes))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "image/png")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing object upload status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, presign.Data.UploadURL, bytes.NewReader(imageBytes))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "image/png")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("put status = %d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/missing/complete", nil)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing complete status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/"+presign.Data.ID+"/complete", nil)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var complete struct {
		Data domain.CompleteImageUploadResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &complete); err != nil {
		t.Fatal(err)
	}
	if complete.Data.URL == "" || strings.HasPrefix(complete.Data.URL, "data:") || complete.Data.AssetKey != presign.Data.AssetKey {
		t.Fatalf("unexpected complete result: %+v", complete.Data)
	}
	var completedStatus string
	var completedAt sql.NullString
	if err := db.QueryRow(`SELECT status, completed_at FROM upload_assets WHERE id = ?`, presign.Data.ID).Scan(&completedStatus, &completedAt); err != nil {
		t.Fatal(err)
	}
	if completedStatus != "completed" || !completedAt.Valid || completedAt.String == "" {
		t.Fatalf("upload was not completed in database: status=%q completedAt=%+v", completedStatus, completedAt)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/"+presign.Data.ID+"/complete", nil)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("idempotent complete status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var repeated struct {
		Data domain.CompleteImageUploadResult `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &repeated); err != nil {
		t.Fatal(err)
	}
	if repeated.Data != complete.Data {
		t.Fatalf("idempotent complete result changed: first=%+v repeated=%+v", complete.Data, repeated.Data)
	}

	objectPath := filepath.Join(uploadDir, filepath.FromSlash(presign.Data.AssetKey))
	if err := os.WriteFile(objectPath, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/"+presign.Data.ID+"/complete", nil)
	request.Header.Set("Authorization", "Bearer "+session.Token)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_upload_object") {
		t.Fatalf("complete must revalidate completed object: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	s3Store, err := storage.NewS3ObjectStore(context.Background(), config.Config{
		S3Endpoint: "https://objects.example.test", S3Bucket: "beta-images", S3Region: "cn-south-1", S3ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	forumHandler.objectStore = s3Store
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/uploads/images/presign", bytes.NewBufferString(`{"fileName":"s3.png","contentType":"image/png","sizeBytes":`+stringInt(len(imageBytes))+`,"width":2,"height":2}`))
	request.Header.Set("Authorization", "Bearer "+session.Token)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), "objects.example.test") {
		t.Fatalf("S3 presign status = %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func stringInt(value int) string {
	return strconv.Itoa(value)
}
