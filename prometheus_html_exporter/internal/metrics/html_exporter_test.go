package metrics

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/antchfx/htmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// 模拟HTTP客户端的响应
type mockReadCloser struct {
	io.Reader
}

func (m mockReadCloser) Close() error {
	return nil
}


// TODO: implement functions
