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

func TestForumRepositoryPostgresLifecycleAndContracts(t *testing.T) {
	postgresURL := os.Getenv("POSTGRES_REPOSITORY_TEST_URL")
	if postgresURL == "" {
		t.Skip("POSTGRES_REPOSITORY_TEST_URL is not set")
	}
	db, err := sql.Open("pgx", postgresURL)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}

	repository := NewForumRepository(db)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	users := make([]int64, 0, 4)
	posts := make([]int64, 0, 8)
	t.Cleanup(func() {
		for _, postID := range posts {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM admin_content_records WHERE id = $1`, fmt.Sprintf("post-user-%d", postID))
			_, _ = db.ExecContext(context.Background(), `DELETE FROM posts WHERE id = $1`, postID)
		}
		for _, userID := range users {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
		}
	})

	createUser := func(label string) domain.User {
		t.Helper()
		user, err := repository.CreateUser(ctx, domain.RegisterInput{
			Email: label + "-" + suffix + "@example.test", Nickname: "PG" + label + suffix,
			Role: "student", Province: "广东", Grade: "高一",
		}, "initial-hash")
		if err != nil {
			t.Fatalf("create %s user: %v", label, err)
		}
		users = append(users, user.ID)
		return user
	}
	createPost := func(user domain.User, title string) domain.Post {
		t.Helper()
		post, err := repository.CreatePost(ctx, user, domain.CreatePostInput{
			Title: title, Content: "用于验证 PostgreSQL repository 合同和分页边界的完整测试内容。",
			Tags: []string{"postgres-contract"}, Track: domain.TrackPhysics,
			Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
			Category:  domain.CategoryQuestion, Grade: user.Grade, Province: user.Province,
		})
		if err != nil {
			t.Fatalf("create post %q: %v", title, err)
		}
		posts = append(posts, post.ID)
		return post
	}

	t.Run("account lifecycle ban restore and deletion", func(t *testing.T) {
		user := createUser("account")
		now := time.Now().UTC()
		if err := repository.CreateAuthSession(ctx, user.ID, "session-a-"+suffix, now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := repository.CreateAuthSession(ctx, user.ID, "session-b-"+suffix, now.Add(2*time.Hour)); err != nil {
			t.Fatal(err)
		}
		sessions, err := repository.ListAuthSessions(ctx, user.ID, "session-a-"+suffix, now)
		if err != nil || len(sessions) != 2 {
			t.Fatalf("list sessions: count=%d err=%v", len(sessions), err)
		}
		var currentID int64
		for _, session := range sessions {
			if session.Current {
				currentID = session.ID
			}
		}
		if currentID == 0 {
			t.Fatal("current session was not identified")
		}
		if err := repository.RevokeAuthSessionByID(ctx, user.ID, currentID, now); err != nil {
			t.Fatalf("revoke session: %v", err)
		}
		if _, err := repository.GetUserBySessionTokenHash(ctx, "session-a-"+suffix, now); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("revoked session error = %v, want sql.ErrNoRows", err)
		}

		updatedID, err := repository.UpdateUserPasswordByEmail(ctx, user.Email, "updated-hash", now)
		if err != nil || updatedID != user.ID {
			t.Fatalf("update password: id=%d err=%v", updatedID, err)
		}
		if hash, err := repository.GetUserPasswordHashByID(ctx, user.ID); err != nil || hash != "updated-hash" {
			t.Fatalf("password hash = %q err=%v", hash, err)
		}

		if _, err := db.ExecContext(ctx, `UPDATE users SET banned_at = $1, banned_reason = 'contract test' WHERE id = $2`, now, user.ID); err != nil {
			t.Fatal(err)
		}
		if err := repository.RevokeAuthSessionsForUser(ctx, user.ID, now); err != nil {
			t.Fatal(err)
		}
		if _, _, err := repository.GetUserByEmail(ctx, user.Email); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("banned login lookup error = %v, want sql.ErrNoRows", err)
		}
		if _, err := repository.GetUserBySessionTokenHash(ctx, "session-b-"+suffix, now); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("banned session error = %v, want sql.ErrNoRows", err)
		}

		if _, err := db.ExecContext(ctx, `UPDATE users SET banned_at = NULL, banned_reason = '' WHERE id = $1`, user.ID); err != nil {
			t.Fatal(err)
		}
		if _, hash, err := repository.GetUserByEmail(ctx, user.Email); err != nil || hash != "updated-hash" {
			t.Fatalf("restored login lookup: hash=%q err=%v", hash, err)
		}
		if err := repository.CreateAuthSession(ctx, user.ID, "session-restored-"+suffix, now.Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if _, err := repository.GetUserBySessionTokenHash(ctx, "session-restored-"+suffix, now); err != nil {
			t.Fatalf("restored session lookup: %v", err)
		}

		if err := repository.DeleteUserAccount(ctx, user.ID, now.Add(time.Minute)); err != nil {
			t.Fatalf("delete account: %v", err)
		}
		if _, _, err := repository.GetUserByEmail(ctx, user.Email); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("deleted login lookup error = %v, want sql.ErrNoRows", err)
		}
		var email, password sql.NullString
		var nickname string
		var deletedAt sql.NullTime
		if err := db.QueryRowContext(ctx, `SELECT email, password_hash, nickname, deleted_at FROM users WHERE id = $1`, user.ID).Scan(&email, &password, &nickname, &deletedAt); err != nil {
			t.Fatal(err)
		}
		if email.Valid || password.Valid || !deletedAt.Valid || nickname != fmt.Sprintf("已注销用户-%d", user.ID) {
			t.Fatalf("account was not anonymized: email=%v password=%v nickname=%q deleted=%v", email.Valid, password.Valid, nickname, deletedAt.Valid)
		}
	})

	t.Run("post ownership update deletion and counter consistency", func(t *testing.T) {
		owner := createUser("owner")
		actor := createUser("actor")
		post := createPost(owner, "PostgreSQL 合同更新前")
		update := domain.UpdatePostInput{
			Title: "PostgreSQL 合同更新后", Content: "更新后的正文用于验证帖子与后台内容记录保持同步。",
			Tags: []string{"updated"}, Track: domain.TrackHistory,
			Electives: []domain.Subject{domain.SubjectPolitics, domain.SubjectGeography}, Category: domain.CategoryExperience,
		}
		if _, err := repository.UpdatePost(ctx, actor.ID, post.ID, update); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("non-owner update error = %v, want sql.ErrNoRows", err)
		}
		updated, err := repository.UpdatePost(ctx, owner.ID, post.ID, update)
		if err != nil || updated.Title != update.Title || updated.Content != update.Content || updated.Track != update.Track {
			t.Fatalf("updated post = %+v err=%v", updated, err)
		}

		if result, err := repository.TogglePostLike(ctx, actor.ID, post.ID); err != nil || !result.Active || result.Count != 1 {
			t.Fatalf("activate like: %+v err=%v", result, err)
		}
		if result, err := repository.TogglePostFavorite(ctx, actor.ID, post.ID); err != nil || !result.Active || result.Count != 1 {
			t.Fatalf("activate favorite: %+v err=%v", result, err)
		}
		if _, err := repository.CreateComment(ctx, actor, post.ID, domain.CreateCommentInput{Content: "计数一致性评论"}); err != nil {
			t.Fatal(err)
		}
		stored, comments, err := repository.GetPost(ctx, &actor.ID, post.ID)
		if err != nil || stored.LikesCount != 1 || stored.FavoritesCount != 1 || stored.CommentsCount != 1 || len(comments) != 1 {
			t.Fatalf("post counters: likes=%d favorites=%d comments=%d rows=%d err=%v", stored.LikesCount, stored.FavoritesCount, stored.CommentsCount, len(comments), err)
		}
		var likes, favorites int
		if err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM post_likes WHERE post_id=$1), (SELECT count(*) FROM post_favorites WHERE post_id=$1)`, post.ID).Scan(&likes, &favorites); err != nil {
			t.Fatal(err)
		}
		if likes != stored.LikesCount || favorites != stored.FavoritesCount {
			t.Fatalf("relation counts (%d,%d) differ from post counters (%d,%d)", likes, favorites, stored.LikesCount, stored.FavoritesCount)
		}
		if result, err := repository.TogglePostLike(ctx, actor.ID, post.ID); err != nil || result.Active || result.Count != 0 {
			t.Fatalf("deactivate like: %+v err=%v", result, err)
		}
		if result, err := repository.TogglePostFavorite(ctx, actor.ID, post.ID); err != nil || result.Active || result.Count != 0 {
			t.Fatalf("deactivate favorite: %+v err=%v", result, err)
		}

		if err := repository.DeletePost(ctx, actor.ID, post.ID); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("non-owner delete error = %v, want sql.ErrNoRows", err)
		}
		if err := repository.DeletePost(ctx, owner.ID, post.ID); err != nil {
			t.Fatalf("owner delete: %v", err)
		}
		if _, _, err := repository.GetPost(ctx, nil, post.ID); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("deleted post lookup error = %v, want sql.ErrNoRows", err)
		}
	})

	t.Run("latest cursor pagination has stable boundaries", func(t *testing.T) {
		user := createUser("paging")
		for i := 0; i < 5; i++ {
			createPost(user, fmt.Sprintf("PostgreSQL 游标边界 %d", i))
		}
		cursor := ""
		seen := map[int64]bool{}
		pages := 0
		for {
			page, err := repository.ListPosts(ctx, nil, domain.FeedFilter{UserID: &user.ID, Sort: domain.SortLatest, Limit: 2, Cursor: cursor})
			if err != nil {
				t.Fatalf("page %d: %v", pages+1, err)
			}
			pages++
			if len(page.Items) == 0 {
				t.Fatalf("page %d unexpectedly empty", pages)
			}
			for _, post := range page.Items {
				if seen[post.ID] {
					t.Fatalf("duplicate post %d across cursor pages", post.ID)
				}
				seen[post.ID] = true
			}
			if !page.HasMore {
				if page.NextCursor != "" {
					t.Fatalf("final page returned cursor %q", page.NextCursor)
				}
				break
			}
			if page.NextCursor == "" {
				t.Fatalf("page %d hasMore without next cursor", pages)
			}
			cursor = page.NextCursor
			if pages > 4 {
				t.Fatal("pagination did not terminate")
			}
		}
		if len(seen) != 5 || pages != 3 {
			t.Fatalf("pagination collected %d posts in %d pages, want 5 in 3", len(seen), pages)
		}
	})

	for _, sortOrder := range []domain.FeedSort{domain.SortRecommended, domain.SortHot} {
		t.Run(string(sortOrder)+" cursor pagination has stable rank boundaries", func(t *testing.T) {
			user := createUser("ranked-" + string(sortOrder))
			created := make([]domain.Post, 0, 5)
			for i := 0; i < 5; i++ {
				created = append(created, createPost(user, fmt.Sprintf("PostgreSQL 排名游标 %s %d", sortOrder, i)))
			}
			// Equal ranks and timestamps force id to act as the final deterministic boundary.
			fixedTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
			for _, post := range created {
				if _, err := db.ExecContext(ctx, `UPDATE posts SET likes_count=10, comments_count=2, favorites_count=1, created_at=$1 WHERE id=$2`, fixedTime, post.ID); err != nil {
					t.Fatalf("prepare ranked boundary: %v", err)
				}
			}

			cursor := ""
			seen := map[int64]bool{}
			for pageNumber := 1; ; pageNumber++ {
				page, err := repository.ListPosts(ctx, nil, domain.FeedFilter{UserID: &user.ID, Sort: sortOrder, Limit: 2, Cursor: cursor})
				if err != nil {
					t.Fatalf("page %d: %v", pageNumber, err)
				}
				for _, post := range page.Items {
					if seen[post.ID] {
						t.Fatalf("duplicate post %d across ranked pages", post.ID)
					}
					seen[post.ID] = true
				}
				if !page.HasMore {
					if page.NextCursor != "" {
						t.Fatalf("final page returned cursor %q", page.NextCursor)
					}
					break
				}
				if page.NextCursor == "" {
					t.Fatalf("page %d hasMore without next cursor", pageNumber)
				}
				cursor = page.NextCursor
				if pageNumber > 3 {
					t.Fatal("ranked pagination did not terminate")
				}
			}
			if len(seen) != len(created) {
				t.Fatalf("ranked pagination collected %d posts, want %d", len(seen), len(created))
			}
		})
	}
}
