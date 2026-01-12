package network

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析网络工具输出的接口
type Parser interface {
	Parse(networkOutput []byte) (*Status, error)
}

// Status 包含所有解析的网络状态信息
type Status struct {
	Networks  []Network
	Timestamp time.Time
}

// Network 表示一个网络实例
type Network struct {
	// 基本信息
	ID        string
	Name      string
	Driver    string
	Interface string
	Labels    string

	// 其他信息
	Created     time.Time
	IPv6Enabled bool
	Internal    bool
	DNSEnabled  bool
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (p *defaultParser) Parse(networkOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 尝试解析JSON格式输出（如podman network ls --format json）
	if err := p.parseJSONOutput(networkOutput, status); err == nil {
		return status, nil
	}

	// 如果JSON解析失败，尝试解析文本格式
	if err := p.parseTextOutput(networkOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse network output")
	}

	return status, nil
}

// parseJSONOutput 解析JSON格式的网络输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var networks []map[string]interface{}

	if err := json.Unmarshal(output, &networks); err != nil {
		return err
	}

	for _, networkData := range networks {
		network := Network{}

		// 解析基本信息
		if id, ok := networkData["id"].(string); ok {
			network.ID = id
		}
		if name, ok := networkData["name"].(string); ok {
			network.Name = name
		}
		if driver, ok := networkData["driver"].(string); ok {
			network.Driver = driver
		}
		if networkInterface, ok := networkData["network_interface"].(string); ok {
			// 移除 "cni-" 前缀，保持与指标示例一致
			network.Interface = strings.TrimPrefix(networkInterface, "cni-")
		}

		// 解析标签信息 - 这里暂时设为空字符串，与示例指标一致
		network.Labels = ""

		// 解析布尔类型字段
		if ipv6Enabled, ok := networkData["ipv6_enabled"].(bool); ok {
			network.IPv6Enabled = ipv6Enabled
		}
		if internal, ok := networkData["internal"].(bool); ok {
			network.Internal = internal
		}
		if dnsEnabled, ok := networkData["dns_enabled"].(bool); ok {
			network.DNSEnabled = dnsEnabled
		}

		// 解析创建时间
		if created, ok := networkData["created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				network.Created = t
			}
		}

		status.Networks = append(status.Networks, network)
	}

	return nil
}

// parseTextOutput 解析文本格式的网络输出
func (p *defaultParser) parseTextOutput(output []byte, status *Status) error {
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NETWORK ID") {
			continue
		}

		network, err := p.parseNetworkLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}

		status.Networks = append(status.Networks, network)
	}

	return nil
}

// parseNetworkLine 解析单行网络信息
func (p *defaultParser) parseNetworkLine(line string) (Network, error) {
	var network Network
	fields := strings.Fields(line)

	if len(fields) < 3 {
		return network, errors.New("insufficient fields in network line")
	}

	// 基本格式: NETWORK_ID NAME DRIVER
	network.ID = fields[0]
	network.Name = fields[1]
	network.Driver = fields[2]
	network.Labels = ""

	return network, nil
}
