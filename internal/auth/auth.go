package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"homebooks/internal/logger"
)

const (
	SessionCookieName = "homebooks_session"
	SessionDuration   = 30 * 24 * time.Hour // 30 days
)

type Auth struct {
	db       *sql.DB
	password string
}

func New(db *sql.DB) *Auth {
	password := os.Getenv("HOMEBOOKS_PASSWORD")
	if password == "" {
		password = "changeme" // Default for development
	}
	return &Auth{db: db, password: password}
}

// CheckPassword verifies the provided password
func (a *Auth) CheckPassword(ctx context.Context, password string) bool {
	success := password == a.password
	l := logger.FromContext(ctx)

	if success {
		l.Info("auth_login_success")
	} else {
		l.Warn("auth_login_failed", "reason", "invalid_password")
	}
	return success
}

// CreateSession creates a new session and returns the token
func (a *Auth) CreateSession(ctx context.Context) (string, error) {
	l := logger.FromContext(ctx)

	token, err := generateToken()
	if err != nil {
		l.Error("auth_session_create_error", "error", err.Error())
		return "", err
	}

	expiresAt := time.Now().Add(SessionDuration)
	_, err = a.db.Exec(`
		INSERT INTO sessions (token, expires_at) VALUES (?, ?)
	`, token, expiresAt)
	if err != nil {
		l.Error("auth_session_create_error", "error", err.Error())
		return "", fmt.Errorf("create session: %w", err)
	}

	l.Info("auth_session_created", "expires_at", expiresAt.Format(time.RFC3339))
	return token, nil
}

// ValidateSession checks if the token is valid and not expired
func (a *Auth) ValidateSession(ctx context.Context, token string) bool {
	l := logger.FromContext(ctx)

	var expiresAt time.Time
	err := a.db.QueryRow(`
		SELECT expires_at FROM sessions WHERE token = ?
	`, token).Scan(&expiresAt)
	if err != nil {
		l.Debug("auth_session_invalid", "reason", "not_found")
		return false
	}

	if time.Now().After(expiresAt) {
		l.Debug("auth_session_invalid", "reason", "expired")
		return false
	}
	return true
}

// DeleteSession removes a session
func (a *Auth) DeleteSession(ctx context.Context, token string) error {
	l := logger.FromContext(ctx)

	_, err := a.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		l.Error("auth_session_delete_error", "error", err.Error())
		return err
	}
	l.Info("auth_logout")
	return nil
}

// CleanExpiredSessions removes expired sessions
func (a *Auth) CleanExpiredSessions() error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE expires_at < datetime('now')`)
	return err
}

// SetSessionCookie sets the session cookie on the response
func (a *Auth) SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie removes the session cookie
func (a *Auth) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// GetSessionFromRequest retrieves the session token from the request cookie
func (a *Auth) GetSessionFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// Middleware checks for valid session, redirects to login if not authenticated
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := logger.FromContext(ctx)

		// Allow access to login page and static files
		if r.URL.Path == "/login" || strings.HasPrefix(r.URL.Path, "/static") {
			next.ServeHTTP(w, r)
			return
		}

		token := a.GetSessionFromRequest(r)
		if token == "" {
			l.Debug("auth_no_session", "path", r.URL.Path)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if !a.ValidateSession(ctx, token) {
			l.Debug("auth_redirect_to_login", "path", r.URL.Path)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
