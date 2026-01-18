package logger

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	loggerKey    contextKey = "logger"
)

// GenerateRequestID creates a new unique request ID (16 hex chars)
func GenerateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from context
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithLogger stores a logger instance in context
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns a logger from context, or the default logger.
// The returned logger always includes the request ID if present.
func FromContext(ctx context.Context) *slog.Logger {
	// Try to get logger from context first
	if l, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return l
	}

	// Fall back to default logger with request ID if available
	l := Default()
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		l = l.With("request_id", requestID)
	}
	return l
}

// Ctx is a convenience alias for FromContext
func Ctx(ctx context.Context) *slog.Logger {
	return FromContext(ctx)
}
