package server

import (
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
	_, _ = r.ResponseWriter.Write([]byte(r.Error.Error()))
}
// Part 2 commit for tc_exporter/internal/server/request.go
