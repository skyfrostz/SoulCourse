package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

func TestAIServiceTagPostFiltersToControlledTags(t *testing.T) {
	service := NewAIService(config.Config{AIAPIKey: "test", AIBaseURL: "https://api.example.test/v1", AIModel: "deepseek-v4-flash"})
	service.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Model != "deepseek-v4-flash" {
			t.Fatalf("unexpected model: %q", request.Model)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"{\"tags\":[\"物理方向\",\"物化生\",\"自造标签\",\"物理方向\"]}"}}]}`)),
		}, nil
	})}
	tags, err := service.TagPost(context.Background(), "物化生怎么选", "想了解物理方向的学习压力。")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(tags, []string{domain.TopicTagPhysicsTrack, "物化生"}) {
		t.Fatalf("unexpected controlled tags: %#v", tags)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCreatePostMergesTagsAndDegradesWhenAITaggerFails(t *testing.T) {
	repository := &createPostRepositoryStub{}
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
	warnings := 0
	forum.ConfigurePostTagger(failingPostTagger{}, func(error) { warnings++ })

	post, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
		Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
		Tags: []string{" 自定义 ", "自定义"}, Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
		Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}
	if warnings != 1 {
		t.Fatalf("expected one observable warning, got %d", warnings)
	}
	if !slices.Equal(post.Tags, []string{"自定义", "物化生"}) {
		t.Fatalf("manual and deterministic tags were not preserved: %#v", post.Tags)
	}
}

type failingPostTagger struct{}

func (failingPostTagger) TagPost(context.Context, string, string) ([]string, error) {
	return nil, errors.New("provider unavailable")
}

type createPostRepositoryStub struct {
	ForumRepository
}

func (s *createPostRepositoryStub) CreatePost(_ context.Context, _ domain.User, input domain.CreatePostInput) (domain.Post, error) {
	return domain.Post{Title: input.Title, Tags: input.Tags}, nil
}
