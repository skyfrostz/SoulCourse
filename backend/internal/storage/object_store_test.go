package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"

	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type failingReader struct {
	read bool
}

func (r *failingReader) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		copy(p, "abc")
		return 3, nil
	}
	return 0, errors.New("source read failed")
}

func TestLocalObjectStoreRoundTripAndKeyIsolation(t *testing.T) {
	store, err := NewLocalObjectStore(t.TempDir(), "/app")
	if err != nil {
		t.Fatal(err)
	}
	const key = "images/2026/07/test.png"
	body := strings.NewReader("image-bytes")
	if err := store.Put(context.Background(), key, "image/png", body, int64(len("image-bytes"))); err != nil {
		t.Fatal(err)
	}
	info, err := store.Head(context.Background(), key)
	if err != nil || info.Size != 11 {
		t.Fatalf("head = %+v, err=%v", info, err)
	}
	read, _, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Close()
	if store.PublicURL(key) != "/app/uploads/images/2026/07/test.png" {
		t.Fatalf("unexpected public URL: %s", store.PublicURL(key))
	}
	if _, err := store.Head(context.Background(), "../outside"); err == nil {
		t.Fatal("expected traversal key to fail")
	}
}

func TestLocalObjectStoreErrorClassificationAndSizeValidation(t *testing.T) {
	store, err := NewLocalObjectStore(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{name: "head missing", run: func() error { _, err := store.Head(context.Background(), "missing.png"); return err }},
		{name: "open missing", run: func() error { _, _, err := store.Open(context.Background(), "missing.png"); return err }},
		{name: "delete missing", run: func() error { return store.Delete(context.Background(), "missing.png") }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, ErrObjectNotFound) {
				t.Fatalf("error = %v, want ErrObjectNotFound", err)
			}
		})
	}
	if err := store.Put(context.Background(), "short.png", "image/png", strings.NewReader("abc"), 4); err == nil || !strings.Contains(err.Error(), "object size mismatch") {
		t.Fatalf("short body error = %v", err)
	}
	if err := store.Put(context.Background(), "long.png", "image/png", strings.NewReader("abcde"), 4); err == nil || !strings.Contains(err.Error(), "object size mismatch") {
		t.Fatalf("long body error = %v", err)
	}
}

func TestLocalObjectStoreDeleteRemovesObject(t *testing.T) {
	store, err := NewLocalObjectStore(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "images/delete.png", "image/png", strings.NewReader("abc"), 3); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), "images/delete.png"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Head(context.Background(), "images/delete.png"); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Head after Delete error = %v, want ErrObjectNotFound", err)
	}
}

func TestS3ObjectStoreDeleteUsesConfiguredBucketAndKey(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.EscapedPath()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	store, err := NewS3ObjectStore(context.Background(), config.Config{
		S3Endpoint: server.URL, S3Bucket: "bucket", S3Region: "region", S3ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), "images/a b.png"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/bucket/images/a%20b.png" {
		t.Fatalf("request = %s %s, want DELETE /bucket/images/a%%20b.png", method, path)
	}
}

func TestLocalObjectStoreCheckRejectsReplacedRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "objects")
	store, err := NewLocalObjectStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("Check() error = %v", err)
	}
}

func TestObjectStoreConfigurationErrors(t *testing.T) {
	if _, err := NewObjectStore(context.Background(), config.Config{StorageDriver: "unknown"}); err == nil || !strings.Contains(err.Error(), "unsupported storage driver") {
		t.Fatalf("unsupported driver error = %v", err)
	}
	if _, err := NewS3ObjectStore(context.Background(), config.Config{S3Bucket: "bucket"}); err == nil || !strings.Contains(err.Error(), "S3_BUCKET, S3_REGION and S3_ENDPOINT") {
		t.Fatalf("incomplete S3 error = %v", err)
	}
	if _, err := NewLocalObjectStore("", ""); err == nil || !strings.Contains(err.Error(), "root is required") {
		t.Fatalf("empty local root error = %v", err)
	}
}

func TestS3ObjectStoreClassifiesNotFoundAndPreservesServiceErrors(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "missing") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `<Error><Code>InternalError</Code><Message>failed</Message></Error>`)
	}))
	defer server.Close()
	store, err := NewS3ObjectStore(context.Background(), config.Config{
		S3Endpoint: server.URL, S3Bucket: "bucket", S3Region: "test-region", S3ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range []struct {
		name string
		run  func() error
	}{
		{name: "head", run: func() error { _, err := store.Head(context.Background(), "missing"); return err }},
		{name: "open", run: func() error { _, _, err := store.Open(context.Background(), "missing"); return err }},
	} {
		t.Run(operation.name, func(t *testing.T) {
			if err := operation.run(); !errors.Is(err, ErrObjectNotFound) {
				t.Fatalf("missing object error = %v, want ErrObjectNotFound", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := store.Head(ctx, "failure"); err == nil || errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Head(failure) error = %v, want preserved service error", err)
	}
}

func TestNewObjectStoreSelectsLocalAndLocalPresign(t *testing.T) {
	root := filepath.Join(t.TempDir(), "objects")
	store, err := NewObjectStore(context.Background(), config.Config{MediaUploadDir: root, AppBasePath: "/app"})
	if err != nil {
		t.Fatal(err)
	}
	local, ok := store.(*LocalObjectStore)
	if !ok {
		t.Fatalf("store = %T, want *LocalObjectStore", store)
	}
	if err := local.Check(context.Background()); err != nil {
		t.Fatalf("Check = %v", err)
	}
	url, err := local.PresignPut(context.Background(), "upload-id", "image/png", time.Minute)
	if err != nil || url != "/api/v1/uploads/images/upload-id/object" {
		t.Fatalf("PresignPut = %q, %v", url, err)
	}
}

func TestS3ObjectStoreOperationsAndPublicURL(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("ETag", `"etag"`)
			w.Header().Set("Content-Length", "3")
		case http.MethodPut:
			_, _ = io.Copy(io.Discard, r.Body)
		case http.MethodGet:
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("ETag", `"etag"`)
			w.Header().Set("Content-Length", "3")
			_, _ = io.WriteString(w, "abc")
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	store, err := NewS3ObjectStore(context.Background(), config.Config{
		S3Endpoint: server.URL, S3Bucket: "bucket", S3Region: "region", S3ForcePathStyle: true, S3CDNBaseURL: "https://cdn.example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Check(context.Background()); err != nil {
		t.Fatalf("Check = %v", err)
	}
	if err := store.Put(context.Background(), "images/test.png", "image/png", strings.NewReader("abc"), 3); err != nil {
		t.Fatalf("Put = %v", err)
	}
	info, err := store.Head(context.Background(), "images/test.png")
	if err != nil || info.Size != 3 || info.ContentType != "image/png" || info.ETag != `"etag"` {
		t.Fatalf("Head = %+v, %v", info, err)
	}
	body, opened, err := store.Open(context.Background(), "images/test.png")
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil || string(data) != "abc" || opened.Size != 3 {
		t.Fatalf("Open = %q %+v, %v", data, opened, err)
	}
	if got := store.PublicURL("/folder/a b+#.png"); got != "https://cdn.example.com/folder/a%20b+%23.png" {
		t.Fatalf("PublicURL = %q", got)
	}
	store.cdn = ""
	if got := store.PublicURL("image.png"); got != "" {
		t.Fatalf("empty CDN PublicURL = %q", got)
	}
}

func TestS3CheckWrapsServiceFailure(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()
	store, err := NewS3ObjectStore(context.Background(), config.Config{S3Endpoint: server.URL, S3Bucket: "bucket", S3Region: "region", S3ForcePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Check(context.Background()); err == nil || !strings.Contains(err.Error(), "head S3 bucket") {
		t.Fatalf("Check error = %v", err)
	}
}

func TestS3PresignPutCarriesKeyContentTypeAndExpiry(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	store, err := NewS3ObjectStore(context.Background(), config.Config{S3Endpoint: "http://127.0.0.1:9000", S3Bucket: "bucket", S3Region: "region", S3ForcePathStyle: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := store.PresignPut(ctx, "images/a.png", "image/png", time.Minute); err == nil {
		t.Fatal("canceled presign context unexpectedly succeeded")
	}
	url, err := store.PresignPut(context.Background(), "images/a.png", "image/png", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"images", "a.png", "X-Amz-Signature", "X-Amz-Expires"} {
		if !strings.Contains(url, part) {
			t.Fatalf("presigned URL %q missing %q", url, part)
		}
	}
}

func TestLocalObjectStorePutAndOpenBoundaries(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalObjectStore(root, "")
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Put(context.Background(), "", "application/octet-stream", strings.NewReader(""), 0); err == nil {
		t.Fatal("Put accepted an empty key")
	}
	if _, _, err := store.Open(context.Background(), ""); err == nil {
		t.Fatal("Open accepted an empty key")
	}

	blocker := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "blocked/image.png", "image/png", strings.NewReader("abc"), 3); err == nil {
		t.Fatal("Put succeeded through a file used as a directory")
	}
	directoryTarget := filepath.Join(root, "directory-target")
	if err := os.Mkdir(directoryTarget, 0750); err != nil {
		t.Fatal(err)
	}
	if err := store.Put(context.Background(), "directory-target", "image/png", strings.NewReader("abc"), 3); err == nil {
		t.Fatal("Put unexpectedly opened a directory as an object")
	}

	if err := store.Put(context.Background(), "reader-error.bin", "application/octet-stream", &failingReader{}, 6); err == nil || !strings.Contains(err.Error(), "source read failed") {
		t.Fatalf("Put reader error = %v", err)
	}

	directoryKey := "directory-object"
	if err := os.Mkdir(filepath.Join(root, directoryKey), 0750); err != nil {
		t.Fatal(err)
	}
	body, info, err := store.Open(context.Background(), directoryKey)
	if err != nil {
		t.Fatalf("Open directory = %v", err)
	}
	defer body.Close()
	if info.Size < 0 {
		t.Fatalf("directory size = %d, want non-negative metadata size", info.Size)
	}
}

func TestLocalObjectStoreConstructionAndCheckFailures(t *testing.T) {
	parent := t.TempDir()
	blocker := filepath.Join(parent, "root")
	if err := os.WriteFile(blocker, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewLocalObjectStore(filepath.Join(blocker, "objects"), ""); err == nil {
		t.Fatal("NewLocalObjectStore succeeded beneath a file")
	}

	root := filepath.Join(parent, "gone")
	store, err := NewLocalObjectStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	}
	if err := store.Check(context.Background()); err == nil {
		t.Fatal("Check succeeded after root removal")
	}
}

func TestLocalObjectStorePreservesNonNotFoundFilesystemErrors(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalObjectStore(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("loop", filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Head(context.Background(), "loop"); err == nil || errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Head loop error = %v, want preserved filesystem error without panic", err)
	}
	if _, _, err := store.Open(context.Background(), "loop"); err == nil || errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("Open loop error = %v, want preserved filesystem error", err)
	}
}

func TestNewObjectStoreSelectsS3(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	store, err := NewObjectStore(context.Background(), config.Config{
		StorageDriver:    "s3",
		S3Endpoint:       "http://127.0.0.1:9000",
		S3Bucket:         "bucket",
		S3Region:         "region",
		S3ForcePathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := store.(*S3ObjectStore); !ok {
		t.Fatalf("store = %T, want *S3ObjectStore", store)
	}
}

func TestClassifyS3ObjectErrorHTTPBoundary(t *testing.T) {
	if err := classifyS3ObjectError(nil); err != nil {
		t.Fatalf("classify nil = %v", err)
	}
	responseErr := &smithyhttp.ResponseError{
		Response: &smithyhttp.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
		Err:      errors.New("proxy returned 404"),
	}
	if err := classifyS3ObjectError(responseErr); !errors.Is(err, ErrObjectNotFound) {
		t.Fatalf("classify HTTP 404 = %v, want ErrObjectNotFound", err)
	}
}
