package repository

import (
	"context"
	"errors"
	"time"

	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/repository/postgres"
	"subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/service"
	"subject-choice-forum/backend/internal/storage"
)

type UploadCleanupRepository interface {
	ListExpiredPendingImageUploads(ctx context.Context, now time.Time, limit int) ([]domain.ImageUploadRecord, error)
	MarkImageUploadsExpired(ctx context.Context, ids []string, now time.Time) (int64, error)
}

func NewForumRepository(database *storage.Database) (service.ForumRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errors.New("database is required")
	}
	switch database.Driver {
	case "sqlite":
		return sqlite.NewForumRepository(database.DB), nil
	case "postgres":
		return postgres.NewForumRepository(database.DB), nil
	default:
		return nil, errors.New("unsupported database driver: " + database.Driver)
	}
}

func NewUploadCleanupRepository(database *storage.Database) (UploadCleanupRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errors.New("database is required")
	}
	switch database.Driver {
	case "sqlite":
		return sqlite.NewForumRepository(database.DB), nil
	case "postgres":
		return postgres.NewForumRepository(database.DB), nil
	default:
		return nil, errors.New("unsupported database driver: " + database.Driver)
	}
}
