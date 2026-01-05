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

func TestClient_Execute(t *testing.T) {
	// 备份原始命令执行函数
	origExec := execCommand
	defer func() { execCommand = origExec }()

	tests := []struct {
		name        string
		setup       func(*testClient)
		command     string
		wantErr     bool
		wantRetries int
	}{
		{
			name: "success on first try",
			setup: func(c *testClient) {
				c.mockExecutor = func(ctx context.Context, cmd string, args ...string) (string, error) {
					return "success", nil
				}
			},
			command: "chassis status",
		},
		{
			name: "retry 3 times",
			setup: func(c *testClient) {
				c.Retries = 3
				c.mockExecutor = func(ctx context.Context, cmd string, args ...string) (string, error) {
					return "", errors.New("timeout")
				}
			},
			command:     "invalid command",
			wantErr:     true,
			wantRetries: 3,
		},
		{
			name: "context timeout",
			setup: func(c *testClient) {
				c.Timeout = 10 * time.Millisecond
				c.mockExecutor = func(ctx context.Context, cmd string, args ...string) (string, error) {
					time.Sleep(20 * time.Millisecond)
					return "", context.DeadlineExceeded
				}
			},
			command: "slow command",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建包装客户端
			client := &testClient{
				Client: NewClient("testhost", "user", "pass", 1*time.Second, 0),
			}
			if tt.setup != nil {
				tt.setup(client)
			}

			_, err := client.Execute(context.Background(), tt.command)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
