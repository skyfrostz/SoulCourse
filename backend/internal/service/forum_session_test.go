package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/domain"
	sqliterepo "subject-choice-forum/backend/internal/repository/sqlite"
	"subject-choice-forum/backend/internal/storage"

	"golang.org/x/crypto/bcrypt"
)

func TestLoginIssuesRevocableServerSession(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	repository := sqliterepo.NewForumRepository(db)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: "login-session@example.com", Nickname: "会话用户", Role: "student", Province: "广东", Grade: "高一",
	}, string(passwordHash))
	if err != nil {
		t.Fatal(err)
	}

	forum := NewForumService(repository, config.Config{JWTSecret: "test-secret"}, nil)
	session, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "correct-password"})
	if err != nil {
		t.Fatal(err)
	}
	if session.Token == "" || session.ExpiresAt.IsZero() {
		t.Fatalf("login returned incomplete session: %+v", session)
	}
	authed, err := forum.UserFromToken(context.Background(), session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if authed.ID != user.ID {
		t.Fatalf("token resolved user ID %d, want %d", authed.ID, user.ID)
	}
	sessions, err := forum.ListAuthSessions(context.Background(), user.ID, session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].Current {
		t.Fatalf("current session was not listed correctly: %#v", sessions)
	}
	if err := forum.RevokeAuthSessionByID(context.Background(), user.ID, sessions[0].ID); err != nil {
		t.Fatal(err)
	}
	sessions, err = forum.ListAuthSessions(context.Background(), user.ID, session.Token)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].RevokedAt == nil || sessions[0].Current {
		t.Fatalf("revoked session was not listed correctly: %#v", sessions)
	}
	if _, err := forum.UserFromToken(context.Background(), session.Token); err == nil {
		t.Fatal("revoked session token still authenticated")
	}

	secondSession, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "correct-password"})
	if err != nil {
		t.Fatal(err)
	}
	if secondSession.Token == session.Token {
		t.Fatal("new login reused a revoked session token")
	}
	if err := forum.Logout(context.Background(), session.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := forum.UserFromToken(context.Background(), secondSession.Token); err != nil {
		t.Fatalf("second session should remain authenticated: %v", err)
	}
}

func TestResetPasswordRevokesExistingSessionsAndAllowsNewPassword(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	repository := sqliterepo.NewForumRepository(db)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("old-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: "reset-session@example.com", Nickname: "重置用户", Role: "student", Province: "广东", Grade: "高一",
	}, string(passwordHash))
	if err != nil {
		t.Fatal(err)
	}

	forum := NewForumService(repository, config.Config{JWTSecret: "test-secret"}, nil)
	session, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "old-password"})
	if err != nil {
		t.Fatal(err)
	}
	codeHash := hashVerificationCode(user.Email, "123456")
	if err := repository.CreateEmailVerificationCode(context.Background(), user.Email, codeHash, time.Now().Add(10*time.Minute)); err != nil {
		t.Fatal(err)
	}

	err = forum.ResetPassword(context.Background(), domain.ResetPasswordInput{
		Email: user.Email, VerificationCode: "123456", Password: "new-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := forum.UserFromToken(context.Background(), session.Token); err == nil {
		t.Fatal("old session token still authenticated after password reset")
	}
	if _, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "old-password"}); err == nil {
		t.Fatal("old password still works after reset")
	}
	if _, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "new-password"}); err != nil {
		t.Fatalf("new password should work after reset: %v", err)
	}
}

func TestDeleteAccountClearsCredentialsAndRevokesSessions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	db, err := storage.NewSQLiteDB(config.Config{
		SQLitePath:     filepath.Join(tempDir, "forum.db"),
		MediaUploadDir: filepath.Join(tempDir, "uploads"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	})

	repository := sqliterepo.NewForumRepository(db)
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("delete-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatal(err)
	}
	user, err := repository.CreateUser(context.Background(), domain.RegisterInput{
		Email: "delete-account@example.com", Nickname: "注销用户", Role: "student", Province: "广东", Grade: "高一",
	}, string(passwordHash))
	if err != nil {
		t.Fatal(err)
	}

	forum := NewForumService(repository, config.Config{JWTSecret: "test-secret"}, nil)
	session, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "delete-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err := forum.DeleteAccount(context.Background(), user.ID, domain.DeleteAccountInput{Password: "wrong-password"}); err == nil {
		t.Fatal("delete account accepted a wrong password")
	}
	if err := forum.DeleteAccount(context.Background(), user.ID, domain.DeleteAccountInput{Password: "delete-password"}); err != nil {
		t.Fatal(err)
	}
	if _, err := forum.UserFromToken(context.Background(), session.Token); err == nil {
		t.Fatal("deleted account session still authenticated")
	}
	if _, err := forum.Login(context.Background(), domain.LoginInput{Email: user.Email, Password: "delete-password"}); err == nil {
		t.Fatal("deleted account can still login")
	}
	var email, password sql.NullString
	if err := db.QueryRow(`SELECT email, password_hash FROM users WHERE id = ?`, user.ID).Scan(&email, &password); err != nil {
		t.Fatal(err)
	}
	if email.Valid || password.Valid {
		t.Fatalf("deleted account retained credentials: email=%v password=%v", email.Valid, password.Valid)
	}
}
