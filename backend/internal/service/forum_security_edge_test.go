package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

type securityRepositoryStub struct {
	ForumRepository
	consumeErr        error
	updatePasswordErr error
	revokeAllErr      error
	passwordHash      string
	passwordHashErr   error
	deleteErr         error
	updated           bool
	revokedAll        bool
	deleted           bool
	revokedID         bool
	revokeIDErr       error
}

func (r *securityRepositoryStub) ConsumeEmailVerificationCode(context.Context, string, string, int) error {
	return r.consumeErr
}

func (r *securityRepositoryStub) UpdateUserPasswordByEmail(context.Context, string, string, time.Time) (int64, error) {
	r.updated = true
	return 42, r.updatePasswordErr
}

func (r *securityRepositoryStub) RevokeAuthSessionsForUser(context.Context, int64, time.Time) error {
	r.revokedAll = true
	return r.revokeAllErr
}

func (r *securityRepositoryStub) GetUserPasswordHashByID(context.Context, int64) (string, error) {
	return r.passwordHash, r.passwordHashErr
}

func (r *securityRepositoryStub) DeleteUserAccount(context.Context, int64, time.Time) error {
	r.deleted = true
	return r.deleteErr
}

func (r *securityRepositoryStub) RevokeAuthSessionByID(context.Context, int64, int64, time.Time) error {
	r.revokedID = true
	return r.revokeIDErr
}

func TestResetPasswordStopsBeforeUnsafeSideEffects(t *testing.T) {
	t.Run("invalid verification code", func(t *testing.T) {
		repository := &securityRepositoryStub{consumeErr: errors.New("invalid code")}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		err := forum.ResetPassword(context.Background(), domain.ResetPasswordInput{Email: "A@Example.com", VerificationCode: "bad", Password: "new-password"})
		if !errors.Is(err, ErrInvalidEmailVerificationCode) {
			t.Fatalf("error = %v", err)
		}
		if repository.updated || repository.revokedAll {
			t.Fatal("invalid code must not update password or revoke sessions")
		}
	})

	t.Run("password update failure", func(t *testing.T) {
		updateErr := errors.New("update failed")
		repository := &securityRepositoryStub{updatePasswordErr: updateErr}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		err := forum.ResetPassword(context.Background(), domain.ResetPasswordInput{Email: "a@example.com", VerificationCode: "123456", Password: "new-password"})
		if !errors.Is(err, updateErr) || repository.revokedAll {
			t.Fatalf("error=%v revoked=%v", err, repository.revokedAll)
		}
	})

	t.Run("session revoke failure", func(t *testing.T) {
		revokeErr := errors.New("revoke failed")
		repository := &securityRepositoryStub{revokeAllErr: revokeErr}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		err := forum.ResetPassword(context.Background(), domain.ResetPasswordInput{Email: "a@example.com", VerificationCode: "123456", Password: "new-password"})
		if !errors.Is(err, revokeErr) || !repository.updated || !repository.revokedAll {
			t.Fatalf("error=%v updated=%v revoked=%v", err, repository.updated, repository.revokedAll)
		}
	})
}

func TestDeleteAccountAuthenticationAndFailureBoundaries(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("password lookup failure", func(t *testing.T) {
		lookupErr := errors.New("lookup failed")
		repository := &securityRepositoryStub{passwordHashErr: lookupErr}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		if err := forum.DeleteAccount(context.Background(), 1, domain.DeleteAccountInput{Password: "correct-password"}); !errors.Is(err, lookupErr) {
			t.Fatalf("error = %v", err)
		}
		if repository.deleted {
			t.Fatal("lookup failure must not delete account")
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		repository := &securityRepositoryStub{passwordHash: string(hash)}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		if err := forum.DeleteAccount(context.Background(), 1, domain.DeleteAccountInput{Password: "wrong"}); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("error = %v", err)
		}
		if repository.deleted {
			t.Fatal("wrong password must not delete account")
		}
	})

	t.Run("delete failure", func(t *testing.T) {
		deleteErr := errors.New("delete failed")
		repository := &securityRepositoryStub{passwordHash: string(hash), deleteErr: deleteErr}
		forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
		if err := forum.DeleteAccount(context.Background(), 1, domain.DeleteAccountInput{Password: "correct-password"}); !errors.Is(err, deleteErr) {
			t.Fatalf("error = %v", err)
		}
		if !repository.deleted {
			t.Fatal("expected delete attempt")
		}
	})
}

func TestRevokeAuthSessionRejectsInvalidIDAndPropagatesRepositoryError(t *testing.T) {
	repository := &securityRepositoryStub{}
	forum := NewForumService(repository, config.Config{JWTSecret: "test"}, nil)
	if err := forum.RevokeAuthSessionByID(context.Background(), 1, 0); err == nil {
		t.Fatal("expected invalid session ID error")
	}
	if repository.revokedID {
		t.Fatal("invalid session ID must not reach repository")
	}

	revokeErr := errors.New("revoke failed")
	repository.revokeIDErr = revokeErr
	if err := forum.RevokeAuthSessionByID(context.Background(), 1, 9); !errors.Is(err, revokeErr) {
		t.Fatalf("error = %v", err)
	}
	if !repository.revokedID {
		t.Fatal("valid session ID should reach repository")
	}
}
