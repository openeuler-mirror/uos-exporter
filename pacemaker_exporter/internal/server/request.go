package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Request wraps HTTP request and response with error handling
type Request struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Error          error
	handlers       []HandlerFunc
	startTime      time.Time
}

// HandlerFunc defines the signature for middleware handlers
type HandlerFunc func(ctx *Request)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error     string    `json:"error"`
	Message   string    `json:"message"`
	Status    int       `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
}

// NewRequest creates a new Request instance with initialization
func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	return &Request{
		Request:        r,
		ResponseWriter: w,
		startTime:      time.Now(),
	}
}

// Fail responds with an error status and structured error message
func (r *Request) Fail(status int) {
	if r.Error == nil {
		r.Error = fmt.Errorf("request failed with status %d", status)
	}

	errorResp := ErrorResponse{
		Error:     http.StatusText(status),
		Message:   r.Error.Error(),
		Status:    status,
		Timestamp: time.Now().UTC(),
		Path:      r.Request.URL.Path,
	}

	// Log the error with request context
	logrus.WithFields(logrus.Fields{
		"method":      r.Request.Method,
		"path":        r.Request.URL.Path,
		"remote_ip":   r.Request.RemoteAddr,
		"user_agent":  r.Request.Header.Get("User-Agent"),
		"status":      status,
		"error":       r.Error.Error(),
		"duration_ms": time.Since(r.startTime).Milliseconds(),
	}).Error("Request failed")

	// Set response headers
	r.ResponseWriter.Header().Set("Content-Type", "application/json")
	r.ResponseWriter.WriteHeader(status)

	// Try to send JSON error response
	if err := json.NewEncoder(r.ResponseWriter).Encode(errorResp); err != nil {
		logrus.WithError(err).Error("Failed to encode error response")
		// Fallback to plain text
		r.ResponseWriter.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(r.ResponseWriter, "Error: %s", r.Error.Error())
	}
}
