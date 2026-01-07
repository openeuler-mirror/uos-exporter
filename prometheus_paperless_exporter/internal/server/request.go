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


// TODO: implement functions
