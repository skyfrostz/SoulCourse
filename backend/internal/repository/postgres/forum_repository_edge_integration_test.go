package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// These tests intentionally exercise the real PostgreSQL implementation. They are skipped
// in ordinary unit-test runs and use unique identities so a shared test database is safe.
func openPostgresRepositoryIntegration(t *testing.T) (*sql.DB, *ForumRepository, context.Context, string) {
	t.Helper()
	url := os.Getenv("POSTGRES_REPOSITORY_TEST_URL")
	if url == "" {
		t.Skip("POSTGRES_REPOSITORY_TEST_URL is not set")
	}
	db, err := sql.Open("pgx", url)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	t.Cleanup(func() { db.Close() })
	return db, NewForumRepository(db), ctx, suffix
}

func TestForumRepositoryPostgresAccountVerificationAndProfile(t *testing.T) {
	db, repo, ctx, suffix := openPostgresRepositoryIntegration(t)
	var userIDs []int64
	var postIDs []int64
	t.Cleanup(func() {
		for _, id := range postIDs {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM posts WHERE id = $1`, id)
		}
		for _, id := range userIDs {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		}
	})
	newUser := func(label string) domain.User {
		u, err := repo.CreateUser(ctx, domain.RegisterInput{Email: label + suffix + "@example.test", Nickname: "edge-" + label + suffix, Role: "student", Province: "广东", Grade: "高一"}, "hash")
		if err != nil {
			t.Fatalf("create user: %v", err)
		}
		userIDs = append(userIDs, u.ID)
		return u
	}
	u := newUser("account")
	other := newUser("other")
	uniqueIP := uint64(time.Now().UnixNano())
	clientIP := fmt.Sprintf("2001:db8:%x:%x::1", (uniqueIP>>16)&0xffff, uniqueIP&0xffff)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM email_verification_attempts WHERE email = $1 OR client_ip = $2`, u.Email, clientIP)
		_, _ = db.ExecContext(context.Background(), `DELETE FROM email_verification_codes WHERE lower(email) = lower($1)`, u.Email)
	})
	post, err := repo.CreatePost(ctx, other, domain.CreatePostInput{
		Title: "可关注的公开帖子", Content: "关注测试帖子", Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry}, Category: domain.CategoryQuestion,
		Grade: other.Grade, Province: other.Province,
	})
	if err != nil {
		t.Fatalf("create follow target post: %v", err)
	}
	postIDs = append(postIDs, post.ID)

	now := time.Now().UTC()
	if limit, err := repo.ReserveEmailVerificationAttempt(ctx, u.Email, clientIP, now, time.Minute, 2, 2); err != nil || !limit.Allowed {
		t.Fatalf("reserve first verification attempt: %+v %v", limit, err)
	}
	if limit, err := repo.ReserveEmailVerificationAttempt(ctx, u.Email, clientIP, now.Add(2*time.Second), time.Minute, 2, 2); err != nil || limit.Allowed || limit.Scope != "cooldown" {
		t.Fatalf("cooldown limit: %+v %v", limit, err)
	}
	if err := repo.CreateEmailVerificationCode(ctx, u.Email, "code-hash", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := repo.ConsumeEmailVerificationCode(ctx, stringsUpper(u.Email), "wrong", 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("wrong code = %v", err)
	}
	if err := repo.ConsumeEmailVerificationCode(ctx, u.Email, "code-hash", 2); err != nil {
		t.Fatalf("consume valid code: %v", err)
	}
	if err := repo.ConsumeEmailVerificationCode(ctx, u.Email, "code-hash", 2); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("reconsume code = %v", err)
	}

	if followed, err := repo.ToggleFollowAuthor(ctx, u.ID, other.Nickname); err != nil || !followed {
		t.Fatalf("follow: %v %v", followed, err)
	}
	profile, err := repo.GetAccountProfileByUserID(ctx, &u.ID, u.ID)
	if err != nil {
		t.Fatalf("own profile: %+v %v", profile, err)
	}
	if len(profile.Following) != 1 || profile.Following[0].Name != other.Nickname {
		t.Fatalf("following = %+v", profile.Following)
	}
	if followed, err := repo.ToggleFollowAuthor(ctx, u.ID, other.Nickname); err != nil || followed {
		t.Fatalf("unfollow: %v %v", followed, err)
	}
	if _, err := repo.GetUserByID(ctx, 999999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing user = %v", err)
	}
}

func TestForumRepositoryPostgresMessageNotificationAndCursorErrors(t *testing.T) {
	db, repo, ctx, suffix := openPostgresRepositoryIntegration(t)
	var ids []int64
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		}
	})
	makeUser := func(label string) domain.User {
		u, err := repo.CreateUser(ctx, domain.RegisterInput{Email: label + suffix + "@example.test", Nickname: "msg-" + label + suffix, Role: "student", Province: "广东", Grade: "高一"}, "hash")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.ID)
		return u
	}
	a, b := makeUser("a"), makeUser("b")
	if _, err := repo.SendDirectMessage(ctx, a.ID, a.Nickname, "self"); err == nil {
		t.Fatal("self message unexpectedly succeeded")
	}
	for i := 0; i < 3; i++ {
		if _, err := repo.SendDirectMessage(ctx, a.ID, b.Nickname, fmt.Sprintf("message-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.ListDirectMessages(ctx, b.ID, a.Nickname, 2, "")
	if err != nil || len(page.Items) != 2 || !page.HasMore || page.NextCursor == "" {
		t.Fatalf("message page: %+v %v", page, err)
	}
	if _, err := repo.ListDirectMessages(ctx, b.ID, a.Nickname, 2, page.NextCursor); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ListDirectMessages(ctx, b.ID, a.Nickname, 2, "bad-cursor"); err == nil {
		t.Fatal("invalid message cursor accepted")
	}
	notifications, err := repo.ListNotifications(ctx, b.ID, 2, "")
	if err != nil || len(notifications.Items) != 2 || !notifications.HasMore {
		t.Fatalf("notification page: %+v %v", notifications, err)
	}
	if err := repo.MarkNotificationRead(ctx, b.ID, &notifications.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkNotificationRead(ctx, b.ID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ListNotifications(ctx, b.ID, 2, "bad-cursor"); err == nil {
		t.Fatal("invalid notification cursor accepted")
	}
	if conversations, err := repo.ListConversations(ctx, b.ID, 10, ""); err != nil || len(conversations.Items) != 1 {
		t.Fatalf("conversations: %+v %v", conversations, err)
	}
	if _, err := repo.ListConversations(ctx, b.ID, 10, "bad-cursor"); err == nil {
		t.Fatal("invalid conversation cursor accepted")
	}
}

func TestForumRepositoryPostgresUploadExpiryOwnershipAndReportErrors(t *testing.T) {
	db, repo, ctx, suffix := openPostgresRepositoryIntegration(t)
	var ids []int64
	t.Cleanup(func() {
		for _, id := range ids {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		}
	})
	makeUser := func(label string) domain.User {
		u, err := repo.CreateUser(ctx, domain.RegisterInput{Email: label + suffix + "@example.test", Nickname: "up-" + label + suffix, Role: "student", Province: "广东", Grade: "高一"}, "hash")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.ID)
		return u
	}
	a, b := makeUser("a"), makeUser("b")
	now := time.Now().UTC()
	record := domain.ImageUploadRecord{ID: "edge-" + suffix, UserID: a.ID, AssetKey: "images/edge-" + suffix + ".png", FileName: "x.png", ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 2, Height: 2, Status: "pending", CreatedAt: now.Add(-time.Hour), ExpiresAt: now.Add(-time.Minute)}
	if err := repo.CreateImageUpload(ctx, record); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetImageUpload(ctx, b.ID, record.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("foreign upload lookup = %v", err)
	}
	expired, err := repo.ListExpiredPendingImageUploads(ctx, now, 10)
	if err != nil || len(expired) == 0 {
		t.Fatalf("expired uploads: %+v %v", expired, err)
	}
	if n, err := repo.MarkImageUploadsExpired(ctx, []string{record.ID}, now); err != nil || n != 1 {
		t.Fatalf("mark expired: %d %v", n, err)
	}
	if _, err := repo.CompleteImageUpload(ctx, a.ID, record.ID, 10, "image/png", 2, 2, now); err == nil {
		t.Fatal("expired upload completed")
	}
	if _, err := repo.ReportPost(ctx, b, 999999999, domain.ReportPostInput{Reason: "missing"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing report target = %v", err)
	}
}

func stringsUpper(value string) string { // Keep the test focused on case-insensitive verification lookup.
	for i := range value {
		if value[i] >= 'a' && value[i] <= 'z' {
			value = value[:i] + string(value[i]-'a'+'A') + value[i+1:]
		}
	}
	return value
}
