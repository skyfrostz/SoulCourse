package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

type delegationRepositoryStub struct {
	ForumRepository
	err       error
	name      string
	author    string
	userID    int64
	postID    int64
	follower  int64
	profile   domain.UpdateProfileInput
	created   domain.RegisterInput
	codeEmail string
	sessionID int64
}

func (r *delegationRepositoryStub) GetPost(context.Context, *int64, int64) (domain.Post, []domain.Comment, error) {
	return domain.Post{ID: 41}, []domain.Comment{{ID: 42}}, r.err
}
func (r *delegationRepositoryStub) ListInsights(context.Context) ([]domain.SubjectInsight, error) {
	return []domain.SubjectInsight{{ID: 1}}, r.err
}
func (r *delegationRepositoryStub) GetInsight(context.Context, int64) (domain.SubjectInsight, error) {
	return domain.SubjectInsight{ID: 2}, r.err
}
func (r *delegationRepositoryStub) ListTopics(context.Context) ([]domain.Topic, error) {
	return []domain.Topic{{ID: 3}}, r.err
}
func (r *delegationRepositoryStub) GetTopic(context.Context, *int64, string) (domain.TopicDetail, error) {
	return domain.TopicDetail{}, r.err
}
func (r *delegationRepositoryStub) TogglePostLike(_ context.Context, userID, postID int64) (domain.ToggleResult, error) {
	r.userID, r.postID = userID, postID
	return domain.ToggleResult{Active: true, Count: 9}, r.err
}
func (r *delegationRepositoryStub) TogglePostFavorite(_ context.Context, userID, postID int64) (domain.ToggleResult, error) {
	r.userID, r.postID = userID, postID
	return domain.ToggleResult{Active: true}, r.err
}
func (r *delegationRepositoryStub) ToggleFollowAuthor(_ context.Context, followerID int64, author string) (bool, error) {
	r.follower, r.author = followerID, author
	return true, r.err
}
func (r *delegationRepositoryStub) GetAccountProfile(_ context.Context, _ *int64, name string) (domain.AccountProfile, error) {
	r.name = name
	return domain.AccountProfile{}, r.err
}
func (r *delegationRepositoryStub) GetAccountProfileByUserID(_ context.Context, _ *int64, userID int64) (domain.AccountProfile, error) {
	r.userID = userID
	return domain.AccountProfile{}, r.err
}
func (r *delegationRepositoryStub) UpdateAccountProfile(_ context.Context, userID int64, input domain.UpdateProfileInput) (domain.AccountProfile, error) {
	r.userID, r.profile = userID, input
	return domain.AccountProfile{}, r.err
}
func (r *delegationRepositoryStub) ConsumeEmailVerificationCode(_ context.Context, email, _ string, _ int) error {
	r.codeEmail = email
	return r.err
}
func (r *delegationRepositoryStub) CreateUser(_ context.Context, input domain.RegisterInput, _ string) (domain.User, error) {
	r.created = input
	return domain.User{ID: 77, Email: input.Email}, r.err
}
func (r *delegationRepositoryStub) CreateAuthSession(_ context.Context, userID int64, _ string, _ time.Time) error {
	r.userID = userID
	return r.err
}

func TestReadAndInteractionMethodsPreserveRepositoryBehavior(t *testing.T) {
	ctx := context.Background()
	repo := &delegationRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)

	post, comments, err := forum.GetPost(ctx, nil, 41)
	if err != nil || post.ID != 41 || len(comments) != 1 || comments[0].ID != 42 {
		t.Fatalf("post=%+v comments=%+v err=%v", post, comments, err)
	}
	insights, err := forum.ListInsights(ctx)
	if err != nil || len(insights) != 1 || insights[0].ID != 1 {
		t.Fatalf("insights=%+v err=%v", insights, err)
	}
	if insight, err := forum.GetInsight(ctx, 2); err != nil || insight.ID != 2 {
		t.Fatalf("insight=%+v err=%v", insight, err)
	}
	if topics, err := forum.ListTopics(ctx); err != nil || len(topics) != 1 || topics[0].ID != 3 {
		t.Fatalf("topics=%+v err=%v", topics, err)
	}
	if _, err := forum.GetTopic(ctx, nil, "physics"); err != nil {
		t.Fatal(err)
	}
	if result, err := forum.TogglePostLike(ctx, 7, 8); err != nil || !result.Active || result.Count != 9 || repo.userID != 7 || repo.postID != 8 {
		t.Fatalf("like=%+v user=%d post=%d err=%v", result, repo.userID, repo.postID, err)
	}
	if result, err := forum.TogglePostFavorite(ctx, 9, 10); err != nil || !result.Active || repo.userID != 9 || repo.postID != 10 {
		t.Fatalf("favorite=%+v user=%d post=%d err=%v", result, repo.userID, repo.postID, err)
	}
	if active, err := forum.ToggleFollowAuthor(ctx, 11, " author "); err != nil || !active || repo.follower != 11 || repo.author != " author " {
		t.Fatalf("follow=%v follower=%d author=%q err=%v", active, repo.follower, repo.author, err)
	}

	repo.err = errors.New("repository unavailable")
	for name, call := range map[string]func() error{
		"post":      func() error { _, _, err := forum.GetPost(ctx, nil, 1); return err },
		"insights":  func() error { _, err := forum.ListInsights(ctx); return err },
		"insight":   func() error { _, err := forum.GetInsight(ctx, 1); return err },
		"topics":    func() error { _, err := forum.ListTopics(ctx); return err },
		"topic":     func() error { _, err := forum.GetTopic(ctx, nil, "x"); return err },
		"like":      func() error { _, err := forum.TogglePostLike(ctx, 1, 1); return err },
		"favorite":  func() error { _, err := forum.TogglePostFavorite(ctx, 1, 1); return err },
		"following": func() error { _, err := forum.ToggleFollowAuthor(ctx, 1, "x"); return err },
	} {
		t.Run(name+" error", func(t *testing.T) {
			if err := call(); !errors.Is(err, repo.err) {
				t.Fatalf("error=%v, want repository error", err)
			}
		})
	}
}

func TestAccountProfileValidationNormalizationAndErrors(t *testing.T) {
	ctx := context.Background()
	repo := &delegationRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)
	viewer := int64(4)

	if _, err := forum.GetAccountProfile(ctx, &viewer, "  Alice  "); err != nil || repo.name != "Alice" {
		t.Fatalf("name=%q err=%v", repo.name, err)
	}
	if _, err := forum.GetAccountProfileByUserID(ctx, &viewer, 19); err != nil || repo.userID != 19 {
		t.Fatalf("userID=%d err=%v", repo.userID, err)
	}
	invalid := domain.UpdateProfileInput{}
	if _, err := forum.UpdateAccountProfile(ctx, 19, invalid); !errors.Is(err, ErrInvalidElectives) {
		t.Fatalf("invalid profile error=%v", err)
	}
	valid := domain.UpdateProfileInput{}
	valid.ChoiceProfile.PreferredSubjects = []domain.Subject{"chemistry", "biology"}
	if _, err := forum.UpdateAccountProfile(ctx, 19, valid); err != nil || repo.userID != 19 || len(repo.profile.ChoiceProfile.PreferredSubjects) != 2 {
		t.Fatalf("profile=%+v err=%v", repo.profile, err)
	}
	repo.err = errors.New("profile store failed")
	if _, err := forum.GetAccountProfile(ctx, nil, "Alice"); !errors.Is(err, repo.err) {
		t.Fatalf("get profile error=%v", err)
	}
	if _, err := forum.GetAccountProfileByUserID(ctx, nil, 19); !errors.Is(err, repo.err) {
		t.Fatalf("get profile by ID error=%v", err)
	}
	if _, err := forum.UpdateAccountProfile(ctx, 19, valid); !errors.Is(err, repo.err) {
		t.Fatalf("update profile error=%v", err)
	}
}

func TestRegisterNormalizesIdentityAndPropagatesStageErrors(t *testing.T) {
	ctx := context.Background()
	input := domain.RegisterInput{Email: "  Student@Example.COM ", Password: "correct horse battery staple", VerificationCode: " 123456 "}
	repo := &delegationRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)

	session, err := forum.Register(ctx, input)
	if err != nil || session.User.ID != 77 || session.Token == "" || repo.codeEmail != "student@example.com" || repo.created.Email != "student@example.com" || repo.created.VerificationCode != "123456" || repo.userID != 77 {
		t.Fatalf("session=%+v codeEmail=%q created=%+v userID=%d err=%v", session, repo.codeEmail, repo.created, repo.userID, err)
	}

	repo.err = errors.New("verification store failed")
	if _, err := forum.Register(ctx, input); !errors.Is(err, ErrInvalidEmailVerificationCode) {
		t.Fatalf("verification error=%v", err)
	}

	createErr := errors.New("user create failed")
	createRepo := &registerStageRepositoryStub{createErr: createErr}
	if _, err := NewForumService(createRepo, config.Config{}, nil).Register(ctx, input); !errors.Is(err, createErr) {
		t.Fatalf("create error=%v", err)
	}
	sessionErr := errors.New("session create failed")
	sessionRepo := &registerStageRepositoryStub{sessionErr: sessionErr}
	if _, err := NewForumService(sessionRepo, config.Config{}, nil).Register(ctx, input); !errors.Is(err, sessionErr) {
		t.Fatalf("session error=%v", err)
	}
}

type registerStageRepositoryStub struct {
	ForumRepository
	createErr  error
	sessionErr error
}

func (*registerStageRepositoryStub) ConsumeEmailVerificationCode(context.Context, string, string, int) error {
	return nil
}
func (r *registerStageRepositoryStub) CreateUser(context.Context, domain.RegisterInput, string) (domain.User, error) {
	return domain.User{ID: 88}, r.createErr
}
func (r *registerStageRepositoryStub) CreateAuthSession(context.Context, int64, string, time.Time) error {
	return r.sessionErr
}
