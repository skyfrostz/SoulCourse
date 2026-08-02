package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidElectives = errors.New("electives must contain two different subjects")
var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrInvalidEmailVerificationCode = errors.New("invalid or expired email verification code")
var ErrInvalidMessage = errors.New("message content is required")
var ErrInvalidUpload = errors.New("invalid image upload")
var ErrInvalidPostImages = errors.New("post images must be completed local uploads")
var ErrInvalidReport = errors.New("report reason is required")
var ErrForbidden = errors.New("operation is not allowed")

type PostTagger interface {
	TagPost(ctx context.Context, title string, content string) ([]string, error)
}

type EmailVerificationRateLimitError struct {
	Limit domain.EmailVerificationAttemptLimit
}

func (e *EmailVerificationRateLimitError) Error() string {
	return "email verification request rate limited"
}

type ForumRepository interface {
	ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) (domain.PostPage, error)
	GetPost(ctx context.Context, viewerID *int64, id int64) (domain.Post, []domain.Comment, error)
	CreatePost(ctx context.Context, user domain.User, input domain.CreatePostInput) (domain.Post, error)
	UpdatePost(ctx context.Context, userID int64, postID int64, input domain.UpdatePostInput) (domain.Post, error)
	DeletePost(ctx context.Context, userID int64, postID int64) error
	CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error)
	ReportPost(ctx context.Context, user domain.User, postID int64, input domain.ReportPostInput) (domain.ContentReport, error)
	ListInsights(ctx context.Context) ([]domain.SubjectInsight, error)
	GetInsight(ctx context.Context, id int64) (domain.SubjectInsight, error)
	ListTopics(ctx context.Context) ([]domain.Topic, error)
	GetTopic(ctx context.Context, viewerID *int64, slug string) (domain.TopicDetail, error)
	CreateUser(ctx context.Context, input domain.RegisterInput, passwordHash string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, string, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	GetUserPasswordHashByID(ctx context.Context, id int64) (string, error)
	UpdateUserPasswordByEmail(ctx context.Context, email string, passwordHash string, now time.Time) (int64, error)
	DeleteUserAccount(ctx context.Context, userID int64, now time.Time) error
	CreateImageUpload(ctx context.Context, record domain.ImageUploadRecord) error
	GetImageUpload(ctx context.Context, userID int64, id string) (domain.ImageUploadRecord, error)
	CompleteImageUpload(ctx context.Context, userID int64, id string, sizeBytes int64, contentType string, width int, height int, now time.Time) (domain.ImageUploadRecord, error)
	CreateAuthSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error
	GetUserBySessionTokenHash(ctx context.Context, tokenHash string, now time.Time) (domain.User, error)
	RevokeAuthSession(ctx context.Context, tokenHash string, now time.Time) error
	RevokeAuthSessionsForUser(ctx context.Context, userID int64, now time.Time) error
	ListAuthSessions(ctx context.Context, userID int64, currentTokenHash string, now time.Time) ([]domain.AccountSession, error)
	RevokeAuthSessionByID(ctx context.Context, userID int64, sessionID int64, now time.Time) error
	ReserveEmailVerificationAttempt(ctx context.Context, email string, clientIP string, now time.Time, cooldown time.Duration, emailHourlyLimit int, ipHourlyLimit int) (domain.EmailVerificationAttemptLimit, error)
	CreateEmailVerificationCode(ctx context.Context, email string, codeHash string, expiresAt time.Time) error
	ConsumeEmailVerificationCode(ctx context.Context, email string, codeHash string, maxAttempts int) error
	TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error)
	TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error)
	ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error)
	GetAccountProfile(ctx context.Context, viewerID *int64, name string) (domain.AccountProfile, error)
	GetAccountProfileByUserID(ctx context.Context, viewerID *int64, userID int64) (domain.AccountProfile, error)
	UpdateAccountProfile(ctx context.Context, userID int64, input domain.UpdateProfileInput) (domain.AccountProfile, error)
	ListNotifications(ctx context.Context, userID int64, limit int, cursor string) (domain.NotificationPage, error)
	MarkNotificationRead(ctx context.Context, userID int64, notificationID *int64) error
	ListConversations(ctx context.Context, userID int64, limit int, cursor string) (domain.ConversationPage, error)
	ListDirectMessages(ctx context.Context, userID int64, peerName string, limit int, cursor string) (domain.DirectMessagePage, error)
	SendDirectMessage(ctx context.Context, senderID int64, recipientName string, content string) (domain.DirectMessage, error)
}

type ForumService struct {
	repo                                   ForumRepository
	postTagger                             PostTagger
	onPostTagError                         func(error)
	sessionTTL                             time.Duration
	emailSender                            EmailSender
	emailVerificationTTL                   time.Duration
	emailVerificationCooldown              time.Duration
	emailVerificationEmailHourlyLimit      int
	emailVerificationIPHourlyLimit         int
	emailVerificationMaxValidationAttempts int
	emailDebugMode                         bool
}

const maxImageUploadBytes int64 = 8 * 1024 * 1024
const imageUploadTTL = 15 * time.Minute

func MaxImageUploadBytes() int64 {
	return maxImageUploadBytes
}

func (s *ForumService) ConfigurePostTagger(tagger PostTagger, onError func(error)) {
	s.postTagger = tagger
	s.onPostTagError = onError
}

func NewForumService(repo ForumRepository, cfg config.Config, emailSender EmailSender) *ForumService {
	ttl := time.Duration(cfg.EmailVerificationTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	cooldownSeconds := cfg.EmailVerificationCooldownSeconds
	if cooldownSeconds <= 0 {
		cooldownSeconds = 60
	}
	emailHourlyLimit := cfg.EmailVerificationEmailHourlyLimit
	if emailHourlyLimit <= 0 {
		emailHourlyLimit = 5
	}
	ipHourlyLimit := cfg.EmailVerificationIPHourlyLimit
	if ipHourlyLimit <= 0 {
		ipHourlyLimit = 20
	}
	maxValidationAttempts := cfg.EmailVerificationMaxValidationAttempts
	if maxValidationAttempts <= 0 {
		maxValidationAttempts = 5
	}
	return &ForumService{
		repo:                                   repo,
		sessionTTL:                             7 * 24 * time.Hour,
		emailSender:                            emailSender,
		emailVerificationTTL:                   ttl,
		emailVerificationCooldown:              time.Duration(cooldownSeconds) * time.Second,
		emailVerificationEmailHourlyLimit:      emailHourlyLimit,
		emailVerificationIPHourlyLimit:         ipHourlyLimit,
		emailVerificationMaxValidationAttempts: maxValidationAttempts,
		emailDebugMode:                         cfg.AppEnv == "local" || cfg.AppEnv == "development",
	}
}

func (s *ForumService) ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) (domain.PostPage, error) {
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	if filter.Sort == "" {
		filter.Sort = domain.SortRecommended
	}
	return s.repo.ListPosts(ctx, viewerID, filter)
}

func (s *ForumService) GetPost(ctx context.Context, viewerID *int64, id int64) (domain.Post, []domain.Comment, error) {
	return s.repo.GetPost(ctx, viewerID, id)
}

func (s *ForumService) CreatePost(ctx context.Context, user domain.User, input domain.CreatePostInput) (domain.Post, error) {
	if len(input.Electives) != 2 || input.Electives[0] == input.Electives[1] {
		return domain.Post{}, ErrInvalidElectives
	}
	imageURLs, err := s.validatePostImageURLs(ctx, user.ID, input.ImageURLs)
	if err != nil {
		return domain.Post{}, err
	}
	input.ImageURLs = imageURLs
	tags := normalizePostTags(input.Tags)
	if len(tags) == 0 && s.postTagger != nil {
		aiTags, err := s.postTagger.TagPost(ctx, input.Title, input.Content)
		if err != nil {
			if s.onPostTagError != nil {
				s.onPostTagError(err)
			}
		} else {
			for _, tag := range aiTags {
				if domain.IsControlledTag(tag) {
					tags = appendUniqueTag(tags, tag)
				}
			}
		}
	}
	if subjectTag, ok := domain.SubjectTagForChoice(input.Track, input.Electives); ok {
		tags = appendUniqueTag(tags, subjectTag)
	}
	input.Tags = tags
	return s.repo.CreatePost(ctx, user, input)
}

func (s *ForumService) UpdatePost(ctx context.Context, userID int64, postID int64, input domain.UpdatePostInput) (domain.Post, error) {
	if len(input.Electives) != 2 || input.Electives[0] == input.Electives[1] {
		return domain.Post{}, ErrInvalidElectives
	}
	tags := normalizePostTags(input.Tags)
	if subjectTag, ok := domain.SubjectTagForChoice(input.Track, input.Electives); ok {
		tags = appendUniqueTag(tags, subjectTag)
	}
	input.Tags = tags
	return s.repo.UpdatePost(ctx, userID, postID, input)
}

func (s *ForumService) DeletePost(ctx context.Context, userID int64, postID int64) error {
	if userID <= 0 || postID <= 0 {
		return ErrForbidden
	}
	return s.repo.DeletePost(ctx, userID, postID)
}

func (s *ForumService) validatePostImageURLs(ctx context.Context, userID int64, imageURLs []string) ([]string, error) {
	if len(imageURLs) > 9 {
		return nil, ErrInvalidPostImages
	}
	result := make([]string, 0, len(imageURLs))
	for _, imageURL := range imageURLs {
		imageURL = strings.TrimSpace(imageURL)
		if imageURL == "" {
			continue
		}
		lower := strings.ToLower(imageURL)
		if len(imageURL) > 512 ||
			strings.HasPrefix(lower, "data:") ||
			strings.HasPrefix(lower, "http://") ||
			strings.HasPrefix(lower, "https://") ||
			strings.HasPrefix(lower, "blob:") ||
			!strings.HasPrefix(imageURL, "/") ||
			!strings.Contains(imageURL, "/uploads/images/") {
			return nil, ErrInvalidPostImages
		}
		uploadPath := imageURL[strings.Index(imageURL, "/uploads/")+len("/uploads/"):]
		uploadID := uploadIDFromAssetKey(uploadPath)
		if uploadID == "" {
			return nil, ErrInvalidPostImages
		}
		record, err := s.repo.GetImageUpload(ctx, userID, uploadID)
		if err != nil || record.Status != "completed" || record.AssetKey != uploadPath {
			return nil, ErrInvalidPostImages
		}
		result = append(result, imageURL)
	}
	return result, nil
}

func uploadIDFromAssetKey(assetKey string) string {
	if !strings.HasPrefix(assetKey, "images/") {
		return ""
	}
	name := assetKey[strings.LastIndex(assetKey, "/")+1:]
	extIndex := strings.LastIndex(name, ".")
	if extIndex <= 0 {
		return ""
	}
	return name[:extIndex]
}

func normalizePostTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		result = appendUniqueTag(result, truncateRunes(tag, 24))
		if len(result) == 8 {
			break
		}
	}
	return result
}

func appendUniqueTag(tags []string, tag string) []string {
	for _, existing := range tags {
		if existing == tag {
			return tags
		}
	}
	return append(tags, tag)
}

func (s *ForumService) CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error) {
	input.Content = strings.TrimSpace(input.Content)
	if input.Content == "" {
		return domain.Comment{}, ErrInvalidMessage
	}
	return s.repo.CreateComment(ctx, user, postID, input)
}

func (s *ForumService) ReportPost(ctx context.Context, user domain.User, postID int64, input domain.ReportPostInput) (domain.ContentReport, error) {
	input.Reason = strings.TrimSpace(input.Reason)
	input.Detail = strings.TrimSpace(input.Detail)
	if input.Reason == "" {
		return domain.ContentReport{}, ErrInvalidReport
	}
	return s.repo.ReportPost(ctx, user, postID, input)
}

func (s *ForumService) ListInsights(ctx context.Context) ([]domain.SubjectInsight, error) {
	return s.repo.ListInsights(ctx)
}

func (s *ForumService) GetInsight(ctx context.Context, id int64) (domain.SubjectInsight, error) {
	return s.repo.GetInsight(ctx, id)
}

func (s *ForumService) ListTopics(ctx context.Context) ([]domain.Topic, error) {
	return s.repo.ListTopics(ctx)
}

func (s *ForumService) GetTopic(ctx context.Context, viewerID *int64, slug string) (domain.TopicDetail, error) {
	return s.repo.GetTopic(ctx, viewerID, slug)
}

func (s *ForumService) Register(ctx context.Context, input domain.RegisterInput) (domain.AuthSession, error) {
	return s.registerWithTTL(ctx, input, s.sessionTTL)
}

func (s *ForumService) registerWithTTL(ctx context.Context, input domain.RegisterInput, ttl time.Duration) (domain.AuthSession, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if err := s.repo.ConsumeEmailVerificationCode(
		ctx,
		input.Email,
		hashVerificationCode(input.Email, input.VerificationCode),
		s.emailVerificationMaxValidationAttempts,
	); err != nil {
		return domain.AuthSession{}, ErrInvalidEmailVerificationCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return domain.AuthSession{}, err
	}
	user, err := s.repo.CreateUser(ctx, input, string(hash))
	if err != nil {
		return domain.AuthSession{}, err
	}
	session, err := s.issueSessionWithTTL(ctx, user, ttl)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return session, nil
}

func (s *ForumService) SendEmailVerificationCode(ctx context.Context, input domain.EmailVerificationCodeInput) (domain.EmailVerificationCodeResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	now := time.Now()
	limit, err := s.repo.ReserveEmailVerificationAttempt(
		ctx,
		email,
		input.ClientIP,
		now,
		s.emailVerificationCooldown,
		s.emailVerificationEmailHourlyLimit,
		s.emailVerificationIPHourlyLimit,
	)
	if err != nil {
		return domain.EmailVerificationCodeResult{}, err
	}
	if !limit.Allowed {
		return domain.EmailVerificationCodeResult{}, &EmailVerificationRateLimitError{Limit: limit}
	}

	code, err := generateVerificationCode()
	if err != nil {
		return domain.EmailVerificationCodeResult{}, err
	}
	expiresAt := now.Add(s.emailVerificationTTL)
	if err := s.repo.CreateEmailVerificationCode(ctx, email, hashVerificationCode(email, code), expiresAt); err != nil {
		return domain.EmailVerificationCodeResult{}, err
	}
	result := domain.EmailVerificationCodeResult{
		Email:             email,
		ExpiresInSeconds:  int(s.emailVerificationTTL.Seconds()),
		RetryAfterSeconds: limit.RetryAfterSeconds,
		HourlyLimit:       limit.EmailHourlyLimit,
		HourlyRemaining:   limit.EmailHourlyRemaining,
	}
	if s.emailSender != nil && s.emailSender.Enabled() {
		if err := s.emailSender.SendVerificationCode(ctx, email, code, s.emailVerificationTTL); err != nil {
			return domain.EmailVerificationCodeResult{}, err
		}
		return result, nil
	}
	if !s.emailDebugMode {
		return domain.EmailVerificationCodeResult{}, errors.New("email sender is not configured")
	}

	result.DebugCode = code
	return result, nil
}

func (s *ForumService) Login(ctx context.Context, input domain.LoginInput) (domain.AuthSession, error) {
	user, hash, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return domain.AuthSession{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)); err != nil {
		return domain.AuthSession{}, ErrInvalidCredentials
	}
	session, err := s.issueSession(ctx, user)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return session, nil
}

func (s *ForumService) RegisterMobile(ctx context.Context, input domain.RegisterInput) (domain.MobileAuthSession, error) {
	session, err := s.registerWithTTL(ctx, input, 30*24*time.Hour)
	if err != nil {
		return domain.MobileAuthSession{}, err
	}
	return domain.MobileAuthSession{User: session.User, AccessToken: session.Token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *ForumService) LoginMobile(ctx context.Context, input domain.LoginInput) (domain.MobileAuthSession, error) {
	user, passwordHash, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)) != nil {
		return domain.MobileAuthSession{}, ErrInvalidCredentials
	}
	session, err := s.issueSessionWithTTL(ctx, user, 30*24*time.Hour)
	if err != nil {
		return domain.MobileAuthSession{}, err
	}
	return domain.MobileAuthSession{User: session.User, AccessToken: session.Token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *ForumService) RefreshMobile(ctx context.Context, token string) (domain.MobileAuthSession, error) {
	user, err := s.UserFromToken(ctx, token)
	if err != nil {
		return domain.MobileAuthSession{}, ErrInvalidCredentials
	}
	if err := s.Logout(ctx, token); err != nil {
		return domain.MobileAuthSession{}, err
	}
	session, err := s.issueSessionWithTTL(ctx, user, 30*24*time.Hour)
	if err != nil {
		return domain.MobileAuthSession{}, err
	}
	return domain.MobileAuthSession{User: session.User, AccessToken: session.Token, ExpiresAt: session.ExpiresAt}, nil
}

func (s *ForumService) ResetPassword(ctx context.Context, input domain.ResetPasswordInput) error {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	code := strings.TrimSpace(input.VerificationCode)
	if err := s.repo.ConsumeEmailVerificationCode(
		ctx,
		email,
		hashVerificationCode(email, code),
		s.emailVerificationMaxValidationAttempts,
	); err != nil {
		return ErrInvalidEmailVerificationCode
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	userID, err := s.repo.UpdateUserPasswordByEmail(ctx, email, string(hash), time.Now())
	if err != nil {
		return err
	}
	return s.repo.RevokeAuthSessionsForUser(ctx, userID, time.Now())
}

func (s *ForumService) DeleteAccount(ctx context.Context, userID int64, input domain.DeleteAccountInput) error {
	hash, err := s.repo.GetUserPasswordHashByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)); err != nil {
		return ErrInvalidCredentials
	}
	return s.repo.DeleteUserAccount(ctx, userID, time.Now())
}

func (s *ForumService) PresignImageUpload(ctx context.Context, userID int64, input domain.PresignImageUploadInput, uploadURL string) (domain.PresignedImageUpload, error) {
	contentType, ext, ok := normalizeImageUploadType(input.ContentType, input.FileName)
	if !ok || input.SizeBytes <= 0 || input.SizeBytes > maxImageUploadBytes || input.Width <= 0 || input.Height <= 0 || input.Width > 12000 || input.Height > 12000 {
		return domain.PresignedImageUpload{}, ErrInvalidUpload
	}
	now := time.Now().UTC()
	id := generateUploadID()
	assetKey := "images/" + now.Format("2006/01/02") + "/" + id + ext
	record := domain.ImageUploadRecord{
		ID:          id,
		UserID:      userID,
		AssetKey:    assetKey,
		FileName:    strings.TrimSpace(input.FileName),
		ContentType: contentType,
		Ext:         ext,
		SizeBytes:   input.SizeBytes,
		Width:       input.Width,
		Height:      input.Height,
		Status:      "pending",
		CreatedAt:   now,
		ExpiresAt:   now.Add(imageUploadTTL),
	}
	if err := s.repo.CreateImageUpload(ctx, record); err != nil {
		return domain.PresignedImageUpload{}, err
	}
	return domain.PresignedImageUpload{
		ID:          id,
		AssetKey:    assetKey,
		UploadURL:   uploadURL,
		Method:      "PUT",
		ContentType: contentType,
		MaxBytes:    maxImageUploadBytes,
		ExpiresAt:   record.ExpiresAt,
	}, nil
}

func (s *ForumService) GetImageUpload(ctx context.Context, userID int64, id string) (domain.ImageUploadRecord, error) {
	record, err := s.repo.GetImageUpload(ctx, userID, strings.TrimSpace(id))
	if err != nil {
		return domain.ImageUploadRecord{}, err
	}
	if record.Status != "pending" || time.Now().After(record.ExpiresAt) {
		return domain.ImageUploadRecord{}, ErrInvalidUpload
	}
	return record, nil
}

func (s *ForumService) GetImageUploadForCompletion(ctx context.Context, userID int64, id string) (domain.ImageUploadRecord, error) {
	record, err := s.repo.GetImageUpload(ctx, userID, strings.TrimSpace(id))
	if err != nil {
		return domain.ImageUploadRecord{}, err
	}
	if record.Status == "completed" {
		return record, nil
	}
	if record.Status != "pending" || time.Now().After(record.ExpiresAt) {
		return domain.ImageUploadRecord{}, ErrInvalidUpload
	}
	return record, nil
}

func (s *ForumService) CompleteImageUpload(ctx context.Context, userID int64, id string, sizeBytes int64, contentType string, width int, height int) (domain.CompleteImageUploadResult, error) {
	record, err := s.repo.GetImageUpload(ctx, userID, strings.TrimSpace(id))
	if err != nil {
		return domain.CompleteImageUploadResult{}, err
	}
	metadataMatches := record.SizeBytes == sizeBytes && record.ContentType == contentType && record.Width == width && record.Height == height
	if !metadataMatches || (record.Status != "pending" && record.Status != "completed") || (record.Status == "pending" && time.Now().After(record.ExpiresAt)) {
		return domain.CompleteImageUploadResult{}, ErrInvalidUpload
	}
	completed := record
	if record.Status == "pending" {
		completed, err = s.repo.CompleteImageUpload(ctx, userID, record.ID, sizeBytes, contentType, width, height, time.Now())
		if err != nil {
			return domain.CompleteImageUploadResult{}, err
		}
	}
	return domain.CompleteImageUploadResult{
		ID:          completed.ID,
		AssetKey:    completed.AssetKey,
		URL:         "/uploads/" + completed.AssetKey,
		ContentType: completed.ContentType,
		SizeBytes:   completed.SizeBytes,
		Width:       completed.Width,
		Height:      completed.Height,
	}, nil
}

func (s *ForumService) UserFromToken(ctx context.Context, tokenString string) (domain.User, error) {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return domain.User{}, ErrInvalidCredentials
	}
	return s.repo.GetUserBySessionTokenHash(ctx, hashSessionToken(tokenString), time.Now())
}

func (s *ForumService) Logout(ctx context.Context, tokenString string) error {
	tokenString = strings.TrimSpace(tokenString)
	if tokenString == "" {
		return nil
	}
	return s.repo.RevokeAuthSession(ctx, hashSessionToken(tokenString), time.Now())
}

func (s *ForumService) ListAuthSessions(ctx context.Context, userID int64, currentToken string) ([]domain.AccountSession, error) {
	currentHash := ""
	if strings.TrimSpace(currentToken) != "" {
		currentHash = hashSessionToken(strings.TrimSpace(currentToken))
	}
	return s.repo.ListAuthSessions(ctx, userID, currentHash, time.Now())
}

func (s *ForumService) RevokeAuthSessionByID(ctx context.Context, userID int64, sessionID int64) error {
	if sessionID <= 0 {
		return sql.ErrNoRows
	}
	return s.repo.RevokeAuthSessionByID(ctx, userID, sessionID, time.Now())
}

func (s *ForumService) TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return s.repo.TogglePostLike(ctx, userID, postID)
}

func (s *ForumService) TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error) {
	return s.repo.TogglePostFavorite(ctx, userID, postID)
}

func (s *ForumService) ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error) {
	return s.repo.ToggleFollowAuthor(ctx, followerID, authorName)
}

func (s *ForumService) GetAccountProfile(ctx context.Context, viewerID *int64, name string) (domain.AccountProfile, error) {
	return s.repo.GetAccountProfile(ctx, viewerID, strings.TrimSpace(name))
}

func (s *ForumService) GetAccountProfileByUserID(ctx context.Context, viewerID *int64, userID int64) (domain.AccountProfile, error) {
	return s.repo.GetAccountProfileByUserID(ctx, viewerID, userID)
}

func (s *ForumService) UpdateAccountProfile(ctx context.Context, userID int64, input domain.UpdateProfileInput) (domain.AccountProfile, error) {
	if len(input.ChoiceProfile.PreferredSubjects) != 2 || input.ChoiceProfile.PreferredSubjects[0] == input.ChoiceProfile.PreferredSubjects[1] {
		return domain.AccountProfile{}, ErrInvalidElectives
	}
	return s.repo.UpdateAccountProfile(ctx, userID, input)
}

func (s *ForumService) ListNotifications(ctx context.Context, userID int64, limit int, cursor string) (domain.NotificationPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.repo.ListNotifications(ctx, userID, limit, cursor)
}

func (s *ForumService) MarkNotificationRead(ctx context.Context, userID int64, notificationID *int64) error {
	return s.repo.MarkNotificationRead(ctx, userID, notificationID)
}

func (s *ForumService) ListConversations(ctx context.Context, userID int64, limit int, cursor string) (domain.ConversationPage, error) {
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	return s.repo.ListConversations(ctx, userID, limit, cursor)
}

func (s *ForumService) ListDirectMessages(ctx context.Context, userID int64, peerName string, limit int, cursor string) (domain.DirectMessagePage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.repo.ListDirectMessages(ctx, userID, strings.TrimSpace(peerName), limit, cursor)
}

func (s *ForumService) SendDirectMessage(ctx context.Context, senderID int64, input domain.SendMessageInput) (domain.DirectMessage, error) {
	content := strings.TrimSpace(input.Content)
	if content == "" {
		return domain.DirectMessage{}, ErrInvalidMessage
	}
	return s.repo.SendDirectMessage(ctx, senderID, strings.TrimSpace(input.RecipientName), content)
}

func (s *ForumService) issueSession(ctx context.Context, user domain.User) (domain.AuthSession, error) {
	return s.issueSessionWithTTL(ctx, user, s.sessionTTL)
}

func (s *ForumService) issueSessionWithTTL(ctx context.Context, user domain.User, ttl time.Duration) (domain.AuthSession, error) {
	token, err := generateSessionToken()
	if err != nil {
		return domain.AuthSession{}, err
	}
	expiresAt := time.Now().Add(ttl).UTC()
	if err := s.repo.CreateAuthSession(ctx, user.ID, hashSessionToken(token), expiresAt); err != nil {
		return domain.AuthSession{}, err
	}
	return domain.AuthSession{User: user, Token: token, ExpiresAt: expiresAt}, nil
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func generateSessionToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func generateUploadID() string {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(token)
}

func normalizeImageUploadType(contentType string, fileName string) (string, string, bool) {
	normalizedType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	ext := strings.ToLower(strings.TrimSpace(pathExt(fileName)))
	switch normalizedType {
	case "image/jpeg":
		return normalizedType, ".jpg", ext == ".jpg" || ext == ".jpeg"
	case "image/png":
		return normalizedType, ".png", ext == ".png"
	case "image/gif":
		return normalizedType, ".gif", ext == ".gif"
	default:
		return "", "", false
	}
}

func pathExt(fileName string) string {
	index := strings.LastIndex(fileName, ".")
	if index < 0 {
		return ""
	}
	return fileName[index:]
}

func hashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hashVerificationCode(email string, code string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}
