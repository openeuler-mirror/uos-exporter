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

// TODO: implement functions
