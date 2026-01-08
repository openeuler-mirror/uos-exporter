package clock

import (
	"time"
)

// Clock 接口定义了时间相关的操作
type Clock interface {
	// Now 返回当前时间
	Now() time.Time
	// Since 返回从指定时间到现在的持续时间
	Since(t time.Time) time.Duration
}

// SystemClock 是Clock接口的系统实现
type SystemClock struct{}

// Now 返回当前系统时间
func (c *SystemClock) Now() time.Time {
	return time.Now()
}

// Since 返回从指定时间到现在的持续时间
func (c *SystemClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}
// Part 2 commit for podman_exporter/internal/clock/clock.go
