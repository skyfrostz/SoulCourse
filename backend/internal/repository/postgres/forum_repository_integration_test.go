package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/domain"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestForumRepositoryPostgresMainJourney(t *testing.T) {
	postgresURL := os.Getenv("POSTGRES_REPOSITORY_TEST_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_REPOSITORY_TEST_URL is not set")
	}
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repository := NewForumRepository(db)
	ctx := context.Background()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	userIDs := make([]int64, 0, 2)
	t.Cleanup(func() {
		for _, userID := range userIDs {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		}
	})

	first, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "pg-first-" + suffix + "@example.com", Nickname: "PG用户甲" + suffix, Role: "student", Province: "广东", Grade: "高一"}, "hash")
	if err != nil {
		t.Fatalf("create first user: %v", err)
	}
	userIDs = append(userIDs, first.ID)
	second, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "pg-second-" + suffix + "@example.com", Nickname: "PG用户乙" + suffix, Role: "student", Province: "广东", Grade: "高二"}, "hash")
	if err != nil {
		t.Fatalf("create second user: %v", err)
	}
	userIDs = append(userIDs, second.ID)
	tokenHash := "token-hash-" + suffix
	if err := repository.CreateAuthSession(ctx, first.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("create auth session: %v", err)
	}
	if _, err := repository.GetUserBySessionTokenHash(ctx, tokenHash, time.Now()); err != nil {
		t.Fatalf("read auth session: %v", err)
	}
	uploadNow := time.Now().UTC()
	upload := domain.ImageUploadRecord{
		ID: "pg-idempotent-upload-" + suffix, UserID: first.ID, AssetKey: "images/2026/07/31/pg-idempotent-upload-" + suffix + ".png",
		FileName: "test.png", ContentType: "image/png", Ext: ".png", SizeBytes: 123, Width: 2, Height: 2,
		Status: "pending", CreatedAt: uploadNow, ExpiresAt: uploadNow.Add(15 * time.Minute),
	}
	if err := repository.CreateImageUpload(ctx, upload); err != nil {
		t.Fatalf("create image upload: %v", err)
	}
	completed, err := repository.CompleteImageUpload(ctx, first.ID, upload.ID, upload.SizeBytes, upload.ContentType, upload.Width, upload.Height, uploadNow)
	if err != nil {
		t.Fatalf("complete image upload: %v", err)
	}
	repeated, err := repository.CompleteImageUpload(ctx, first.ID, upload.ID, upload.SizeBytes, upload.ContentType, upload.Width, upload.Height, uploadNow.Add(time.Second))
	if err != nil {
		t.Fatalf("repeat image upload completion: %v", err)
	}
	if completed.CompletedAt == nil || repeated.CompletedAt == nil || !repeated.CompletedAt.Equal(*completed.CompletedAt) {
		t.Fatalf("repeat image upload changed completion: first=%+v repeated=%+v", completed, repeated)
	}

	post, err := repository.CreatePost(ctx, first, domain.CreatePostInput{
		Title:   "PostgreSQL 公测主链路",
		Content: "验证 PostgreSQL repository 的发布、互动与消息主链路。",
		Track:   "physics", Electives: []domain.Subject{"chemistry", "biology"}, Category: "question",
		Grade: first.Grade, Province: first.Province,
	})
	if err != nil {
		t.Fatalf("create post: %v", err)
	}
	if _, err := repository.CreateComment(ctx, second, post.ID, domain.CreateCommentInput{Content: "这是一条 PostgreSQL 评论。"}); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if _, err := repository.TogglePostLike(ctx, second.ID, post.ID); err != nil {
		t.Fatalf("toggle like: %v", err)
	}
	if _, err := repository.TogglePostFavorite(ctx, second.ID, post.ID); err != nil {
		t.Fatalf("toggle favorite: %v", err)
	}
	if _, err := repository.ReportPost(ctx, second, post.ID, domain.ReportPostInput{Reason: "测试举报"}); err != nil {
		t.Fatalf("report post: %v", err)
	}
	if _, err := repository.SendDirectMessage(ctx, first.ID, second.Nickname, "PostgreSQL 私信测试"); err != nil {
		t.Fatalf("send message: %v", err)
	}
	if page, err := repository.ListNotifications(ctx, second.ID, 20, ""); err != nil || len(page.Items) == 0 || page.Items[0].Type != "message" {
		t.Fatalf("message notification: items=%#v err=%v", page.Items, err)
	}
	if page, err := repository.ListPosts(ctx, &second.ID, domain.FeedFilter{Sort: domain.SortLatest, Limit: 10}); err != nil || len(page.Items) == 0 {
		t.Fatalf("list posts: items=%d err=%v", len(page.Items), err)
	}
	if page, err := repository.ListPosts(ctx, &second.ID, domain.FeedFilter{
		Sort: domain.SortLatest, Limit: 10, Keyword: "PostgreSQL", Subjects: []domain.Subject{"chemistry"},
	}); err != nil || len(page.Items) == 0 || page.Items[0].ID != post.ID {
		t.Fatalf("search PostgreSQL posts: items=%d err=%v", len(page.Items), err)
	}
	if page, err := repository.ListConversations(ctx, second.ID, 10, ""); err != nil || len(page.Items) == 0 {
		t.Fatalf("list conversations: items=%d err=%v", len(page.Items), err)
	}
	if page, err := repository.ListNotifications(ctx, first.ID, 20, ""); err != nil || len(page.Items) == 0 {
		t.Fatalf("list notifications: items=%d err=%v", len(page.Items), err)
	}
}
