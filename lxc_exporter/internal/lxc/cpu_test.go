package lxc

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock 结构体
type MockLxc struct {
	existingContainers map[string]string // 模拟容器名到 cpu.stat 内容的映射
}

// Mock 方法：检查容器是否存在

// TODO: implement
