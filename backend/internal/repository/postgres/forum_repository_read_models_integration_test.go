package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/domain"
)

// This test keeps the read-model and aggregation paths exercised against the
// same PostgreSQL schema used by CI. All rows are uniquely named and removed
// explicitly because the integration database is shared by repository tests.
func TestForumRepositoryPostgresReadModelsTaxonomyAndProfile(t *testing.T) {
	db, repo, ctx, suffix := openPostgresRepositoryIntegration(t)
	var userIDs, postIDs []int64
	topicSlug := "pg-read-model-" + suffix
	insightID := int64(0)
	t.Cleanup(func() {
		if insightID != 0 {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM subject_insights WHERE id = $1`, insightID)
		}
		_, _ = db.ExecContext(context.Background(), `DELETE FROM topics WHERE slug = $1`, topicSlug)
		for _, id := range postIDs {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM posts WHERE id = $1`, id)
		}
		for _, id := range userIDs {
			_, _ = db.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, id)
		}
	})
	newUser := func(label string) domain.User {
		u, err := repo.CreateUser(ctx, domain.RegisterInput{
			Email: label + suffix + "@example.test", Nickname: "read-" + label + suffix,
			Role: "student", Province: "广东", Grade: "高一",
		}, "hash")
		if err != nil {
			t.Fatal(err)
		}
		userIDs = append(userIDs, u.ID)
		return u
	}
	owner, viewer := newUser("owner"), newUser("viewer")
	post, err := repo.CreatePost(ctx, owner, domain.CreatePostInput{
		Title: "真实 taxonomy 帖子 " + suffix, Content: "用于验证详情、画像、收藏与计数事务。",
		Tags: []string{"pg-read-tag-" + suffix}, Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category:  domain.CategoryQuestion, Grade: owner.Grade, Province: owner.Province,
	})
	if err != nil {
		t.Fatal(err)
	}
	postIDs = append(postIDs, post.ID)
	if _, err = repo.CreateComment(ctx, viewer, post.ID, domain.CreateCommentInput{Content: "详情评论"}); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.TogglePostFavorite(ctx, viewer.ID, post.ID); err != nil {
		t.Fatal(err)
	}
	if ok, err := repo.ToggleFollowAuthor(ctx, viewer.ID, owner.Nickname); err != nil || !ok {
		t.Fatalf("follow: %v %v", ok, err)
	}

	// Exercise the transaction-backed profile update and every profile list.
	profileInput := domain.UpdateProfileInput{Bio: "  已完成 PostgreSQL 画像  ", ChoiceProfile: domain.ChoiceProfile{
		MBTI: "INTJ", TargetMajors: "临床医学", PreferredTrack: domain.TrackPhysics,
		PreferredSubjects: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
	}}
	profile, err := repo.UpdateAccountProfile(ctx, viewer.ID, profileInput)
	if err != nil || profile.Bio != "已完成 PostgreSQL 画像" || len(profile.Favorites) != 1 || len(profile.Following) != 1 {
		t.Fatalf("updated profile: %+v err=%v", profile, err)
	}
	if len(profile.Followers) != 0 {
		t.Fatalf("unexpected owner followers in viewer profile: %+v", profile.Followers)
	}
	public, err := repo.GetAccountProfile(ctx, &viewer.ID, owner.Nickname)
	if err != nil || public.User.Email != "" || !public.ViewerFollowing || len(public.Comments) != 0 {
		t.Fatalf("public profile: %+v err=%v", public, err)
	}
	if _, err := repo.GetAccountProfileByUserID(ctx, nil, 999999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing profile: %v", err)
	}

	// Detail reads cover viewer state, comments and the not-found boundary.
	detail, comments, err := repo.GetPost(ctx, &viewer.ID, post.ID)
	if err != nil || len(comments) != 1 || detail.ViewerLiked || !detail.ViewerFavorited || detail.FavoritesCount != 1 {
		t.Fatalf("post detail: post=%+v comments=%d err=%v", detail, len(comments), err)
	}
	if _, _, err := repo.GetPost(ctx, nil, 999999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing post: %v", err)
	}

	// Seed only this test's taxonomy rows; no production fixture is required.
	created := time.Now().UTC().Truncate(time.Microsecond)
	if err := db.QueryRowContext(ctx, `INSERT INTO topics (slug, topic_tag, title, summary, views_count, created_at) VALUES ($1,$2,$3,$4,0,$5) RETURNING id`, topicSlug, "pg-read-tag-"+suffix, "PG topic", "topic summary", created).Scan(new(int64)); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO subject_insights (combination, trend, heat, match_rate, advice, details, metric_type, unit, province, data_year, source_name, source_url, scope, sample_size, captured_at, methodology, updated_at) VALUES ($1,'up',88,0.8,'advice','details','rate','%','广东',2026,'test','https://example.test','test',10,$2,'integration test',$3) RETURNING id`, "pg-combination-"+suffix, created, created).Scan(&insightID); err != nil {
		t.Fatal(err)
	}
	insights, err := repo.ListInsights(ctx)
	if err != nil || len(insights) == 0 {
		t.Fatalf("list insights: %d %v", len(insights), err)
	}
	gotInsight, err := repo.GetInsight(ctx, insightID)
	if err != nil || gotInsight.Heat != 88 {
		t.Fatalf("get insight: %+v %v", gotInsight, err)
	}
	if _, err := repo.GetInsight(ctx, 999999999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing insight: %v", err)
	}
	topics, err := repo.ListTopics(ctx)
	if err != nil || len(topics) == 0 {
		t.Fatalf("list topics: %d %v", len(topics), err)
	}
	topic, err := repo.GetTopic(ctx, &viewer.ID, topicSlug)
	if err != nil || len(topic.Posts) != 1 || topic.Topic.PostsCount != 1 {
		t.Fatalf("topic detail: %+v %v", topic, err)
	}
	if _, err := repo.GetTopic(ctx, nil, "missing-"+suffix); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing topic: %v", err)
	}

	// Explicitly hit the empty/invalid boundaries used by callers.
	if _, err := repo.ListPosts(ctx, nil, domain.FeedFilter{Sort: domain.SortLatest, Limit: 0, Keyword: fmt.Sprintf("no-such-%s", suffix)}); err != nil {
		t.Fatal(err)
	}
}

func TestForumRepositoryPostgresStableHelperBoundaries(t *testing.T) {
	post := domain.Post{LikesCount: 3, CommentsCount: 2, FavoritesCount: 1, CreatedAt: time.Unix(10, 0)}
	if got := recommendationScore(post); got <= 0 {
		t.Fatalf("recommendation score = %v", got)
	}
	if got := minInt(2, 5); got != 2 || minInt(5, 2) != 2 || minInt(0, 0) != 0 {
		t.Fatalf("minInt boundary result = %d", got)
	}
	for _, category := range []domain.PostCategory{domain.CategoryQuestion, domain.CategoryExperience, domain.CategoryData} {
		if postContentType(category) == "" {
			t.Fatalf("empty content type for %q", category)
		}
	}
}
