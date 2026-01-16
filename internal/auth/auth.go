package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"
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
func (a *Auth) CheckPassword(password string) bool {
	return password == a.password
}

// CreateSession creates a new session and returns the token
func (a *Auth) CreateSession() (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(SessionDuration)
	_, err = a.db.Exec(`
		INSERT INTO sessions (token, expires_at) VALUES (?, ?)
	`, token, expiresAt)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

// ValidateSession checks if the token is valid and not expired
func (a *Auth) ValidateSession(token string) bool {
	var expiresAt time.Time
	err := a.db.QueryRow(`
		SELECT expires_at FROM sessions WHERE token = ?
	`, token).Scan(&expiresAt)
	if err != nil {
		return false
	}
	return time.Now().Before(expiresAt)
}

// DeleteSession removes a session
func (a *Auth) DeleteSession(token string) error {
	_, err := a.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
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
		// Allow access to login page and static files
		if r.URL.Path == "/login" || r.URL.Path == "/static/style.css" {
			next.ServeHTTP(w, r)
			return
		}

		token := a.GetSessionFromRequest(r)
		if token == "" || !a.ValidateSession(token) {
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
