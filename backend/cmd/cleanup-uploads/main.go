package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/repository"
	"subject-choice-forum/backend/internal/storage"
)

func main() {
	var execute bool
	var limit int
	flag.BoolVar(&execute, "execute", false, "delete expired pending upload objects and mark records expired")
	flag.IntVar(&limit, "limit", 100, "maximum expired pending uploads to inspect")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	database, err := storage.NewDatabase(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		os.Exit(1)
	}
	defer database.DB.Close()
	uploadRepository, err := repository.NewUploadCleanupRepository(database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize upload cleanup repository: %v\n", err)
		os.Exit(1)
	}
	objectStore, err := storage.NewObjectStore(ctx, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "initialize object store: %v\n", err)
		os.Exit(1)
	}

	report, err := CleanupExpiredUploads(ctx, uploadRepository, objectStore, time.Now().UTC(), limit, execute)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cleanup expired uploads: %v\n", err)
		os.Exit(1)
	}
	mode := "dry-run"
	if execute {
		mode = "execute"
	}
	fmt.Printf("cleanup expired uploads %s: scanned=%d deleted=%d missing=%d marked=%d\n", mode, report.Scanned, report.Deleted, report.Missing, report.Marked)
}

type UploadCleanupRepository interface {
	ListExpiredPendingImageUploads(ctx context.Context, now time.Time, limit int) ([]domain.ImageUploadRecord, error)
	MarkImageUploadsExpired(ctx context.Context, ids []string, now time.Time) (int64, error)
}

type ObjectDeleter interface {
	Delete(ctx context.Context, key string) error
}

type CleanupReport struct {
	Scanned int
	Deleted int
	Missing int
	Marked  int64
}

func CleanupExpiredUploads(ctx context.Context, repository UploadCleanupRepository, objectStore ObjectDeleter, now time.Time, limit int, execute bool) (CleanupReport, error) {
	records, err := repository.ListExpiredPendingImageUploads(ctx, now, limit)
	if err != nil {
		return CleanupReport{}, err
	}
	report := CleanupReport{Scanned: len(records)}
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
		if !execute {
			continue
		}
		if err := objectStore.Delete(ctx, record.AssetKey); err != nil {
			if errors.Is(err, storage.ErrObjectNotFound) {
				report.Missing++
				continue
			}
			return report, fmt.Errorf("delete %s: %w", record.AssetKey, err)
		}
		report.Deleted++
	}
	if execute {
		report.Marked, err = repository.MarkImageUploadsExpired(ctx, ids, now)
		if err != nil {
			return report, err
		}
	}
	return report, nil
}
