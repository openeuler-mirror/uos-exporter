package ipmi

import (
	"context"
	"errors"
	"os/exec"
	"testing"
	"time"
)

var execCommand = exec.CommandContext

// 测试专用包装结构体
type testClient struct {
	*Client
	mockExecutor func(ctx context.Context, cmd string, args ...string) (string, error)
}

// 重写Execute方法实现mock
func (tc *testClient) Execute(ctx context.Context, command string, args ...string) (string, error) {
	if tc.mockExecutor != nil {
		return tc.mockExecutor(ctx, command, args...)
	}
	return tc.Client.Execute(ctx, command, args...)
}


// TODO: implement functions
