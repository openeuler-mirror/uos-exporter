package server

import (
	"fmt"
	"net/http"
)

type Request struct {
	Request        *http.Request
	ResponseWriter http.ResponseWriter
	Error          error
	handlers       []HandlerFunc
}

type HandlerFunc func(ctx *Request)

func NewRequest(w http.ResponseWriter, r *http.Request) *Request {
	return &Request{
		Request:        r,
		ResponseWriter: w,
	}
}

func (r *Request) Fail(status int) {
	r.ResponseWriter.Header().Set("Content-Type", "text/html")
	r.ResponseWriter.WriteHeader(status)
	if _, err := r.ResponseWriter.Write([]byte(r.Error.Error())); err != nil {
		fmt.Printf("Warning: Failed to write error response to client: %v", err)
	}
}
// Part 2 commit for mongodb_exporter/internal/server/request.go
