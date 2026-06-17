package ipmi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Client struct {
	Host     string
	User     string
	Password string
	Timeout  time.Duration
	Retries  int
}

func NewClient(host, user, password string, timeout time.Duration, retries int) *Client {
	return &Client{
		Host:     host,
		User:     user,
		Password: password,
		Timeout:  timeout,
		Retries:  retries,
	}
}

func (c *Client) Execute(ctx context.Context, command string, args ...string) (string, error) {
	baseArgs := []string{
		"-H", c.Host,
		"-U", c.User,
		"-P", c.Password,
		"-I", "lanplus",
	}
	allowedCommands := map[string]bool{
		"chassis status":     true,
		"mc info":            true,
		"lan print":          true,
		"sdr elist full":     true,
		"dcmi power reading": true,
		"sel list":           true,
	}
	if !allowedCommands[command] {
		return "", fmt.Errorf("unsupported command: %s", command)
	}
	var output []byte
	var err error

	for i := 0; i <= c.Retries; i++ {
		ctx, cancel := context.WithTimeout(ctx, c.Timeout)
		defer cancel()

		cmdArgs := append(baseArgs, strings.Fields(command)...)
		cmdArgs = append(cmdArgs, args...)

		cmd := exec.CommandContext(ctx, "ipmitool", cmdArgs...) // #nosec G204 - cmd is hardcoded and trusted
		output, err = cmd.CombinedOutput()

		if err == nil {
			return string(output), nil
		}

		if ctx.Err() == context.DeadlineExceeded {
			time.Sleep(1 * time.Second)
		}
	}

	return "", fmt.Errorf("command failed after %d retries: %v\nOutput: %s",
		c.Retries, err, string(output))
}

// 专用命令封装
func (c *Client) GetSensorData(ctx context.Context) (string, error) {
	return c.Execute(ctx, "sdr elist full")
}

func (c *Client) GetPowerMetrics(ctx context.Context) (string, error) {
	return c.Execute(ctx, "dcmi power reading")
}

func (c *Client) GetBMCInfo(ctx context.Context) (string, error) {
	return c.Execute(ctx, "mc info")
}

func (c *Client) GetNetworkStatus(ctx context.Context) (string, error) {
	return c.Execute(ctx, "lan print")
}

func (c *Client) GetChassisStatus(ctx context.Context) (string, error) {
	return c.Execute(ctx, "chassis status")
}

func (c *Client) GetSELList(ctx context.Context) (string, error) {
	// 创建带独立超时的上下文
	selCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return c.Execute(selCtx, "sel list")
}

// ...其他专用命令方法
