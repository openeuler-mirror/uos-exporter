package volume

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析存储卷工具输出的接口
type Parser interface {
	Parse(volumeOutput []byte) (*Status, error)
}

// Status 包含所有解析的存储卷状态信息
type Status struct {
	Volumes   []Volume
	Timestamp time.Time
}

// Volume 表示一个存储卷实例
type Volume struct {
	// 基本信息
	Name       string
	Driver     string
	MountPoint string

	// 时间信息
	Created time.Time

	// 其他信息
	Scope       string
	Anonymous   bool
	MountCount  int
	NeedsCopyUp bool
	LockNumber  int
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (p *defaultParser) Parse(volumeOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 尝试解析JSON格式输出（如podman volume ls --format json）
	if err := p.parseJSONOutput(volumeOutput, status); err == nil {
		return status, nil
	}

	// 如果JSON解析失败，尝试解析文本格式
	if err := p.parseTextOutput(volumeOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse volume output")
	}

	return status, nil
}

// parseJSONOutput 解析JSON格式的存储卷输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var volumes []map[string]interface{}

	if err := json.Unmarshal(output, &volumes); err != nil {
		return err
	}

	for _, volumeData := range volumes {
		volume := Volume{}

		// 解析基本信息
		if name, ok := volumeData["Name"].(string); ok {
			volume.Name = name
		}
		if driver, ok := volumeData["Driver"].(string); ok {
			volume.Driver = driver
		}
		if mountPoint, ok := volumeData["Mountpoint"].(string); ok {
			volume.MountPoint = mountPoint
		}

		// 解析其他信息
		if scope, ok := volumeData["Scope"].(string); ok {
			volume.Scope = scope
		}
		if anonymous, ok := volumeData["Anonymous"].(bool); ok {
			volume.Anonymous = anonymous
		}
		if mountCount, ok := volumeData["MountCount"].(float64); ok {
			volume.MountCount = int(mountCount)
		}
		if needsCopyUp, ok := volumeData["NeedsCopyUp"].(bool); ok {
			volume.NeedsCopyUp = needsCopyUp
		}
		if lockNumber, ok := volumeData["LockNumber"].(float64); ok {
			volume.LockNumber = int(lockNumber)
		}

		// 解析创建时间
		if createdAt, ok := volumeData["CreatedAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				volume.Created = t
			}
		}

		status.Volumes = append(status.Volumes, volume)
	}

	return nil
}

// parseTextOutput 解析文本格式的存储卷输出
func (p *defaultParser) parseTextOutput(output []byte, status *Status) error {
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "DRIVER") {
			continue
		}

		volume, err := p.parseVolumeLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}

		status.Volumes = append(status.Volumes, volume)
	}

	return nil
}

// parseVolumeLine 解析单行存储卷信息
func (p *defaultParser) parseVolumeLine(line string) (Volume, error) {
	var volume Volume
	fields := strings.Fields(line)

	if len(fields) < 2 {
		return volume, errors.New("insufficient fields in volume line")
	}

	// 基本格式: DRIVER VOLUME_NAME
	volume.Driver = fields[0]
	volume.Name = fields[1]

	return volume, nil
}
// Part 2 commit for podman_exporter/internal/metrics/collectors/volume/parser.go
