package server

import (
	"fmt"
	"net/http"
)

// Request represents an HTTP request
type Request struct {
	W        http.ResponseWriter
	R        *http.Request
	handlers []HandlerFunc
	Error    error
}

// HandlerFunc is a function that processes a request
type HandlerFunc func(*Request)

// NewRequest creates a new request

// TODO: implement functions
