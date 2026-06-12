package system

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析系统工具输出的接口
type Parser interface {
	Parse(systemOutput []byte) (*Status, error)
}

// Status 包含所有解析的系统状态信息
type Status struct {
	APIVersion     string
	BuildahVersion string
	ConmonVersion  string
	RuntimeVersion string
	Timestamp      time.Time
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{}
}

type defaultParser struct{}

func (p *defaultParser) Parse(systemOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 解析JSON格式输出（podman system info --format json）
	if err := p.parseJSONOutput(systemOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse system output")
	}

	return status, nil
}

// parseJSONOutput 解析JSON格式的系统输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var systemData map[string]interface{}

	if err := json.Unmarshal(output, &systemData); err != nil {
		return err
	}

	// 解析版本信息
	if version, ok := systemData["version"].(map[string]interface{}); ok {
		if apiVersion, ok := version["APIVersion"].(string); ok {
			status.APIVersion = apiVersion
		}
	}

	// 解析主机信息
	if host, ok := systemData["host"].(map[string]interface{}); ok {
		// Buildah版本
		if buildahVersion, ok := host["buildahVersion"].(string); ok {
			status.BuildahVersion = buildahVersion
		}

		// Conmon版本
		if conmon, ok := host["conmon"].(map[string]interface{}); ok {
			if conmonVersion, ok := conmon["version"].(string); ok {
				status.ConmonVersion = conmonVersion
			}
		}

		// OCI运行时版本
		if ociRuntime, ok := host["ociRuntime"].(map[string]interface{}); ok {
			if runtimeVersion, ok := ociRuntime["version"].(string); ok {
				status.RuntimeVersion = runtimeVersion
			}
		}
	}

	return nil
}
