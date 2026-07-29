package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
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

func TestPostTagFilterUsesExactJSONValue(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	user, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "tag-filter@example.com", Nickname: "标签测试用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		title string
		tag   string
	}{
		{title: "精确标签测试帖子", tag: "精确标签"},
		{title: "相似标签测试帖子", tag: "精确标签扩展"},
	} {
		if _, err := repository.CreatePost(ctx, user, domain.CreatePostInput{
			Title: item.title, Content: "这是一条用于验证 JSON 标签精确匹配的测试内容。", Tags: []string{item.tag},
			Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
			Category: domain.CategoryQuestion,
		}); err != nil {
			t.Fatal(err)
		}
	}

	posts, err := repository.ListPosts(ctx, nil, domain.FeedFilter{Tag: "精确标签", Sort: domain.SortLatest, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 || posts[0].Title != "精确标签测试帖子" {
		t.Fatalf("tag filter must match one exact JSON value: %#v", posts)
	}
}

func TestTopicsUsePostTagsAfterLegacyMigration(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()

	if _, err := repository.db.ExecContext(ctx, `DELETE FROM topic_posts`); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.ExecContext(ctx, `UPDATE topics SET posts_count = 999 WHERE slug = 'physics-track-how-to-choose'`); err != nil {
		t.Fatal(err)
	}
	topics, err := repository.ListTopics(ctx)
	if err != nil {
		t.Fatal(err)
	}
	physicsTopic := findTopic(topics, "physics-track-how-to-choose")
	if physicsTopic == nil || physicsTopic.TopicTag != domain.TopicTagPhysicsTrack || physicsTopic.PostsCount != 3 {
		t.Fatalf("legacy topic links were not migrated into post tags: %#v", physicsTopic)
	}

	detail, err := repository.GetTopic(ctx, nil, "physics-track-how-to-choose")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Posts) != physicsTopic.PostsCount {
		t.Fatalf("topic detail and dynamic count diverged: count=%d posts=%d", physicsTopic.PostsCount, len(detail.Posts))
	}
	for _, post := range detail.Posts {
		if !slices.Contains(post.Tags, domain.TopicTagPhysicsTrack) {
			t.Fatalf("topic returned a post without its exact tag: %#v", post.Tags)
		}
	}
}

func findTopic(topics []domain.Topic, slug string) *domain.Topic {
	for index := range topics {
		if topics[index].Slug == slug {
			return &topics[index]
		}
	}
	return nil
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

func TestAccountProfileAndNotificationsPersist(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	repository := NewForumRepository(db)
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "owner@example.com", Nickname: "资料用户", Role: "student", Province: "广东", Grade: "高一",
	}, "owner-hash")
	if err != nil {
		t.Fatal(err)
	}
	actor, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "actor@example.com", Nickname: "互动用户", Role: "student", Province: "广东", Grade: "高一",
	}, "actor-hash")
	if err != nil {
		t.Fatal(err)
	}

	updated, err := repository.UpdateAccountProfile(ctx, owner.ID, domain.UpdateProfileInput{
		Bio: "目标计算机专业。",
		ChoiceProfile: domain.ChoiceProfile{
			SchoolType: "普通高中", MBTI: "INTJ", TargetMajors: "计算机科学与技术",
			PreferredTrack:    domain.TrackPhysics,
			PreferredSubjects: []domain.Subject{domain.SubjectChemistry, domain.SubjectGeography},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Bio != "目标计算机专业。" || updated.ChoiceProfile.MBTI != "INTJ" {
		t.Fatalf("profile did not persist: %#v", updated)
	}

	publicProfile, err := repository.GetAccountProfile(ctx, nil, owner.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if publicProfile.User.Email != "" {
		t.Fatal("public profile must not expose email")
	}

	post, err := repository.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "通知链路测试帖子", Content: "用于验证真实互动通知可以正确持久化。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectGeography},
		Category: domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.CreateComment(ctx, actor, post.ID, domain.CreateCommentInput{Content: "这是一条真实评论。"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.TogglePostLike(ctx, actor.ID, post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.TogglePostFavorite(ctx, actor.ID, post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ToggleFollowAuthor(ctx, actor.ID, owner.Nickname); err != nil {
		t.Fatal(err)
	}

	notifications, err := repository.ListNotifications(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]bool, len(notifications))
	for _, notification := range notifications {
		types[notification.Type] = true
	}
	for _, notificationType := range []string{"profile", "comment", "like", "favorite", "follow"} {
		if !types[notificationType] {
			t.Fatalf("missing %q notification in %#v", notificationType, types)
		}
	}

	if err := repository.MarkNotificationRead(ctx, owner.ID, nil); err != nil {
		t.Fatal(err)
	}
	notifications, err = repository.ListNotifications(ctx, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, notification := range notifications {
		if notification.ReadAt == nil {
			t.Fatalf("notification %d was not marked read", notification.ID)
		}
	}
}
