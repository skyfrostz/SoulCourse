package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/domain"
)

func TestBuildPostListQueryUsesRankedCursorWithoutOffset(t *testing.T) {
	post := domain.Post{
		ID: 42, Title: "广东选科经验", Tags: []string{"选科"}, AuthorRole: "teacher",
		LikesCount: 12, CommentsCount: 3, FavoritesCount: 4,
		CreatedAt: time.Date(2026, time.July, 31, 8, 30, 0, 123, time.UTC),
	}

	for _, sortOrder := range []domain.FeedSort{domain.SortRecommended, domain.SortHot} {
		t.Run(string(sortOrder), func(t *testing.T) {
			cursor := postPageCursor(sortOrder, post)
			query, args := buildPostListQuery(domain.FeedFilter{
				Sort: sortOrder, Limit: 20, Offset: 5000, Cursor: cursor,
			})

			if strings.Contains(query, " OFFSET ") {
				t.Fatalf("ranked cursor query contains OFFSET: %s", query)
			}
			if !strings.Contains(query, "p.created_at < ?") || !strings.Contains(query, "p.id < ?") {
				t.Fatalf("ranked cursor query lacks stable time/id boundary: %s", query)
			}
			if len(args) != 6 { // rank twice, time twice, id, limit+1
				t.Fatalf("ranked cursor args = %#v, want six boundary/limit args", args)
			}
			if args[len(args)-1] != 21 {
				t.Fatalf("query limit = %#v, want 21", args[len(args)-1])
			}
		})
	}
}

func TestBuildPostListQueryKeepsLegacyInitialOffset(t *testing.T) {
	query, args := buildPostListQuery(domain.FeedFilter{Sort: domain.SortHot, Limit: 20, Offset: 40})
	if !strings.Contains(query, " OFFSET ?") {
		t.Fatalf("legacy initial offset missing from query: %s", query)
	}
	if len(args) != 2 || args[0] != 21 || args[1] != 40 {
		t.Fatalf("legacy offset args = %#v, want [21 40]", args)
	}

	query, _ = buildPostListQuery(domain.FeedFilter{Sort: domain.SortHot, Limit: 20})
	if strings.Contains(query, " OFFSET ") {
		t.Fatalf("zero-offset first page should not emit OFFSET: %s", query)
	}
}

func TestRankedPostCursorRoundTripAndSortBinding(t *testing.T) {
	post := domain.Post{
		ID: 7, Title: "普通帖子", AuthorRole: "student", LikesCount: 10,
		CommentsCount: 2, FavoritesCount: 1,
		CreatedAt: time.Date(2026, time.July, 31, 9, 0, 0, 0, time.FixedZone("CST", 8*60*60)),
	}
	cursorValue := postPageCursor(domain.SortRecommended, post)
	cursor, err := parseRankedPostCursor(cursorValue, domain.SortRecommended)
	if err != nil {
		t.Fatalf("parse ranked cursor: %v", err)
	}
	if cursor.ID != post.ID || cursor.Rank != postRank(domain.SortRecommended, post) {
		t.Fatalf("cursor = %+v, post = %+v", cursor, post)
	}
	if _, err := parseRankedPostCursor(cursorValue, domain.SortHot); err == nil {
		t.Fatal("recommended cursor must not be accepted for hot feed")
	}
	if _, err := parseRankedPostCursor("not-a-cursor", domain.SortRecommended); err == nil {
		t.Fatal("malformed ranked cursor was accepted")
	}
}

func TestLatestPostCursorRemainsBackwardCompatible(t *testing.T) {
	createdAt := time.Date(2026, time.July, 31, 10, 0, 0, 456, time.UTC)
	value := postPageCursor(domain.SortLatest, domain.Post{ID: 99, CreatedAt: createdAt})
	want := postCursor(createdAt, 99)
	if value != want || strings.HasPrefix(value, "r1.") {
		t.Fatalf("latest cursor = %q, want legacy %q", value, want)
	}
}

func TestNormalizePostSortKeepsRecommendedFallback(t *testing.T) {
	for _, value := range []domain.FeedSort{"", "unknown"} {
		if got := normalizePostSort(value); got != domain.SortRecommended {
			t.Fatalf("normalizePostSort(%q) = %q, want recommended", value, got)
		}
	}
}

func TestListPostsRejectsInvalidCursorBeforeQuery(t *testing.T) {
	repository := NewForumRepository(nil)
	for _, sortOrder := range []domain.FeedSort{domain.SortLatest, domain.SortHot, domain.SortRecommended} {
		if _, err := repository.ListPosts(context.Background(), nil, domain.FeedFilter{Sort: sortOrder, Cursor: "not-a-cursor", Limit: 10}); err == nil || !strings.Contains(err.Error(), "invalid post cursor") {
			t.Errorf("sort=%s invalid cursor error = %v", sortOrder, err)
		}
	}
}
