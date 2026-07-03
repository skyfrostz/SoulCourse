package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidElectives = errors.New("electives must contain two different subjects")
var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrInvalidEmailVerificationCode = errors.New("invalid or expired email verification code")

type ForumRepository interface {
	ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) ([]domain.Post, error)
	GetPost(ctx context.Context, viewerID *int64, id int64) (domain.Post, []domain.Comment, error)
	CreatePost(ctx context.Context, user domain.User, input domain.CreatePostInput) (domain.Post, error)
	CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error)
	ListInsights(ctx context.Context) ([]domain.SubjectInsight, error)
	GetInsight(ctx context.Context, id int64) (domain.SubjectInsight, error)
	ListTopics(ctx context.Context) ([]domain.Topic, error)
	GetTopic(ctx context.Context, viewerID *int64, slug string) (domain.TopicDetail, error)
	CreateUser(ctx context.Context, input domain.RegisterInput, passwordHash string) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, string, error)
	GetUserByID(ctx context.Context, id int64) (domain.User, error)
	CreateEmailVerificationCode(ctx context.Context, email string, codeHash string, expiresAt time.Time) error
	ConsumeEmailVerificationCode(ctx context.Context, email string, codeHash string) error
	TogglePostLike(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error)
	TogglePostFavorite(ctx context.Context, userID int64, postID int64) (domain.ToggleResult, error)
	ToggleFollowAuthor(ctx context.Context, followerID int64, authorName string) (bool, error)
}

type ForumService struct {
	repo                 ForumRepository
	jwtSecret            []byte
	emailSender          EmailSender
	emailVerificationTTL time.Duration
	emailDebugMode       bool
}

func NewForumService(repo ForumRepository, cfg config.Config, emailSender EmailSender) *ForumService {
	ttl := time.Duration(cfg.EmailVerificationTTLMinutes) * time.Minute
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &ForumService{
		repo:                 repo,
		jwtSecret:            []byte(cfg.JWTSecret),
		emailSender:          emailSender,
		emailVerificationTTL: ttl,
		emailDebugMode:       cfg.AppEnv == "local" || cfg.AppEnv == "development",
	}
}

func (s *ForumService) ListPosts(ctx context.Context, viewerID *int64, filter domain.FeedFilter) ([]domain.Post, error) {
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
	return s.repo.CreatePost(ctx, user, input)
}

func (s *ForumService) CreateComment(ctx context.Context, user domain.User, postID int64, input domain.CreateCommentInput) (domain.Comment, error) {
	return s.repo.CreateComment(ctx, user, postID, input)
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
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.VerificationCode = strings.TrimSpace(input.VerificationCode)
	if err := s.repo.ConsumeEmailVerificationCode(ctx, input.Email, hashVerificationCode(input.Email, input.VerificationCode)); err != nil {
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
	token, err := s.issueToken(user)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return domain.AuthSession{User: user, Token: token}, nil
}

func (s *ForumService) SendEmailVerificationCode(ctx context.Context, input domain.EmailVerificationCodeInput) (domain.EmailVerificationCodeResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	code, err := generateVerificationCode()
	if err != nil {
		return domain.EmailVerificationCodeResult{}, err
	}
	expiresAt := time.Now().Add(s.emailVerificationTTL)
	if err := s.repo.CreateEmailVerificationCode(ctx, email, hashVerificationCode(email, code), expiresAt); err != nil {
		return domain.EmailVerificationCodeResult{}, err
	}
	if s.emailSender != nil && s.emailSender.Enabled() {
		if err := s.emailSender.SendVerificationCode(ctx, email, code, s.emailVerificationTTL); err != nil {
			return domain.EmailVerificationCodeResult{}, err
		}
		return domain.EmailVerificationCodeResult{Email: email, ExpiresInSeconds: int(s.emailVerificationTTL.Seconds())}, nil
	}
	if !s.emailDebugMode {
		return domain.EmailVerificationCodeResult{}, errors.New("email sender is not configured")
	}

	return domain.EmailVerificationCodeResult{
		Email:            email,
		ExpiresInSeconds: int(s.emailVerificationTTL.Seconds()),
		DebugCode:        code,
	}, nil
}

func (s *ForumService) Login(ctx context.Context, input domain.LoginInput) (domain.AuthSession, error) {
	user, hash, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return domain.AuthSession{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(input.Password)); err != nil {
		return domain.AuthSession{}, ErrInvalidCredentials
	}
	token, err := s.issueToken(user)
	if err != nil {
		return domain.AuthSession{}, err
	}
	return domain.AuthSession{User: user, Token: token}, nil
}

func (s *ForumService) UserFromToken(ctx context.Context, tokenString string) (domain.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return domain.User{}, ErrInvalidCredentials
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		return domain.User{}, err
	}
	userID, err := parseInt64(subject)
	if err != nil {
		return domain.User{}, err
	}
	return s.repo.GetUserByID(ctx, userID)
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

func (s *ForumService) issueToken(user domain.User) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", user.ID),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

func parseInt64(value string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(value, "%d", &id)
	return id, err
}

func generateVerificationCode() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func hashVerificationCode(email string, code string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(email)) + ":" + strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}
