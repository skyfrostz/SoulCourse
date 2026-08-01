package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/storage"
)

func TestCleanupExpiredUploadsDryRunAndExecute(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	uploadDir := filepath.Join(tempDir, "uploads")
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "soulcourse.db"),
		MediaUploadDir: uploadDir,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	repository := sqlite.NewForumRepository(db)
	objectStore, err := storage.NewLocalObjectStore(uploadDir, "")
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "cleanup@example.com", Nickname: "清理用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	expired := domain.ImageUploadRecord{
		ID: "expired-upload", UserID: user.ID, AssetKey: "images/2026/07/31/expired-upload.png", FileName: "expired.png",
		ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 1, Height: 1, Status: "pending",
		CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}
	missing := domain.ImageUploadRecord{
		ID: "missing-upload", UserID: user.ID, AssetKey: "images/2026/07/31/missing-upload.png", FileName: "missing.png",
		ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 1, Height: 1, Status: "pending",
		CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}
	future := domain.ImageUploadRecord{
		ID: "future-upload", UserID: user.ID, AssetKey: "images/2026/07/31/future-upload.png", FileName: "future.png",
		ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 1, Height: 1, Status: "pending",
		CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(time.Hour),
	}
	completed := domain.ImageUploadRecord{
		ID: "completed-upload", UserID: user.ID, AssetKey: "images/2026/07/31/completed-upload.png", FileName: "completed.png",
		ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 1, Height: 1, Status: "completed",
		CreatedAt: now.Add(-2 * time.Hour), ExpiresAt: now.Add(-time.Hour),
	}
	for _, record := range []domain.ImageUploadRecord{expired, missing, future, completed} {
		if err := repository.CreateImageUpload(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	expiredPath := filepath.Join(uploadDir, expired.AssetKey)
	if err := os.MkdirAll(filepath.Dir(expiredPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(expiredPath, []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}

	report, err := CleanupExpiredUploads(ctx, repository, objectStore, now, 100, false)
	if err != nil {
		t.Fatal(err)
	}
	if report.Scanned != 2 || report.Deleted != 0 || report.Marked != 0 {
		t.Fatalf("dry-run report = %+v, want scanned only", report)
	}
	if _, err := os.Stat(expiredPath); err != nil {
		t.Fatalf("dry-run removed file: %v", err)
	}
	stillPending, err := repository.GetImageUpload(ctx, user.ID, expired.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillPending.Status != "pending" {
		t.Fatalf("dry-run status = %q, want pending", stillPending.Status)
	}

	report, err = CleanupExpiredUploads(ctx, repository, objectStore, now, 100, true)
	if err != nil {
		t.Fatal(err)
	}
	if report.Scanned != 2 || report.Deleted != 1 || report.Missing != 1 || report.Marked != 2 {
		t.Fatalf("execute report = %+v, want one deleted, one missing and two marked", report)
	}
	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Fatalf("expired file still exists or unexpected stat error: %v", err)
	}
	expiredRecord, err := repository.GetImageUpload(ctx, user.ID, expired.ID)
	if err != nil {
		t.Fatal(err)
	}
	if expiredRecord.Status != "expired" || expiredRecord.CompletedAt == nil {
		t.Fatalf("expired record = %+v, want expired with timestamp", expiredRecord)
	}
	futureRecord, err := repository.GetImageUpload(ctx, user.ID, future.ID)
	if err != nil {
		t.Fatal(err)
	}
	if futureRecord.Status != "pending" {
		t.Fatalf("future status = %q, want pending", futureRecord.Status)
	}
	completedRecord, err := repository.GetImageUpload(ctx, user.ID, completed.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completedRecord.Status != "completed" {
		t.Fatalf("completed status = %q, want completed", completedRecord.Status)
	}
}

type cleanupRepositoryStub struct {
	records []domain.ImageUploadRecord
	marked  []string
}

func (r *cleanupRepositoryStub) ListExpiredPendingImageUploads(context.Context, time.Time, int) ([]domain.ImageUploadRecord, error) {
	return r.records, nil
}

func (r *cleanupRepositoryStub) MarkImageUploadsExpired(_ context.Context, ids []string, _ time.Time) (int64, error) {
	r.marked = append([]string(nil), ids...)
	return int64(len(ids)), nil
}

type failingObjectDeleter struct{ err error }

func (d failingObjectDeleter) Delete(context.Context, string) error { return d.err }

func TestCleanupExpiredUploadsDoesNotMarkRecordsWhenObjectDeletionFails(t *testing.T) {
	t.Parallel()
	repository := &cleanupRepositoryStub{records: []domain.ImageUploadRecord{{ID: "upload-1", AssetKey: "images/upload-1.png"}}}
	wantErr := errors.New("storage unavailable")

	report, err := CleanupExpiredUploads(context.Background(), repository, failingObjectDeleter{err: wantErr}, time.Now(), 100, true)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if report.Scanned != 1 || report.Deleted != 0 || len(repository.marked) != 0 {
		t.Fatalf("report = %+v marked=%v, want no database mutation", report, repository.marked)
	}
}
