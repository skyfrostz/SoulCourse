package sqlite

import (
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/domain"
)

func TestRecommendedFeedPrioritizesSubjectChoiceContent(t *testing.T) {
	query, _ := buildPostListQuery(domain.FeedFilter{Sort: domain.SortRecommended})
	if !strings.Contains(query, "p.title LIKE '%选科%'") {
		t.Fatal("recommended feed must prioritize subject-choice posts")
	}
}
