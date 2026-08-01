package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
)

type workflowRepositoryStub struct {
	ForumRepository
	mu  sync.Mutex
	err error

	filter             domain.FeedFilter
	comment            domain.CreateCommentInput
	report             domain.ReportPostInput
	upload             domain.ImageUploadRecord
	completed          domain.ImageUploadRecord
	notificationLimit  int
	conversationLimit  int
	messageLimit       int
	peerName           string
	recipientName      string
	messageContent     string
	messageCalls       int
	markedNotification *int64
	tokenUser          domain.User
	revokedTokenHash   string
}

func (r *workflowRepositoryStub) ListPosts(_ context.Context, _ *int64, filter domain.FeedFilter) (domain.PostPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.filter = filter
	return domain.PostPage{Items: []domain.Post{{ID: 1}}}, r.err
}

func (r *workflowRepositoryStub) CreateComment(_ context.Context, _ domain.User, _ int64, input domain.CreateCommentInput) (domain.Comment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.comment = input
	return domain.Comment{ID: 2, Content: input.Content}, r.err
}

func (r *workflowRepositoryStub) ReportPost(_ context.Context, _ domain.User, _ int64, input domain.ReportPostInput) (domain.ContentReport, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.report = input
	return domain.ContentReport{ID: 3, Reason: input.Reason, Detail: input.Detail}, r.err
}

func (r *workflowRepositoryStub) CreateImageUpload(_ context.Context, record domain.ImageUploadRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.upload = record
	return r.err
}

func (r *workflowRepositoryStub) GetImageUpload(_ context.Context, _ int64, _ string) (domain.ImageUploadRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.upload, r.err
}

func (r *workflowRepositoryStub) CompleteImageUpload(_ context.Context, _ int64, _ string, _ int64, _ string, _ int, _ int, now time.Time) (domain.ImageUploadRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return domain.ImageUploadRecord{}, r.err
	}
	r.completed = r.upload
	r.completed.Status = "completed"
	r.completed.CompletedAt = &now
	r.upload = r.completed
	return r.completed, nil
}

func (r *workflowRepositoryStub) ListNotifications(_ context.Context, _ int64, limit int, _ string) (domain.NotificationPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notificationLimit = limit
	return domain.NotificationPage{}, r.err
}

func (r *workflowRepositoryStub) MarkNotificationRead(_ context.Context, _ int64, id *int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.markedNotification = id
	return r.err
}

func (r *workflowRepositoryStub) ListConversations(_ context.Context, _ int64, limit int, _ string) (domain.ConversationPage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.conversationLimit = limit
	return domain.ConversationPage{}, r.err
}

func (r *workflowRepositoryStub) ListDirectMessages(_ context.Context, _ int64, peer string, limit int, _ string) (domain.DirectMessagePage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peerName, r.messageLimit = peer, limit
	return domain.DirectMessagePage{}, r.err
}

func (r *workflowRepositoryStub) SendDirectMessage(_ context.Context, senderID int64, recipient, content string) (domain.DirectMessage, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recipientName, r.messageContent = recipient, content
	r.messageCalls++
	return domain.DirectMessage{ID: int64(r.messageCalls), SenderID: senderID, RecipientName: recipient, Content: content}, r.err
}

func (r *workflowRepositoryStub) GetUserBySessionTokenHash(_ context.Context, tokenHash string, _ time.Time) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revokedTokenHash = tokenHash
	return r.tokenUser, r.err
}

func (r *workflowRepositoryStub) RevokeAuthSession(_ context.Context, tokenHash string, _ time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.revokedTokenHash = tokenHash
	return r.err
}

func TestForumWorkflowValidationAndRepositoryErrors(t *testing.T) {
	repoErr := errors.New("repository unavailable")
	user := domain.User{ID: 7, Nickname: "tester"}

	t.Run("comment trims and rejects empty content", func(t *testing.T) {
		repo := &workflowRepositoryStub{}
		forum := NewForumService(repo, config.Config{}, nil)
		if _, err := forum.CreateComment(context.Background(), user, 11, domain.CreateCommentInput{Content: " \n "}); !errors.Is(err, ErrInvalidMessage) {
			t.Fatalf("empty comment error = %v", err)
		}
		comment, err := forum.CreateComment(context.Background(), user, 11, domain.CreateCommentInput{Content: "  useful reply  "})
		if err != nil || comment.Content != "useful reply" || repo.comment.Content != "useful reply" {
			t.Fatalf("comment=%+v stored=%+v err=%v", comment, repo.comment, err)
		}
		repo.err = repoErr
		if _, err := forum.CreateComment(context.Background(), user, 11, domain.CreateCommentInput{Content: "valid"}); !errors.Is(err, repoErr) {
			t.Fatalf("repository error = %v", err)
		}
	})

	t.Run("report trims fields and rejects missing reason", func(t *testing.T) {
		repo := &workflowRepositoryStub{}
		forum := NewForumService(repo, config.Config{}, nil)
		if _, err := forum.ReportPost(context.Background(), user, 12, domain.ReportPostInput{Reason: "  "}); !errors.Is(err, ErrInvalidReport) {
			t.Fatalf("empty report error = %v", err)
		}
		report, err := forum.ReportPost(context.Background(), user, 12, domain.ReportPostInput{Reason: " spam ", Detail: " repeated links "})
		if err != nil || report.Reason != "spam" || report.Detail != "repeated links" {
			t.Fatalf("report=%+v err=%v", report, err)
		}
		repo.err = repoErr
		if _, err := forum.ReportPost(context.Background(), user, 12, domain.ReportPostInput{Reason: "spam"}); !errors.Is(err, repoErr) {
			t.Fatalf("repository error = %v", err)
		}
	})
}

func TestForumPaginationDefaultsAndMessageNormalization(t *testing.T) {
	repo := &workflowRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)
	ctx := context.Background()

	if _, err := forum.ListPosts(ctx, nil, domain.FeedFilter{Limit: 99, Offset: -4}); err != nil {
		t.Fatal(err)
	}
	if repo.filter.Limit != 20 || repo.filter.Offset != 0 || repo.filter.Sort != domain.SortRecommended {
		t.Fatalf("normalized feed filter = %+v", repo.filter)
	}
	if _, err := forum.ListNotifications(ctx, 1, 101, "n"); err != nil {
		t.Fatal(err)
	}
	if _, err := forum.ListConversations(ctx, 1, -1, "c"); err != nil {
		t.Fatal(err)
	}
	if _, err := forum.ListDirectMessages(ctx, 1, "  peer  ", 101, "m"); err != nil {
		t.Fatal(err)
	}
	if repo.notificationLimit != 30 || repo.conversationLimit != 100 || repo.messageLimit != 50 || repo.peerName != "peer" {
		t.Fatalf("limits notification=%d conversation=%d messages=%d peer=%q", repo.notificationLimit, repo.conversationLimit, repo.messageLimit, repo.peerName)
	}
	if _, err := forum.SendDirectMessage(ctx, 1, domain.SendMessageInput{RecipientName: "peer", Content: " \t "}); !errors.Is(err, ErrInvalidMessage) {
		t.Fatalf("empty message error = %v", err)
	}
	message, err := forum.SendDirectMessage(ctx, 1, domain.SendMessageInput{RecipientName: "  peer  ", Content: "  hello  "})
	if err != nil || message.Content != "hello" || repo.recipientName != "peer" || repo.messageCalls != 1 {
		t.Fatalf("message=%+v recipient=%q calls=%d err=%v", message, repo.recipientName, repo.messageCalls, err)
	}
	id := int64(9)
	if err := forum.MarkNotificationRead(ctx, 1, &id); err != nil {
		t.Fatal(err)
	}
	if repo.markedNotification == nil || *repo.markedNotification != id {
		t.Fatal("notification ID was not forwarded")
	}
}

func TestImageUploadLifecycleAndErrors(t *testing.T) {
	ctx := context.Background()
	valid := domain.PresignImageUploadInput{FileName: " photo.JPEG ", ContentType: " Image/JPEG; charset=binary ", SizeBytes: 1024, Width: 640, Height: 480}

	for _, input := range []domain.PresignImageUploadInput{
		{FileName: "x.webp", ContentType: "image/webp", SizeBytes: 1, Width: 1, Height: 1},
		{FileName: "x.png", ContentType: "image/png", SizeBytes: MaxImageUploadBytes() + 1, Width: 1, Height: 1},
		{FileName: "x.png", ContentType: "image/png", SizeBytes: 1, Width: 0, Height: 1},
	} {
		forum := NewForumService(&workflowRepositoryStub{}, config.Config{}, nil)
		if _, err := forum.PresignImageUpload(ctx, 1, input, "https://upload.invalid"); !errors.Is(err, ErrInvalidUpload) {
			t.Fatalf("invalid presign input %+v returned %v", input, err)
		}
	}

	repo := &workflowRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)
	presigned, err := forum.PresignImageUpload(ctx, 7, valid, "https://upload.invalid")
	if err != nil {
		t.Fatal(err)
	}
	if presigned.Method != "PUT" || presigned.ContentType != "image/jpeg" || repo.upload.Ext != ".jpg" || repo.upload.Status != "pending" {
		t.Fatalf("presigned=%+v record=%+v", presigned, repo.upload)
	}
	if _, err := forum.GetImageUpload(ctx, 7, " "+presigned.ID+" "); err != nil {
		t.Fatal(err)
	}
	result, err := forum.CompleteImageUpload(ctx, 7, presigned.ID, 1024, "image/jpeg", 640, 480)
	if err != nil || result.URL != "/uploads/"+presigned.AssetKey {
		t.Fatalf("complete=%+v err=%v", result, err)
	}
	second, err := forum.CompleteImageUpload(ctx, 7, presigned.ID, 1024, "image/jpeg", 640, 480)
	if err != nil || second != result {
		t.Fatalf("idempotent complete=%+v err=%v", second, err)
	}
	if _, err := forum.GetImageUpload(ctx, 7, presigned.ID); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("completed pending lookup error=%v", err)
	}
	if record, err := forum.GetImageUploadForCompletion(ctx, 7, presigned.ID); err != nil || record.Status != "completed" {
		t.Fatalf("completion lookup=%+v err=%v", record, err)
	}
	if _, err := forum.CompleteImageUpload(ctx, 7, presigned.ID, 999, "image/jpeg", 640, 480); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("metadata mismatch error=%v", err)
	}

	expiredRepo := &workflowRepositoryStub{upload: domain.ImageUploadRecord{ID: "expired", Status: "pending", ExpiresAt: time.Now().Add(-time.Minute)}}
	expiredForum := NewForumService(expiredRepo, config.Config{}, nil)
	if _, err := expiredForum.GetImageUploadForCompletion(ctx, 7, "expired"); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("expired lookup error=%v", err)
	}
	if _, err := expiredForum.CompleteImageUpload(ctx, 7, "expired", 0, "", 0, 0); !errors.Is(err, ErrInvalidUpload) {
		t.Fatalf("expired complete error=%v", err)
	}

	repo.err = errors.New("storage failed")
	if _, err := forum.PresignImageUpload(ctx, 7, valid, "url"); !errors.Is(err, repo.err) {
		t.Fatalf("create upload error=%v", err)
	}
}

func TestConcurrentDirectMessagesAreRaceFreeAndNotDropped(t *testing.T) {
	repo := &workflowRepositoryStub{}
	forum := NewForumService(repo, config.Config{}, nil)
	const workers = 32
	var wg sync.WaitGroup
	errCh := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := forum.SendDirectMessage(context.Background(), 1, domain.SendMessageInput{RecipientName: " peer ", Content: " hello "})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if repo.messageCalls != workers {
		t.Fatalf("message calls=%d, want %d", repo.messageCalls, workers)
	}
}

func TestSessionTokenValidationAndRepositoryErrors(t *testing.T) {
	ctx := context.Background()
	repo := &workflowRepositoryStub{tokenUser: domain.User{ID: 12}}
	forum := NewForumService(repo, config.Config{}, nil)

	if _, err := forum.UserFromToken(ctx, " \t "); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("empty token error = %v", err)
	}
	user, err := forum.UserFromToken(ctx, " session-token ")
	if err != nil || user.ID != 12 || repo.revokedTokenHash != hashSessionToken("session-token") {
		t.Fatalf("user=%+v hash=%q err=%v", user, repo.revokedTokenHash, err)
	}
	previousHash := repo.revokedTokenHash
	if err := forum.Logout(ctx, " "); err != nil || repo.revokedTokenHash != previousHash {
		t.Fatalf("empty logout err=%v hash=%q", err, repo.revokedTokenHash)
	}
	repo.err = errors.New("session store failed")
	if err := forum.Logout(ctx, "session-token"); !errors.Is(err, repo.err) {
		t.Fatalf("logout repository error = %v", err)
	}
	if _, err := forum.UserFromToken(ctx, "session-token"); !errors.Is(err, repo.err) {
		t.Fatalf("lookup repository error = %v", err)
	}
}
