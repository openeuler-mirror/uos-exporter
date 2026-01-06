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
func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	return &Request{
		W: w,
		R: r,
	}
}

// Fail writes a failure response
func (r *Request) Fail(status int) {
	r.W.Header().Set("Content-Type", "text/html")
	r.W.WriteHeader(status)
	if r.Error != nil {
		if _, err := r.W.Write([]byte(r.Error.Error())); err != nil {
			fmt.Printf("The write error response failed: %v\n", err)
		}
	} else {
		if _, err := r.W.Write([]byte(fmt.Sprintf("Error %d", status))); err != nil {
			fmt.Printf("The write error response failed: %v\n", err)
		}
	}
}