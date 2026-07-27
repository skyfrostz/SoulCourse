package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	"subject-choice-forum/backend/internal/storage"
)

func TestRecommendedFeedPrioritizesSubjectChoiceContent(t *testing.T) {
	query, _ := buildPostListQuery(domain.FeedFilter{Sort: domain.SortRecommended})
	if !strings.Contains(query, "p.title LIKE '%选科%'") {
		t.Fatal("recommended feed must prioritize subject-choice posts")
	}
}

func TestReserveEmailVerificationAttemptEnforcesCooldownAndEmailLimit(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	start := time.Date(2026, time.July, 28, 1, 0, 0, 0, time.UTC)

	first, err := repository.ReserveEmailVerificationAttempt(ctx, "student@example.com", "192.0.2.10", start, time.Minute, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Allowed || first.EmailHourlyRemaining != 1 || first.RetryAfterSeconds != 60 {
		t.Fatalf("unexpected first attempt result: %+v", first)
	}

	cooldown, err := repository.ReserveEmailVerificationAttempt(ctx, "student@example.com", "192.0.2.10", start.Add(30*time.Second), time.Minute, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if cooldown.Allowed || cooldown.Scope != "cooldown" || cooldown.RetryAfterSeconds != 30 {
		t.Fatalf("unexpected cooldown result: %+v", cooldown)
	}

	second, err := repository.ReserveEmailVerificationAttempt(ctx, "student@example.com", "192.0.2.10", start.Add(61*time.Second), time.Minute, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Allowed || second.EmailHourlyRemaining != 0 {
		t.Fatalf("unexpected second attempt result: %+v", second)
	}

	hourly, err := repository.ReserveEmailVerificationAttempt(ctx, "student@example.com", "192.0.2.10", start.Add(122*time.Second), time.Minute, 2, 20)
	if err != nil {
		t.Fatal(err)
	}
	if hourly.Allowed || hourly.Scope != "email_hourly" || hourly.RetryAfterSeconds != 3478 {
		t.Fatalf("unexpected hourly result: %+v", hourly)
	}
}

func TestReserveEmailVerificationAttemptEnforcesIPLimitAcrossEmails(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	start := time.Date(2026, time.July, 28, 2, 0, 0, 0, time.UTC)

	for index, email := range []string{"first@example.com", "second@example.com"} {
		result, err := repository.ReserveEmailVerificationAttempt(
			ctx,
			email,
			"192.0.2.20",
			start.Add(time.Duration(index)*time.Second),
			time.Minute,
			5,
			2,
		)
		if err != nil {
			t.Fatal(err)
		}
		if !result.Allowed {
			t.Fatalf("attempt %d should be allowed: %+v", index+1, result)
		}
	}

	blocked, err := repository.ReserveEmailVerificationAttempt(
		ctx,
		"third@example.com",
		"192.0.2.20",
		start.Add(2*time.Second),
		time.Minute,
		5,
		2,
	)
	if err != nil {
		t.Fatal(err)
	}
	if blocked.Allowed || blocked.Scope != "ip_hourly" || blocked.RetryAfterSeconds != 3598 {
		t.Fatalf("unexpected IP limit result: %+v", blocked)
	}
}

func TestConsumeEmailVerificationCodeInvalidatesAfterFailedAttemptLimit(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	const email = "student@example.com"
	if err := repository.CreateEmailVerificationCode(ctx, email, "correct-hash", time.Now().Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	for attempt := 1; attempt <= 4; attempt++ {
		err := repository.ConsumeEmailVerificationCode(ctx, email, "wrong-hash", 5)
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("attempt %d: expected invalid code, got %v", attempt, err)
		}
	}

	var failedAttempts int
	var usedAt sql.NullString
	if err := repository.db.QueryRow(`
		SELECT failed_attempts, used_at
		FROM email_verification_codes
		WHERE email = ?
		ORDER BY id DESC
		LIMIT 1
	`, email).Scan(&failedAttempts, &usedAt); err != nil {
		t.Fatal(err)
	}
	if failedAttempts != 4 || usedAt.Valid {
		t.Fatalf("code should remain active after four failures: attempts=%d used=%v", failedAttempts, usedAt.Valid)
	}

	if err := repository.ConsumeEmailVerificationCode(ctx, email, "wrong-hash", 5); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("fifth attempt: expected invalid code, got %v", err)
	}
	if err := repository.ConsumeEmailVerificationCode(ctx, email, "correct-hash", 5); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("code must be invalid after five failures, got %v", err)
	}

	if err := repository.db.QueryRow(`
		SELECT failed_attempts, used_at
		FROM email_verification_codes
		WHERE email = ?
		ORDER BY id DESC
		LIMIT 1
	`, email).Scan(&failedAttempts, &usedAt); err != nil {
		t.Fatal(err)
	}
	if failedAttempts != 5 || !usedAt.Valid {
		t.Fatalf("code should be invalidated after five failures: attempts=%d used=%v", failedAttempts, usedAt.Valid)
	}
}

func newVerificationLimitTestRepository(t *testing.T) *ForumRepository {
	t.Helper()
	tempDir := t.TempDir()
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "soulcourse.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})
	return NewForumRepository(db)
}
