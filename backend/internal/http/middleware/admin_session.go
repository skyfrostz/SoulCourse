package middleware

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const AdminSessionCookieName = "scf_admin_session"
const AdminCSRFCookieName = "scf_admin_csrf"

type AdminSessionStore struct {
	mu       sync.Mutex
	sessions map[string]adminSession
	ttl      time.Duration
}

type adminSession struct {
	principal AdminPrincipal
	expiresAt time.Time
}

func NewAdminSessionStore(ttl time.Duration) *AdminSessionStore {
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	return &AdminSessionStore{sessions: make(map[string]adminSession), ttl: ttl}
}

// Issue accepts one principal. The empty call is retained temporarily for the
// existing single-admin login path until the handler is wired to pass identity.
func (s *AdminSessionStore) Issue(principals ...AdminPrincipal) (string, time.Time, error) {
	if len(principals) > 1 {
		return "", time.Time{}, ErrInvalidAdminPrincipal
	}
	principal := AdminPrincipal{Email: "legacy-admin", Role: AdminRoleSuperAdmin}
	if len(principals) == 1 {
		principal = principals[0]
	}
	normalized, err := normalizeAdminPrincipal(principal)
	if err != nil {
		return "", time.Time{}, err
	}
	token, err := randomURLToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(s.ttl).UTC()
	s.mu.Lock()
	s.sessions[hashAdminSessionToken(token)] = adminSession{principal: normalized, expiresAt: expiresAt}
	s.mu.Unlock()
	return token, expiresAt, nil
}

func (s *AdminSessionStore) Valid(token string) bool {
	_, ok := s.Resolve(token)
	return ok
}

func (s *AdminSessionStore) Resolve(token string) (AdminPrincipal, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AdminPrincipal{}, false
	}
	hash := hashAdminSessionToken(token)
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[hash]
	if !ok || now.After(session.expiresAt) {
		delete(s.sessions, hash)
		return AdminPrincipal{}, false
	}
	return cloneAdminPrincipal(session.principal), true
}

func (s *AdminSessionStore) Revoke(token string) {
	token = strings.TrimSpace(token)
	if token == "" {
		return
	}
	s.mu.Lock()
	delete(s.sessions, hashAdminSessionToken(token))
	s.mu.Unlock()
}

func SetAdminSessionCookie(c *gin.Context, token string, maxAgeSeconds int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		Secure:   secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAdminSessionCookie(c *gin.Context, secure bool) {
	SetAdminSessionCookie(c, "", -1, secure)
}

func SetAdminCSRFCookie(c *gin.Context, token string, maxAgeSeconds int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     AdminCSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   maxAgeSeconds,
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func ClearAdminCSRFCookie(c *gin.Context, secure bool) {
	SetAdminCSRFCookie(c, "", -1, secure)
}

func GenerateCSRFToken() (string, error) {
	return randomURLToken(32)
}

func randomURLToken(size int) (string, error) {
	token := make([]byte, size)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(token), nil
}

func hashAdminSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
