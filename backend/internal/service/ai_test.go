package service

import (
	"context"
	"database/sql"
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

func TestAIServiceChoiceAdviceMinimizesSensitiveProfileBeforeProvider(t *testing.T) {
	service := NewAIService(config.Config{AIAPIKey: "test", AIBaseURL: "https://api.example.test/v1", AIModel: "deepseek-v4-flash"})
	var prompt string
	service.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var request chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if len(request.Messages) != 2 {
			t.Fatalf("unexpected message count: %d", len(request.Messages))
		}
		prompt = request.Messages[1].Content
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"建议先核对要求\",\"risks\":[\"政策需核对\"],\"actions\":[\"查看广东要求\"],\"querySuggestions\":[\"广东 选科要求\"]}"}}]}`)),
		}, nil
	})}

	_, err := service.ChoiceAdvice(context.Background(), domain.ChoiceAdviceInput{
		Profile: map[string]any{
			"realName":          "张三",
			"email":             "student@example.com",
			"phone":             "13800138000",
			"school":            "示例一中",
			"preferredTrack":    "physics",
			"preferredSubjects": []any{"chemistry", "biology"},
			"targetMajors":      "临床医学",
		},
		Question: "邮箱 student@example.com，手机号 13800138000，帮我看看。",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"张三", "student@example.com", "13800138000", "示例一中", "realName", "phone", "school"} {
		if strings.Contains(prompt, forbidden) {
			t.Fatalf("provider prompt leaked sensitive value %q: %s", forbidden, prompt)
		}
	}
	for _, expected := range []string{"preferredTrack", "physics", "targetMajors", "临床医学", "[邮箱已移除]", "[手机号已移除]"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("provider prompt missing expected safe value %q: %s", expected, prompt)
		}
	}
}

func TestAIServiceChoiceAdviceReturnsFallbackOnDisabledOrInvalidProvider(t *testing.T) {
	input := domain.ChoiceAdviceInput{Profile: map[string]any{
		"preferredTrack": "physics", "preferredSubjects": []any{"chemistry", "biology"},
	}}

	t.Run("disabled", func(t *testing.T) {
		advice, err := NewAIService(config.Config{}).ChoiceAdvice(context.Background(), input)
		if !errors.Is(err, ErrAIDisabled) || advice.Source != "fallback" || len(advice.Actions) == 0 {
			t.Fatalf("unexpected disabled fallback: advice=%+v err=%v", advice, err)
		}
	})

	for _, tt := range []struct {
		name   string
		status int
		body   string
	}{
		{"provider status", http.StatusTooManyRequests, `{}`},
		{"malformed response", http.StatusOK, `{not-json`},
		{"no choices", http.StatusOK, `{"choices":[]}`},
		{"invalid advice", http.StatusOK, `{"choices":[{"message":{"content":"not-json"}}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ai := NewAIService(config.Config{AIAPIKey: "test", AIBaseURL: "https://api.example.test/v1", AIModel: "model"})
			ai.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tt.status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(tt.body))}, nil
			})}
			advice, err := ai.ChoiceAdvice(context.Background(), input)
			if err == nil || advice.Source != "fallback" || advice.Summary == "" {
				t.Fatalf("expected fallback with provider error: advice=%+v err=%v", advice, err)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCreatePostSkipsAITaggerWhenManualTagsExist(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
	calls := 0
	forum.ConfigurePostTagger(countingPostTagger{calls: &calls}, nil)

	post, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
		Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
		Tags: []string{" 自定义 ", "自定义"}, Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
		Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("expected manual tags to skip AI, got %d calls", calls)
	}
	if !slices.Equal(post.Tags, []string{"自定义", "物化生"}) {
		t.Fatalf("manual and deterministic tags were not preserved: %#v", post.Tags)
	}
}

func TestCreatePostUsesAIOnlyWithoutManualTagsAndDegradesOnFailure(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
	warnings := 0
	forum.ConfigurePostTagger(failingPostTagger{}, func(error) { warnings++ })

	post, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
		Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
		Track:     domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
		Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}
	if warnings != 1 {
		t.Fatalf("expected one observable warning, got %d", warnings)
	}
	if !slices.Equal(post.Tags, []string{"物化生"}) {
		t.Fatalf("deterministic fallback tag was not preserved: %#v", post.Tags)
	}
}

func TestCreatePostRejectsInlineOrExternalImages(t *testing.T) {
	forum := NewForumService(newCreatePostRepositoryStub(), config.Config{JWTSecret: "test"}, nil)

	for _, imageURL := range []string{
		"data:image/png;base64,iVBORw0KGgo=",
		"https://cdn.example.test/uploads/images/post.png",
		"blob:https://example.test/image",
	} {
		_, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
			Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
			ImageURLs: []string{imageURL},
			Track:     domain.TrackPhysics,
			Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
			Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
		})
		if !errors.Is(err, ErrInvalidPostImages) {
			t.Fatalf("expected invalid image error for %q, got %v", imageURL, err)
		}
	}
}

func TestCreatePostAcceptsCompletedLocalUploadImages(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)

	post, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
		Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
		ImageURLs: []string{" /uploads/images/2026/07/31/upload1.png "},
		Track:     domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
		Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(post.ImageURLs, []string{"/uploads/images/2026/07/31/upload1.png"}) {
		t.Fatalf("unexpected image urls: %#v", post.ImageURLs)
	}
}

func TestCreatePostRejectsUncompletedLocalUploadImages(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	repository.uploads["pending1"] = domain.ImageUploadRecord{
		ID:       "pending1",
		UserID:   1,
		AssetKey: "images/2026/07/31/pending1.png",
		Status:   "pending",
	}
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)

	_, err := forum.CreatePost(context.Background(), domain.User{ID: 1, Nickname: "测试用户"}, domain.CreatePostInput{
		Title: "测试帖子标题", Content: "这是一段足够长的测试帖子内容。",
		ImageURLs: []string{"/uploads/images/2026/07/31/pending1.png"},
		Track:     domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectBiology, domain.SubjectChemistry},
		Category:  domain.CategoryQuestion, Grade: "高一", Province: "广东",
	})
	if !errors.Is(err, ErrInvalidPostImages) {
		t.Fatalf("expected invalid image error, got %v", err)
	}
}

func TestDeletePostOnlyDeletesOwnerPost(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)

	if err := forum.DeletePost(context.Background(), 1, 42); err != nil {
		t.Fatal(err)
	}
	if !repository.deleted[42] {
		t.Fatal("expected owner post to be marked deleted")
	}

	err := forum.DeletePost(context.Background(), 2, 42)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected non-owner delete to fail without leaking ownership, got %v", err)
	}
}

func TestUpdatePostValidatesOwnerAndElectives(t *testing.T) {
	repository := newCreatePostRepositoryStub()
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)

	_, err := forum.UpdatePost(context.Background(), 1, 42, domain.UpdatePostInput{
		Title: "更新后的帖子标题", Content: "这是一段足够长的更新后帖子正文。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectChemistry},
		Category: domain.CategoryQuestion,
	})
	if !errors.Is(err, ErrInvalidElectives) {
		t.Fatalf("expected invalid electives, got %v", err)
	}

	post, err := forum.UpdatePost(context.Background(), 1, 42, domain.UpdatePostInput{
		Title: "更新后的帖子标题", Content: "这是一段足够长的更新后帖子正文。",
		Tags: []string{" 自定义 "}, Track: domain.TrackPhysics,
		Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category:  domain.CategoryExperience,
	})
	if err != nil {
		t.Fatal(err)
	}
	if post.Title != "更新后的帖子标题" || !slices.Contains(post.Tags, "物化生") {
		t.Fatalf("unexpected updated post: %+v", post)
	}

	if _, err := forum.UpdatePost(context.Background(), 2, 42, domain.UpdatePostInput{
		Title: "其他用户修改", Content: "这是一段足够长的更新后帖子正文。",
		Track: domain.TrackPhysics, Electives: []domain.Subject{domain.SubjectChemistry, domain.SubjectBiology},
		Category: domain.CategoryQuestion,
	}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected non-owner update to fail, got %v", err)
	}
}

type countingPostTagger struct {
	calls *int
}

func (tagger countingPostTagger) TagPost(context.Context, string, string) ([]string, error) {
	(*tagger.calls)++
	return []string{domain.TopicTagPhysicsTrack}, nil
}

type failingPostTagger struct{}

func (failingPostTagger) TagPost(context.Context, string, string) ([]string, error) {
	return nil, errors.New("provider unavailable")
}

type createPostRepositoryStub struct {
	ForumRepository
	uploads map[string]domain.ImageUploadRecord
	deleted map[int64]bool
}

func newCreatePostRepositoryStub() *createPostRepositoryStub {
	return &createPostRepositoryStub{
		uploads: map[string]domain.ImageUploadRecord{
			"upload1": {
				ID:       "upload1",
				UserID:   1,
				AssetKey: "images/2026/07/31/upload1.png",
				Status:   "completed",
			},
		},
		deleted: map[int64]bool{},
	}
}

func (s *createPostRepositoryStub) CreatePost(_ context.Context, _ domain.User, input domain.CreatePostInput) (domain.Post, error) {
	return domain.Post{Title: input.Title, ImageURLs: input.ImageURLs, Tags: input.Tags}, nil
}

func (s *createPostRepositoryStub) UpdatePost(_ context.Context, userID int64, postID int64, input domain.UpdatePostInput) (domain.Post, error) {
	if userID != 1 || postID != 42 {
		return domain.Post{}, sql.ErrNoRows
	}
	return domain.Post{ID: postID, Title: input.Title, Content: input.Content, Tags: input.Tags}, nil
}

func (s *createPostRepositoryStub) DeletePost(_ context.Context, userID int64, postID int64) error {
	if userID != 1 || postID != 42 || s.deleted[postID] {
		return sql.ErrNoRows
	}
	s.deleted[postID] = true
	return nil
}

func (s *createPostRepositoryStub) GetImageUpload(_ context.Context, userID int64, id string) (domain.ImageUploadRecord, error) {
	record, ok := s.uploads[id]
	if !ok || record.UserID != userID {
		return domain.ImageUploadRecord{}, sql.ErrNoRows
	}
	return record, nil
}
