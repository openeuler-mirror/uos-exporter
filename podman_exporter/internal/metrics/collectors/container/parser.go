package container

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析容器工具输出的接口
type Parser interface {
	Parse(containerOutput []byte) (*Status, error)
	ParseStats(statsOutput []byte, status *Status) error
}

// Status 包含所有解析的容器状态信息
type Status struct {
	Containers []Container
	Timestamp  time.Time
}

// Container 表示一个容器实例
type Container struct {
	// 基本信息
	ID      string
	Name    string
	Image   string
	Status  string
	Running bool
	PodID   string
	PodName string
	Ports   string

	// 状态信息
	State    int // -1=unknown,0=created,1=initialized,2=running,3=stopped,4=paused,5=exited,6=removing,7=stopping
	Health   int // -1=unknown,0=healthy,1=unhealthy,2=starting
	ExitCode int
	PIDs     float64

	// 时间信息
	Created       time.Time
	Started       time.Time
	Exited        time.Time
	UptimeSeconds int64

	// 资源使用
	MemoryUsage      int64   // 字节
	MemoryLimit      int64   // 字节
	CPUPercent       float64 // 百分比
	CPUSeconds       float64 // CPU 时间（秒）
	CPUSystemSeconds float64 // 系统 CPU 时间（秒）

	// 存储信息
	RootfsSize int64 // 根文件系统大小（字节）
	RwSize     int64 // 读写层大小（字节）

	// 块设备 I/O
	BlockInput  int64 // 块输入（字节）
	BlockOutput int64 // 块输出（字节）

	// 网络统计
	NetInputBytes    int64 // 网络输入字节
	NetOutputBytes   int64 // 网络输出字节
	NetInputPackets  int64 // 网络输入包数
	NetOutputPackets int64 // 网络输出包数
	NetInputDropped  int64 // 网络输入丢包
	NetOutputDropped int64 // 网络输出丢包
	NetInputErrors   int64 // 网络输入错误
	NetOutputErrors  int64 // 网络输出错误
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{
		patterns: map[string]*regexp.Regexp{
			"containerID":   regexp.MustCompile(`(?m)^([a-f0-9]{12})`),
			"containerName": regexp.MustCompile(`(?m)^\S+\s+(\S+)`),
			"status":        regexp.MustCompile(`(?m)(Up|Exited|Created|Running|Stopped)`),
		},
	}
}

type defaultParser struct {
	patterns map[string]*regexp.Regexp
}

func (p *defaultParser) Parse(containerOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 尝试解析JSON格式输出（如podman ps --format json）
	if err := p.parseJSONOutput(containerOutput, status); err == nil {
		return status, nil
	}

	// 如果JSON解析失败，尝试解析文本格式
	if err := p.parseTextOutput(containerOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse container output")
	}

	return status, nil
}

// ParseStats 解析容器统计信息
func (p *defaultParser) ParseStats(statsOutput []byte, status *Status) error {
	var statsData []map[string]interface{}

	if err := json.Unmarshal(statsOutput, &statsData); err != nil {
		return errors.Wrap(err, "failed to parse stats JSON")
	}

	// 创建 ID 到容器的映射，使用短 ID 进行匹配
	containerMap := make(map[string]*Container)
	for i := range status.Containers {
		shortID := status.Containers[i].ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		containerMap[shortID] = &status.Containers[i]
	}

	// 更新统计信息
	for _, stats := range statsData {
		id, ok := stats["id"].(string)
		if !ok {
			continue
		}

		container, exists := containerMap[id]
		if !exists {
			continue
		}

		// 解析进程数
		if pidsStr, ok := stats["pids"].(string); ok {
			if pids, err := strconv.ParseFloat(pidsStr, 64); err == nil {
				container.PIDs = pids
			}
		}

		// 解析 CPU 时间（从字符串格式如 "1.23s"）
		if cpuTimeStr, ok := stats["cpu_time"].(string); ok {
			if cpuTime, err := time.ParseDuration(cpuTimeStr); err == nil {
				container.CPUSeconds = cpuTime.Seconds()
			}
		}

		// 解析内存使用（从字符串格式如 "100MB / 2GB"）
		if memUsageStr, ok := stats["mem_usage"].(string); ok {
			parts := strings.Split(memUsageStr, " / ")
			if len(parts) == 2 {
				if usage, err := p.parseSizeToBytes(parts[0]); err == nil {
					container.MemoryUsage = usage
				}
				if limit, err := p.parseSizeToBytes(parts[1]); err == nil {
					container.MemoryLimit = limit
				}
			}
		}

		// 解析网络 I/O（从字符串格式如 "1.2kB / 500B"）
		if netIOStr, ok := stats["net_io"].(string); ok {
			parts := strings.Split(netIOStr, " / ")
			if len(parts) == 2 {
				if input, err := p.parseSizeToBytes(parts[0]); err == nil {
					container.NetInputBytes = input
				}
				if output, err := p.parseSizeToBytes(parts[1]); err == nil {
					container.NetOutputBytes = output
				}
			}
		}

		// 解析块 I/O（从字符串格式如 "1.2MB / 800kB"）
		if blockIOStr, ok := stats["block_io"].(string); ok {
			parts := strings.Split(blockIOStr, " / ")
			if len(parts) == 2 {
				if input, err := p.parseSizeToBytes(parts[0]); err == nil {
					container.BlockInput = input
				}
				if output, err := p.parseSizeToBytes(parts[1]); err == nil {
					container.BlockOutput = output
				}
			}
		}
	}

	return nil
}

// parseJSONOutput 解析JSON格式的容器输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var containers []map[string]interface{}

	if err := json.Unmarshal(output, &containers); err != nil {
		return err
	}

	for _, containerData := range containers {
		container := Container{}

		if id, ok := containerData["Id"].(string); ok {
			container.ID = id
		}
		if names, ok := containerData["Names"].([]interface{}); ok && len(names) > 0 {
			if name, ok := names[0].(string); ok {
				container.Name = name
			}
		}
		if image, ok := containerData["Image"].(string); ok {
			container.Image = image
		}
		if state, ok := containerData["State"].(string); ok {
			container.Status = state
			container.Running = strings.ToLower(state) == "running"
			container.State = p.parseStateToInt(state)
		}

		// 解析 Pod 信息（注意字段名）
		if pod, ok := containerData["Pod"].(string); ok {
			container.PodID = pod
		}
		if podName, ok := containerData["PodName"].(string); ok {
			container.PodName = podName
		}

		// 解析端口信息（处理对象数组格式）
		if portsData, ok := containerData["Ports"].([]interface{}); ok {
			var portStrs []string
			for _, portInterface := range portsData {
				if port, ok := portInterface.(map[string]interface{}); ok {
					hostPort, hasHostPort := port["host_port"]
					containerPort, hasContainerPort := port["container_port"]
					protocol, hasProtocol := port["protocol"]
					hostIP, hasHostIP := port["host_ip"]

					if hasHostPort && hasContainerPort && hasProtocol {
						var portStr string
						if hasHostIP && hostIP != "" {
							portStr = fmt.Sprintf("%v:%v->%v/%v", hostIP, hostPort, containerPort, protocol)
						} else {
							portStr = fmt.Sprintf("0.0.0.0:%v->%v/%v", hostPort, containerPort, protocol)
						}
						portStrs = append(portStrs, portStr)
					}
				}
			}
			container.Ports = strings.Join(portStrs, ",")
		}

		// 解析时间字段（处理 Unix 时间戳）
		if created, ok := containerData["Created"].(float64); ok {
			container.Created = time.Unix(int64(created), 0)
			if container.Running {
				container.UptimeSeconds = int64(time.Since(container.Created).Seconds())
			}
		}

		if startedAt, ok := containerData["StartedAt"].(float64); ok {
			container.Started = time.Unix(int64(startedAt), 0)
		}

		if exitedAt, ok := containerData["ExitedAt"].(float64); ok {
			container.Exited = time.Unix(int64(exitedAt), 0)
		}

		// 解析退出码
		if exitCode, ok := containerData["ExitCode"].(float64); ok {
			container.ExitCode = int(exitCode)
		}

		status.Containers = append(status.Containers, container)
	}

	return nil
}

// parseStateToInt 将状态字符串转换为整数
func (p *defaultParser) parseStateToInt(state string) int {
	switch strings.ToLower(state) {
	case "created":
		return 0
	case "initialized":
		return 1
	case "running":
		return 2
	case "stopped":
		return 3
	case "paused":
		return 4
	case "exited":
		return 5
	case "removing":
		return 6
	case "stopping":
		return 7
	default:
		return -1 // unknown
	}
}

// parseTextOutput 解析文本格式的容器输出
func (p *defaultParser) parseTextOutput(output []byte, status *Status) error {
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "CONTAINER") {
			continue
		}

		container, err := p.parseContainerLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}

		status.Containers = append(status.Containers, container)
	}

	return nil
}

// parseContainerLine 解析单行容器信息
func (p *defaultParser) parseContainerLine(line string) (Container, error) {
	var container Container
	fields := strings.Fields(line)

	if len(fields) < 3 {
		return container, errors.New("insufficient fields in container line")
	}

	container.ID = fields[0]
	container.Image = fields[1]

	// 查找状态信息
	for _, field := range fields {
		if p.patterns["status"].MatchString(field) {
			container.Status = field
			container.Running = strings.Contains(strings.ToLower(field), "up") ||
				strings.Contains(strings.ToLower(field), "running")
			container.State = p.parseStateToInt(field)
			break
		}
	}

	// 尝试从字段中提取容器名称
	for i, field := range fields {
		if i > 2 && !p.patterns["status"].MatchString(field) {
			container.Name = field
			break
		}
	}

	return container, nil
}

// Helper 方法：从正则表达式匹配中提取命名组
func (p *defaultParser) extractNamedGroups(re *regexp.Regexp, match [][]byte) map[string]string {
	groups := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			groups[name] = string(match[i])
		}
	}
	return groups
}

// Helper 方法：解析大小字符串（如 "1.2GB", "0B"）为字节数
func (p *defaultParser) parseSizeToBytes(sizeStr string) (int64, error) {
	if sizeStr == "" || sizeStr == "0B" {
		return 0, nil
	}

	// 移除空格并转为小写
	sizeStr = strings.ToLower(strings.TrimSpace(sizeStr))

	// 提取数字部分和单位
	var numStr string
	var unit string

	for i, r := range sizeStr {
		if r >= '0' && r <= '9' || r == '.' {
			numStr += string(r)
		} else {
			unit = sizeStr[i:]
			break
		}
	}

	if numStr == "" {
		return 0, errors.New("no numeric value found")
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse size number: %s", numStr)
	}

	// 转换单位（支持更多格式）
	switch unit {
	case "b", "":
		return int64(num), nil
	case "kb":
		return int64(num * 1000), nil
	case "mb":
		return int64(num * 1000 * 1000), nil
	case "gb":
		return int64(num * 1000 * 1000 * 1000), nil
	case "tb":
		return int64(num * 1000 * 1000 * 1000 * 1000), nil
	case "k", "kib":
		return int64(num * 1024), nil
	case "m", "mib":
		return int64(num * 1024 * 1024), nil
	case "g", "gib":
		return int64(num * 1024 * 1024 * 1024), nil
	case "t", "tib":
		return int64(num * 1024 * 1024 * 1024 * 1024), nil
	default:
		return int64(num), nil
	}
}
// Part 2 commit for podman_exporter/internal/metrics/collectors/container/parser.go
