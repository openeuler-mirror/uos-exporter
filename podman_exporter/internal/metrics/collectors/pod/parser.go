package pod

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析Pod工具输出的接口
type Parser interface {
	Parse(podOutput []byte) (*Status, error)
}

// Status 包含所有解析的Pod状态信息
type Status struct {
	Pods      []Pod
	Timestamp time.Time
}

// Pod 表示一个Pod实例
type Pod struct {
	// 基本信息
	ID      string
	Name    string
	InfraID string

	// 状态信息
	State      string // Created, Running, Stopped, etc.
	StateValue int    // 状态对应的数值

	// 容器信息
	Containers int // Pod中容器的数量

	// 时间信息
	Created time.Time
}

// Pod状态映射
var podStateMap = map[string]int{
	"unknown":  -1,
	"created":  0,
	"error":    1,
	"exited":   2,
	"paused":   3,
	"running":  4,
	"degraded": 5,
	"stopped":  6,
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{
		patterns: map[string]*regexp.Regexp{
			"podID": regexp.MustCompile(`(?m)^([a-f0-9]{12})`),
		},
	}
}

type defaultParser struct {
	patterns map[string]*regexp.Regexp
}

func (p *defaultParser) Parse(podOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 尝试解析JSON格式输出（如podman pod ls --format json）
	if err := p.parseJSONOutput(podOutput, status); err == nil {
		return status, nil
	}

	// 如果JSON解析失败，尝试解析文本格式
	if err := p.parseTextOutput(podOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse pod output")
	}

	return status, nil
}

// parseJSONOutput 解析JSON格式的Pod输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var pods []map[string]interface{}

	if err := json.Unmarshal(output, &pods); err != nil {
		return err
	}

	for _, podData := range pods {
		pod := Pod{}

		// 解析基本信息
		if id, ok := podData["Id"].(string); ok {
			pod.ID = id
		}
		if name, ok := podData["Name"].(string); ok {
			pod.Name = name
		}
		if infraId, ok := podData["InfraId"].(string); ok {
			pod.InfraID = infraId
		}

		// 解析状态信息
		if state, ok := podData["Status"].(string); ok {
			pod.State = strings.ToLower(state)
			if stateValue, exists := podStateMap[pod.State]; exists {
				pod.StateValue = stateValue
			} else {
				pod.StateValue = -1 // unknown
			}
		}

		// 解析容器数量 - 从Containers数组的长度获取
		if containers, ok := podData["Containers"].([]interface{}); ok {
			pod.Containers = len(containers)
		} else if containerCount, ok := podData["NumberOfContainers"].(float64); ok {
			// 备用：如果有NumberOfContainers字段也尝试解析
			pod.Containers = int(containerCount)
		}

		// 解析创建时间
		if created, ok := podData["Created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				pod.Created = t
			}
		} else if created, ok := podData["Created"].(float64); ok {
			pod.Created = time.Unix(int64(created), 0)
		}

		status.Pods = append(status.Pods, pod)
	}

	return nil
}

// parseTextOutput 解析文本格式的Pod输出
func (p *defaultParser) parseTextOutput(output []byte, status *Status) error {
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "POD ID") {
			continue
		}

		pod, err := p.parsePodLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}

		status.Pods = append(status.Pods, pod)
	}

	return nil
}

// parsePodLine 解析单行Pod信息
func (p *defaultParser) parsePodLine(line string) (Pod, error) {
	var pod Pod
	fields := strings.Fields(line)

	if len(fields) < 3 {
		return pod, errors.New("insufficient fields in pod line")
	}

	// 基本格式: POD_ID NAME STATUS CREATED ...
	pod.ID = fields[0]
	pod.Name = fields[1]

	if len(fields) > 2 {
		pod.State = strings.ToLower(fields[2])
		if stateValue, exists := podStateMap[pod.State]; exists {
			pod.StateValue = stateValue
		} else {
			pod.StateValue = -1 // unknown
		}
	}

	return pod, nil
}

// getPodStateValue 根据状态字符串获取对应的数值
func (p *defaultParser) getPodStateValue(state string) int {
	state = strings.ToLower(strings.TrimSpace(state))
	if value, exists := podStateMap[state]; exists {
		return value
	}
	return -1 // unknown
}
