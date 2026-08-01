package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"slices"
	"strconv"
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

func TestFormatUserPublicID(t *testing.T) {
	for _, testCase := range []struct {
		id   int64
		want string
	}{
		{id: 1, want: "00000001"},
		{id: 100000000, want: "100000000"},
	} {
		if got := formatUserPublicID(testCase.id); got != testCase.want {
			t.Fatalf("formatUserPublicID(%d) = %q, want %q", testCase.id, got, testCase.want)
		}
	}
}

func TestCompleteImageUploadIsIdempotentForMatchingMetadata(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	user, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "upload-idempotency@example.com", Nickname: "上传幂等测试", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	record := domain.ImageUploadRecord{
		ID: "idempotent-upload", UserID: user.ID, AssetKey: "images/2026/07/31/idempotent-upload.png",
		FileName: "test.png", ContentType: "image/png", Ext: ".png", SizeBytes: 123, Width: 2, Height: 2,
		Status: "pending", CreatedAt: now, ExpiresAt: now.Add(15 * time.Minute),
	}
	if err := repository.CreateImageUpload(ctx, record); err != nil {
		t.Fatal(err)
	}
	first, err := repository.CompleteImageUpload(ctx, user.ID, record.ID, record.SizeBytes, record.ContentType, record.Width, record.Height, now)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := repository.CompleteImageUpload(ctx, user.ID, record.ID, record.SizeBytes, record.ContentType, record.Width, record.Height, now.Add(time.Second))
	if err != nil {
		t.Fatalf("repeat complete: %v", err)
	}
	if repeated.ID != first.ID || repeated.CompletedAt == nil || first.CompletedAt == nil || !repeated.CompletedAt.Equal(*first.CompletedAt) {
		t.Fatalf("repeat changed completed asset: first=%+v repeated=%+v", first, repeated)
	}
	if _, err := repository.CompleteImageUpload(ctx, user.ID, record.ID, record.SizeBytes+1, record.ContentType, record.Width, record.Height, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("mismatched repeat error = %v, want sql.ErrNoRows", err)
	}
	if _, err := repository.CompleteImageUpload(ctx, user.ID+1, record.ID, record.SizeBytes, record.ContentType, record.Width, record.Height, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("wrong-owner repeat error = %v, want sql.ErrNoRows", err)
	}
}

func TestUserPublicIDsFollowInternalCreationOrderAndKeepSourceIDsSeparate(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	first, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "public-id-first@example.com", Nickname: "编号测试一", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "public-id-second@example.com", Nickname: "编号测试二", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID+1 {
		t.Fatalf("user IDs must follow creation order: first=%d second=%d", first.ID, second.ID)
	}
	if first.PublicID != formatUserPublicID(first.ID) || second.PublicID != formatUserPublicID(second.ID) {
		t.Fatalf("public IDs must derive from internal IDs: first=%+v second=%+v", first, second)
	}

	firstByEmail, _, err := repository.GetUserByEmail(ctx, first.Email)
	if err != nil {
		t.Fatal(err)
	}
	if firstByEmail.PublicID != first.PublicID {
		t.Fatalf("email lookup returned inconsistent public ID: got=%q want=%q", firstByEmail.PublicID, first.PublicID)
	}

	post, err := repository.CreatePost(ctx, first, domain.CreatePostInput{
		Title: "外部来源编号隔离测试", Content: "公开用户编号不能覆盖或混用外部平台的笔记编号。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology}, Category: domain.CategoryData,
	})
	if err != nil {
		t.Fatal(err)
	}
	const sourceNoteID = "xhs_external_note_65f0abc123"
	if _, err := repository.db.ExecContext(ctx, `
		INSERT INTO content_sources (post_id, source_platform, source_url, source_note_id, captured_at)
		VALUES (?, 'xiaohongshu', 'https://www.xiaohongshu.com/explore/65f0abc123', ?, ?)
	`, post.ID, sourceNoteID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	userByID, err := repository.GetUserByID(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	var storedSourceNoteID string
	if err := repository.db.QueryRowContext(ctx, `SELECT source_note_id FROM content_sources WHERE post_id = ?`, post.ID).Scan(&storedSourceNoteID); err != nil {
		t.Fatal(err)
	}
	if storedSourceNoteID != sourceNoteID || userByID.PublicID == sourceNoteID {
		t.Fatalf("external source ID and internal public ID were mixed: source=%q public=%q", storedSourceNoteID, userByID.PublicID)
	}
}

func TestAuthSessionLifecycle(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	user, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "session@example.com", Nickname: "会话测试", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	const tokenHash = "session-token-hash"
	now := time.Now().UTC()
	if err := repository.CreateAuthSession(ctx, user.ID, tokenHash, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	sessions, err := repository.ListAuthSessions(ctx, user.ID, tokenHash, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].Current {
		t.Fatalf("session list did not mark current session: %#v", sessions)
	}
	authed, err := repository.GetUserBySessionTokenHash(ctx, tokenHash, now)
	if err != nil {
		t.Fatal(err)
	}
	if authed.ID != user.ID {
		t.Fatalf("session resolved user ID %d, want %d", authed.ID, user.ID)
	}
	if err := repository.RevokeAuthSession(ctx, tokenHash, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetUserBySessionTokenHash(ctx, tokenHash, now.Add(2*time.Minute)); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("revoked session lookup error = %v, want sql.ErrNoRows", err)
	}
	sessions, err = repository.ListAuthSessions(ctx, user.ID, tokenHash, now.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].RevokedAt == nil || sessions[0].Current {
		t.Fatalf("revoked session list item is wrong: %#v", sessions)
	}
	if err := repository.RevokeAuthSessionByID(ctx, user.ID+999, sessions[0].ID, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-user revoke error = %v, want sql.ErrNoRows", err)
	}
}

func TestShadowUserProfileSupportsNullCredentials(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := repository.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, nickname, role, province, grade, is_shadow, created_at, updated_at)
		VALUES (NULL, NULL, '外部作者主页测试', 'counselor', '广东', '选科用户', 1, ?, ?)
	`, now, now)
	if err != nil {
		t.Fatal(err)
	}
	userID, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.GetUserByID(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if user.Email != "" || user.PublicID != formatUserPublicID(userID) {
		t.Fatalf("unexpected shadow user: %+v", user)
	}
	post, err := repository.CreatePost(ctx, user, domain.CreatePostInput{
		Title: "外部作者的真实公开帖子", Content: "该帖子用于验证迁移作者主页可以按 user ID 正常加载。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology}, Category: domain.CategoryExperience,
	})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := repository.GetAccountProfile(ctx, nil, user.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Posts) != 1 || profile.Posts[0].ID != post.ID || profile.User.ID != userID {
		t.Fatalf("shadow profile did not resolve its post by user ID: %+v", profile)
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
	if len(posts.Items) != 1 || posts.Items[0].Title != "精确标签测试帖子" {
		t.Fatalf("tag filter must match one exact JSON value: %#v", posts)
	}
}

func TestDeletePostSoftDeletesOnlyOwnerPostAndAdminRecord(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "delete-owner@example.com", Nickname: "删帖用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "delete-other@example.com", Nickname: "其他用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	post, err := repository.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "准备删除的帖子", Content: "这是一条用于验证作者软删除能力的帖子内容。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category: domain.CategoryQuestion,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := repository.DeletePost(ctx, other.ID, post.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-owner delete error = %v, want sql.ErrNoRows", err)
	}
	if err := repository.DeletePost(ctx, owner.ID, post.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repository.GetPost(ctx, &owner.ID, post.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted post detail error = %v, want sql.ErrNoRows", err)
	}
	posts, err := repository.ListPosts(ctx, &owner.ID, domain.FeedFilter{Sort: domain.SortLatest, Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range posts.Items {
		if item.ID == post.ID {
			t.Fatalf("deleted post still visible in list: %#v", posts.Items)
		}
	}
	var adminDeletedAt sql.NullString
	var status string
	if err := repository.db.QueryRowContext(ctx, `
		SELECT deleted_at, status FROM admin_content_records WHERE id = ?
	`, "post-user-"+strconv.FormatInt(post.ID, 10)).Scan(&adminDeletedAt, &status); err != nil {
		t.Fatal(err)
	}
	if !adminDeletedAt.Valid || status != "用户已删除" {
		t.Fatalf("admin record was not marked deleted: deleted_at=%v status=%q", adminDeletedAt.Valid, status)
	}
}

func TestUpdatePostOnlyOwnerAndSyncsAdminRecord(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "edit-owner@example.com", Nickname: "改帖用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "edit-other@example.com", Nickname: "不能改帖用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	post, err := repository.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "原始帖子标题", Content: "这是一条用于验证作者编辑能力的帖子内容。",
		Tags: []string{"原始标签"}, Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category:  domain.CategoryQuestion,
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := repository.UpdatePost(ctx, other.ID, post.ID, domain.UpdatePostInput{
		Title: "越权修改标题", Content: "这是一条不应该成功的越权修改内容。",
		Track: domain.TrackHistory, Electives: []domain.Subject{domain.SubjectPolitics, domain.SubjectGeography},
		Category: domain.CategoryExperience,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("non-owner update error = %v, want sql.ErrNoRows", err)
	}

	updated, err := repository.UpdatePost(ctx, owner.ID, post.ID, domain.UpdatePostInput{
		Title: "更新后的帖子标题", Content: "这是一条已经由作者更新后的帖子内容。",
		Tags: []string{"更新标签"}, Track: domain.TrackHistory,
		Electives: []domain.Subject{domain.SubjectPolitics, domain.SubjectGeography},
		Category:  domain.CategoryExperience,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "更新后的帖子标题" || updated.Track != domain.TrackHistory || updated.Category != domain.CategoryExperience {
		t.Fatalf("unexpected updated post: %+v", updated)
	}
	var title string
	var contentType string
	var summary string
	var payloadRaw string
	if err := repository.db.QueryRowContext(ctx, `
		SELECT title, content_type, summary, payload FROM admin_content_records WHERE id = ?
	`, "post-user-"+strconv.FormatInt(post.ID, 10)).Scan(&title, &contentType, &summary, &payloadRaw); err != nil {
		t.Fatal(err)
	}
	if title != "更新后的帖子标题" || contentType != postContentType(domain.CategoryExperience) || summary != "这是一条已经由作者更新后的帖子内容。" {
		t.Fatalf("admin record was not synced: title=%q type=%q summary=%q", title, contentType, summary)
	}
	if !strings.Contains(payloadRaw, "editedByUserId") || !strings.Contains(payloadRaw, "更新后的帖子内容") {
		t.Fatalf("admin payload was not synced: %s", payloadRaw)
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

func TestAccountProfilePostsUseImmutableUserOwnership(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	first, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "first-owner@example.com", Nickname: "同名用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "second-owner@example.com", Nickname: "同名用户", Role: "student", Province: "广东", Grade: "高二",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	firstPost, err := repository.CreatePost(ctx, first, domain.CreatePostInput{
		Title: "第一个账号的真实帖子", Content: "这篇帖子必须始终归属于第一个真实账号。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology}, Category: domain.CategoryExperience,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondPost, err := repository.CreatePost(ctx, second, domain.CreatePostInput{
		Title: "第二个账号的真实帖子", Content: "即使昵称相同，也不能出现在第一个账号主页。",
		Track: domain.TrackHistory, Electives: []domain.Subject{domain.SubjectPolitics, domain.SubjectGeography}, Category: domain.CategoryExperience,
	})
	if err != nil {
		t.Fatal(err)
	}

	publicProfile, err := repository.GetAccountProfile(ctx, nil, "同名用户")
	if err != nil {
		t.Fatal(err)
	}
	assertProfileOwnsPosts(t, publicProfile, first.ID, []int64{firstPost.ID})
	secondProfile, err := repository.GetAccountProfileByUserID(ctx, &second.ID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertProfileOwnsPosts(t, secondProfile, second.ID, []int64{secondPost.ID})

	if _, err := repository.db.ExecContext(ctx, `UPDATE users SET nickname = '改名后的用户' WHERE id = ?`, first.ID); err != nil {
		t.Fatal(err)
	}
	renamedProfile, err := repository.GetAccountProfile(ctx, nil, "改名后的用户")
	if err != nil {
		t.Fatal(err)
	}
	assertProfileOwnsPosts(t, renamedProfile, first.ID, []int64{firstPost.ID})

	if _, err := repository.db.ExecContext(ctx, `UPDATE posts SET deleted_at = ? WHERE id = ?`, nowString(), firstPost.ID); err != nil {
		t.Fatal(err)
	}
	hiddenProfile, err := repository.GetAccountProfileByUserID(ctx, &first.ID, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if hiddenProfile.Stats.Posts != 0 || len(hiddenProfile.Posts) != 0 {
		t.Fatalf("non-public posts must be excluded: stats=%d posts=%#v", hiddenProfile.Stats.Posts, hiddenProfile.Posts)
	}
}

func assertProfileOwnsPosts(t *testing.T, profile domain.AccountProfile, userID int64, postIDs []int64) {
	t.Helper()
	if profile.User.ID != userID || profile.Stats.Posts != len(postIDs) || len(profile.Posts) != len(postIDs) {
		t.Fatalf("unexpected profile ownership: user=%d stats=%d posts=%#v", profile.User.ID, profile.Stats.Posts, profile.Posts)
	}
	for index, post := range profile.Posts {
		if post.UserID == nil || *post.UserID != userID || post.ID != postIDs[index] {
			t.Fatalf("profile contains a post owned by another account: %#v", post)
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
	counselor, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "counselor@example.com", Nickname: "规划师", Role: "counselor", Province: "广东", Grade: "老师",
	}, "counselor-hash")
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
	ownerProfile, err := repository.GetAccountProfile(ctx, &actor.ID, owner.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if !ownerProfile.ViewerFollowing {
		t.Fatal("viewer follow state was not persisted")
	}
	actorProfile, err := repository.GetAccountProfile(ctx, &actor.ID, actor.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if len(actorProfile.Following) != 1 || actorProfile.Following[0].Name != owner.Nickname {
		t.Fatalf("unexpected following list: %#v", actorProfile.Following)
	}
	ownerProfile, err = repository.GetAccountProfile(ctx, &owner.ID, owner.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if len(ownerProfile.Followers) != 1 || ownerProfile.Followers[0].Name != actor.Nickname {
		t.Fatalf("unexpected followers list: %#v", ownerProfile.Followers)
	}

	if _, err := repository.SendDirectMessage(ctx, counselor.ID, owner.Nickname, "先约一次画像梳理。"); err != nil {
		t.Fatal(err)
	}
	sent, err := repository.SendDirectMessage(ctx, actor.ID, owner.Nickname, "想请教一下选科经验。")
	if err != nil {
		t.Fatal(err)
	}
	if sent.SenderName != actor.Nickname || sent.RecipientName != owner.Nickname {
		t.Fatalf("unexpected sent message: %#v", sent)
	}
	conversationPage, err := repository.ListConversations(ctx, owner.ID, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	conversations := conversationPage.Items
	if len(conversations) != 2 || conversations[0].UnreadCount != 1 || conversations[0].User.Nickname != actor.Nickname || conversations[1].User.Nickname != counselor.Nickname {
		t.Fatalf("unexpected unread conversation: %#v", conversations)
	}
	firstConversationPage, err := repository.ListConversations(ctx, owner.ID, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstConversationPage.Items) != 1 || !firstConversationPage.HasMore || firstConversationPage.NextCursor == "" {
		t.Fatalf("first conversation page = %#v, want one item and next cursor", firstConversationPage)
	}
	secondConversationPage, err := repository.ListConversations(ctx, owner.ID, 1, firstConversationPage.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondConversationPage.Items) != 1 || secondConversationPage.HasMore || secondConversationPage.NextCursor != "" {
		t.Fatalf("second conversation page = %#v, want final item", secondConversationPage)
	}
	if secondConversationPage.Items[0].User.ID == firstConversationPage.Items[0].User.ID {
		t.Fatalf("conversation cursor repeated peer %d", secondConversationPage.Items[0].User.ID)
	}
	if _, err := repository.ListConversations(ctx, owner.ID, 1, "not-a-cursor"); err == nil {
		t.Fatal("invalid conversation cursor must fail")
	}
	messagesPage, err := repository.ListDirectMessages(ctx, owner.ID, actor.Nickname, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(messagesPage.Items) != 1 || messagesPage.Items[0].Content != sent.Content || messagesPage.Items[0].ReadAt == nil {
		t.Fatalf("unexpected message thread: %#v", messagesPage)
	}
	if _, err := repository.SendDirectMessage(ctx, actor.ID, owner.Nickname, "第二条消息"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SendDirectMessage(ctx, actor.ID, owner.Nickname, "第三条消息"); err != nil {
		t.Fatal(err)
	}
	firstMessagePage, err := repository.ListDirectMessages(ctx, owner.ID, actor.Nickname, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstMessagePage.Items) != 2 || !firstMessagePage.HasMore || firstMessagePage.NextCursor == "" {
		t.Fatalf("first message page = %#v, want 2 items and next cursor", firstMessagePage)
	}
	secondMessagePage, err := repository.ListDirectMessages(ctx, owner.ID, actor.Nickname, 2, firstMessagePage.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondMessagePage.Items) != 1 || secondMessagePage.Items[0].ID == firstMessagePage.Items[0].ID {
		t.Fatalf("cursor returned unexpected message page: first=%#v second=%#v", firstMessagePage, secondMessagePage)
	}
	conversationPage, err = repository.ListConversations(ctx, owner.ID, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	conversations = conversationPage.Items
	if conversations[0].UnreadCount != 0 {
		t.Fatalf("opening a thread must mark messages read: %#v", conversations[0])
	}

	notificationsPage, err := repository.ListNotifications(ctx, owner.ID, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	types := make(map[string]bool, len(notificationsPage.Items))
	for _, notification := range notificationsPage.Items {
		types[notification.Type] = true
	}
	for _, notificationType := range []string{"profile", "comment", "like", "favorite", "follow", "message"} {
		if !types[notificationType] {
			t.Fatalf("missing %q notification in %#v", notificationType, types)
		}
	}
	firstPage, err := repository.ListNotifications(ctx, owner.ID, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(firstPage.Items) != 2 || !firstPage.HasMore || firstPage.NextCursor == "" {
		t.Fatalf("first notification page = %#v, want 2 items and next cursor", firstPage)
	}
	secondPage, err := repository.ListNotifications(ctx, owner.ID, 2, firstPage.NextCursor)
	if err != nil {
		t.Fatal(err)
	}
	if len(secondPage.Items) == 0 {
		t.Fatal("second notification page is empty")
	}
	if firstPage.Items[0].ID == secondPage.Items[0].ID || firstPage.Items[1].ID == secondPage.Items[0].ID {
		t.Fatalf("cursor returned duplicate notification: first=%#v second=%#v", firstPage.Items, secondPage.Items)
	}

	if err := repository.MarkNotificationRead(ctx, owner.ID, nil); err != nil {
		t.Fatal(err)
	}
	notificationsPage, err = repository.ListNotifications(ctx, owner.ID, 100, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, notification := range notificationsPage.Items {
		if notification.ReadAt == nil {
			t.Fatalf("notification %d was not marked read", notification.ID)
		}
	}
}

func TestLatestPostCursorPaginationHasNoDuplicates(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	user, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "post-cursor@example.com", Nickname: "帖子分页用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 5; index++ {
		if _, err := repository.CreatePost(ctx, user, domain.CreatePostInput{
			Title: "游标分页帖子 " + strconv.Itoa(index), Content: "用于验证最新帖子游标分页不会重复或遗漏内容。",
			Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
			Category: domain.CategoryQuestion, Grade: "高一", Province: "广东",
		}); err != nil {
			t.Fatal(err)
		}
	}

	seen := make(map[int64]bool)
	cursor := ""
	for pageNumber := 0; pageNumber < 3; pageNumber++ {
		page, err := repository.ListPosts(ctx, &user.ID, domain.FeedFilter{Sort: domain.SortLatest, Limit: 2, Cursor: cursor, UserID: &user.ID})
		if err != nil {
			t.Fatal(err)
		}
		for _, post := range page.Items {
			if seen[post.ID] {
				t.Fatalf("post %d appeared on more than one cursor page", post.ID)
			}
			seen[post.ID] = true
		}
		if pageNumber < 2 && !page.HasMore {
			t.Fatalf("page %d unexpectedly reported no more results: %#v", pageNumber, page)
		}
		cursor = page.NextCursor
	}
	if len(seen) != 5 || cursor != "" {
		t.Fatalf("cursor walk returned %d unique posts and final cursor %q, want 5 and empty", len(seen), cursor)
	}
}

func TestListPostsRejectsInvalidLatestCursorBeforeQuery(t *testing.T) {
	repository := NewForumRepository(nil)
	_, err := repository.ListPosts(context.Background(), nil, domain.FeedFilter{
		Sort:   domain.SortLatest,
		Cursor: "not-a-cursor",
		Limit:  10,
	})
	if err == nil || !strings.Contains(err.Error(), "invalid post cursor") {
		t.Fatalf("invalid cursor error = %v", err)
	}
}

func TestCursorParsersRejectMalformedAndNonPositiveIDs(t *testing.T) {
	invalid := []string{"missing-separator", "_1", "2026-07-31T00:00:00Z_", "2026-07-31T00:00:00Z_zero", "2026-07-31T00:00:00Z_0", "2026-07-31T00:00:00Z_-1"}
	for _, cursor := range invalid {
		if _, _, err := parsePostCursor(cursor); err == nil {
			t.Errorf("parsePostCursor(%q) succeeded", cursor)
		}
		if _, _, err := parseDirectMessageCursor(cursor); err == nil {
			t.Errorf("parseDirectMessageCursor(%q) succeeded", cursor)
		}
		if _, _, err := parseConversationCursor(cursor); err == nil {
			t.Errorf("parseConversationCursor(%q) succeeded", cursor)
		}
		if _, _, err := parseNotificationCursor(cursor); err == nil {
			t.Errorf("parseNotificationCursor(%q) succeeded", cursor)
		}
	}
	if createdAt, id, err := parsePostCursor(""); err != nil || createdAt != "" || id != 0 {
		t.Fatalf("empty cursor = (%q, %d, %v), want zero values", createdAt, id, err)
	}
}

func TestPostRelationCountersStayConsistentAndNonNegative(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "counter-owner@example.com", Nickname: "计数作者", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	actor, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "counter-actor@example.com", Nickname: "计数互动者", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	post, err := repository.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "互动计数一致性", Content: "验证点赞收藏评论计数和关系表保持一致。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectGeography},
		Category: domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}

	liked, err := repository.TogglePostLike(ctx, actor.ID, post.ID)
	if err != nil || !liked.Active || liked.Count != 1 {
		t.Fatalf("first like = %#v, %v", liked, err)
	}
	unliked, err := repository.TogglePostLike(ctx, actor.ID, post.ID)
	if err != nil || unliked.Active || unliked.Count != 0 {
		t.Fatalf("second like = %#v, %v", unliked, err)
	}
	favorited, err := repository.TogglePostFavorite(ctx, actor.ID, post.ID)
	if err != nil || !favorited.Active || favorited.Count != 1 {
		t.Fatalf("first favorite = %#v, %v", favorited, err)
	}
	if _, err := repository.CreateComment(ctx, actor, post.ID, domain.CreateCommentInput{Content: "计数测试评论"}); err != nil {
		t.Fatal(err)
	}

	stored, comments, err := repository.GetPost(ctx, &actor.ID, post.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.LikesCount != 0 || stored.FavoritesCount != 1 || stored.CommentsCount != 1 || len(comments) != 1 {
		t.Fatalf("stored counters do not match relations: post=%#v comments=%d", stored, len(comments))
	}
	if _, err := repository.db.ExecContext(ctx, `UPDATE posts SET favorites_count = 0 WHERE id = ?`, post.ID); err != nil {
		t.Fatal(err)
	}
	unfavorited, err := repository.TogglePostFavorite(ctx, actor.ID, post.ID)
	if err != nil || unfavorited.Active || unfavorited.Count != 0 {
		t.Fatalf("counter floor failed: %#v, %v", unfavorited, err)
	}
	if _, err := repository.TogglePostLike(ctx, actor.ID, post.ID+9999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing-post toggle error = %v, want sql.ErrNoRows", err)
	}
}

func TestImageUploadOwnershipExpirationAndCompletionBoundaries(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "upload-owner@example.com", Nickname: "上传所有者", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repository.CreateUser(ctx, domain.RegisterInput{
		Email: "upload-other@example.com", Nickname: "上传其他用户", Role: "student", Province: "广东", Grade: "高一",
	}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	newRecord := func(id string, expiresAt time.Time) domain.ImageUploadRecord {
		return domain.ImageUploadRecord{ID: id, UserID: owner.ID, AssetKey: "images/" + id + ".png", FileName: id + ".png", ContentType: "image/png", Ext: ".png", SizeBytes: 10, Width: 2, Height: 2, Status: "pending", CreatedAt: now.Add(-time.Hour), ExpiresAt: expiresAt}
	}
	expired := newRecord("expired-upload", now.Add(-time.Minute))
	future := newRecord("future-upload", now.Add(time.Minute))
	completed := newRecord("completed-upload", now.Add(-time.Minute))
	for _, record := range []domain.ImageUploadRecord{expired, future, completed} {
		if err := repository.CreateImageUpload(ctx, record); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.CompleteImageUpload(ctx, owner.ID, completed.ID, completed.SizeBytes, completed.ContentType, completed.Width, completed.Height, now); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetImageUpload(ctx, other.ID, expired.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("cross-owner get error = %v, want sql.ErrNoRows", err)
	}
	records, err := repository.ListExpiredPendingImageUploads(ctx, now, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ID != expired.ID {
		t.Fatalf("expired pending uploads = %#v, want only %q", records, expired.ID)
	}
	if affected, err := repository.MarkImageUploadsExpired(ctx, []string{"", expired.ID, completed.ID}, now); err != nil || affected != 1 {
		t.Fatalf("mark expired = (%d, %v), want (1, nil)", affected, err)
	}
	if affected, err := repository.MarkImageUploadsExpired(ctx, []string{"  "}, now); err != nil || affected != 0 {
		t.Fatalf("blank IDs = (%d, %v), want (0, nil)", affected, err)
	}
	if affected, err := repository.MarkImageUploadsExpired(ctx, nil, now); err != nil || affected != 0 {
		t.Fatalf("nil IDs = (%d, %v), want (0, nil)", affected, err)
	}
	if _, err := repository.CompleteImageUpload(ctx, owner.ID, expired.ID, expired.SizeBytes, expired.ContentType, expired.Width, expired.Height, now); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("complete expired upload error = %v, want sql.ErrNoRows", err)
	}
}

func TestMessagesAndNotificationsEnforceUserBoundaries(t *testing.T) {
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	first, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "boundary-first@example.com", Nickname: "边界用户甲", Role: "student", Province: "广东", Grade: "高一"}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	second, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "boundary-second@example.com", Nickname: "边界用户乙", Role: "student", Province: "广东", Grade: "高一"}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	third, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "boundary-third@example.com", Nickname: "边界用户丙", Role: "student", Province: "广东", Grade: "高一"}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.SendDirectMessage(ctx, first.ID, first.Nickname, "不能发给自己"); err == nil {
		t.Fatal("self-message unexpectedly succeeded")
	}
	if _, err := repository.SendDirectMessage(ctx, first.ID, "不存在的用户", "找不到收件人"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing recipient error = %v, want sql.ErrNoRows", err)
	}
	if _, err := repository.SendDirectMessage(ctx, first.ID, second.Nickname, "只属于甲乙的消息"); err != nil {
		t.Fatal(err)
	}
	thirdPage, err := repository.ListDirectMessages(ctx, third.ID, second.Nickname, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(thirdPage.Items) != 0 {
		t.Fatalf("third user saw another conversation: %#v", thirdPage.Items)
	}
	if _, err := repository.ListDirectMessages(ctx, first.ID, second.Nickname, 20, "bad-cursor"); err == nil {
		t.Fatal("invalid direct-message cursor unexpectedly succeeded")
	}

	actorID := second.ID
	if err := repository.createNotification(ctx, first.ID, &actorID, "test", "测试通知", "仅甲可读", "/test"); err != nil {
		t.Fatal(err)
	}
	page, err := repository.ListNotifications(ctx, first.ID, 10, "")
	if err != nil {
		t.Fatalf("notification page = %#v, %v", page, err)
	}
	var notificationID int64
	for _, item := range page.Items {
		if item.Type == "test" {
			notificationID = item.ID
			break
		}
	}
	if notificationID == 0 {
		t.Fatalf("test notification missing from %#v", page.Items)
	}
	if err := repository.MarkNotificationRead(ctx, third.ID, &notificationID); err != nil {
		t.Fatal(err)
	}
	page, err = repository.ListNotifications(ctx, first.ID, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.ID == notificationID && item.ReadAt != nil {
			t.Fatal("another user marked the notification as read")
		}
	}
	if _, err := repository.ListNotifications(ctx, first.ID, 10, "bad-cursor"); err == nil {
		t.Fatal("invalid notification cursor unexpectedly succeeded")
	}
}

func TestAccountModerationTaxonomyAndRelationshipBranches(t *testing.T) {
	t.Parallel()
	repository := newVerificationLimitTestRepository(t)
	ctx := context.Background()
	owner, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "branches-owner@example.com", Nickname: "分支作者", Role: "student", Province: "广东", Grade: "高一"}, "old-hash")
	if err != nil {
		t.Fatal(err)
	}
	actor, err := repository.CreateUser(ctx, domain.RegisterInput{Email: "branches-actor@example.com", Nickname: "分支读者", Role: "student", Province: "广东", Grade: "高二"}, "actor-hash")
	if err != nil {
		t.Fatal(err)
	}

	post, err := repository.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "分支覆盖帖子", Content: "用于验证举报、关注和帖子读取分支。", Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology}, Category: domain.CategoryQuestion,
		Tags: []string{"分支测试"}, Province: "广东", Grade: "高一",
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := repository.ListPosts(ctx, &actor.ID, domain.FeedFilter{Limit: 10, Sort: domain.SortLatest, Tag: "分支测试"})
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != post.ID {
		t.Fatalf("post read = %#v, %v", page, err)
	}

	following, err := repository.ToggleFollowAuthor(ctx, actor.ID, owner.Nickname)
	if err != nil || !following {
		t.Fatalf("first follow = %v, %v", following, err)
	}
	following, err = repository.ToggleFollowAuthor(ctx, actor.ID, owner.Nickname)
	if err != nil || following {
		t.Fatalf("second follow = %v, %v", following, err)
	}
	if _, err := repository.ToggleFollowAuthor(ctx, actor.ID, actor.Nickname); err == nil {
		t.Fatal("self-follow unexpectedly succeeded")
	}
	if _, err := repository.ToggleFollowAuthor(ctx, actor.ID, "不存在的作者"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing author error = %v", err)
	}

	report, err := repository.ReportPost(ctx, actor, post.ID, domain.ReportPostInput{Reason: "不实信息", Detail: "测试举报详情"})
	if err != nil {
		t.Fatal(err)
	}
	if report.TargetID != post.ID || report.ReporterName != actor.Nickname || report.Status != "open" {
		t.Fatalf("report = %#v", report)
	}
	updated, err := repository.ReportPost(ctx, actor, post.ID, domain.ReportPostInput{Reason: "补充原因", Detail: "更新后的详情"})
	if err != nil || updated.ID != report.ID || updated.Reason != "补充原因" || updated.Status != "open" {
		t.Fatalf("upserted report = %#v, %v", updated, err)
	}
	if _, err := repository.ReportPost(ctx, actor, post.ID+999, domain.ReportPostInput{Reason: "不存在"}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing post report error = %v", err)
	}

	insightTime := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = repository.db.ExecContext(ctx, `INSERT INTO subject_insights
		(combination, trend, heat, match_rate, advice, details, metric_type, unit, province, data_year, source_name, source_url, scope, sample_size, captured_at, methodology, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"物理+化学", "上升", 88, 0.76, "建议关注", "样本详情", "choice", "%", "广东", 2026, "测试来源", "https://example.com/source", "广东高中", 1200, insightTime, "抽样统计", insightTime)
	if err != nil {
		t.Fatal(err)
	}
	insights, err := repository.ListInsights(ctx)
	if err != nil || len(insights) == 0 {
		t.Fatalf("insights = %#v, %v", insights, err)
	}
	var testInsight domain.SubjectInsight
	for _, item := range insights {
		if item.SourceURL == "https://example.com/source" {
			testInsight = item
			break
		}
	}
	if testInsight.ID == 0 || testInsight.Heat != 88 || testInsight.SampleSize != 1200 {
		t.Fatalf("inserted insight missing: %#v", insights)
	}
	gotInsight, err := repository.GetInsight(ctx, testInsight.ID)
	if err != nil || gotInsight.SourceURL != "https://example.com/source" {
		t.Fatalf("insight = %#v, %v", gotInsight, err)
	}
	if _, err := repository.GetInsight(ctx, testInsight.ID+999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing insight error = %v", err)
	}

	if hash, err := repository.GetUserPasswordHashByID(ctx, owner.ID); err != nil || hash != "old-hash" {
		t.Fatalf("password hash = %q, %v", hash, err)
	}
	changedID, err := repository.UpdateUserPasswordByEmail(ctx, owner.Email, "new-hash", time.Now())
	if err != nil || changedID != owner.ID {
		t.Fatalf("password update = %d, %v", changedID, err)
	}
	if _, err := repository.UpdateUserPasswordByEmail(ctx, "missing@example.com", "hash", time.Now()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing password update = %v", err)
	}
	if err := repository.DeleteUserAccount(ctx, owner.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.GetUserByID(ctx, owner.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted account lookup = %v", err)
	}
	if _, err := repository.GetUserPasswordHashByID(ctx, owner.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("deleted password lookup = %v", err)
	}
}
