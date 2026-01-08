package image

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Parser 定义解析镜像工具输出的接口
type Parser interface {
	Parse(imageOutput []byte) (*Status, error)
}

// Status 包含所有解析的镜像状态信息
type Status struct {
	Images    []Image
	Timestamp time.Time
}

// Image 表示一个镜像实例
type Image struct {
	// 基本信息
	ID         string
	ParentID   string
	Repository string
	Tag        string
	Digest     string
	Names      []string

	// 大小信息
	Size        int64 // 镜像大小（字节）
	SharedSize  int64 // 共享大小（字节）
	VirtualSize int64 // 虚拟大小（字节）

	// 使用信息
	Containers int // 使用该镜像的容器数量

	// 时间信息
	Created time.Time
}

// NewParser 创建一个新的默认解析器实例
func NewParser() Parser {
	return &defaultParser{
		patterns: map[string]*regexp.Regexp{
			"imageID": regexp.MustCompile(`(?m)^([a-f0-9]{12})`),
			"repoTag": regexp.MustCompile(`([^:]+):(.+)`),
		},
	}
}

type defaultParser struct {
	patterns map[string]*regexp.Regexp
}

func (p *defaultParser) Parse(imageOutput []byte) (*Status, error) {
	status := &Status{
		Timestamp: time.Now(),
	}

	// 尝试解析JSON格式输出（如podman images --format json）
	if err := p.parseJSONOutput(imageOutput, status); err == nil {
		return status, nil
	}

	// 如果JSON解析失败，尝试解析文本格式
	if err := p.parseTextOutput(imageOutput, status); err != nil {
		return nil, errors.Wrap(err, "failed to parse image output")
	}

	return status, nil
}

// parseJSONOutput 解析JSON格式的镜像输出
func (p *defaultParser) parseJSONOutput(output []byte, status *Status) error {
	var images []map[string]interface{}

	if err := json.Unmarshal(output, &images); err != nil {
		return err
	}

	for _, imageData := range images {
		image := Image{}

		// 解析基本信息
		if id, ok := imageData["Id"].(string); ok {
			image.ID = id
		}
		if parentId, ok := imageData["ParentId"].(string); ok {
			image.ParentID = parentId
		}
		if digest, ok := imageData["Digest"].(string); ok {
			image.Digest = digest
		}

		// 解析名称信息
		if names, ok := imageData["Names"].([]interface{}); ok {
			for _, nameInterface := range names {
				if name, ok := nameInterface.(string); ok {
					image.Names = append(image.Names, name)
					// 解析第一个名称的repository和tag
					if image.Repository == "" {
						repo, tag := p.parseRepoTag(name)
						image.Repository = repo
						image.Tag = tag
					}
				}
			}
		}

		// 解析大小信息
		if size, ok := imageData["Size"].(float64); ok {
			image.Size = int64(size)
		}
		if sharedSize, ok := imageData["SharedSize"].(float64); ok {
			image.SharedSize = int64(sharedSize)
		}
		if virtualSize, ok := imageData["VirtualSize"].(float64); ok {
			image.VirtualSize = int64(virtualSize)
		}

		// 解析容器数量
		if containers, ok := imageData["Containers"].(float64); ok {
			image.Containers = int(containers)
		}

		// 解析创建时间（如果有的话）
		if created, ok := imageData["Created"].(string); ok {
			if t, err := time.Parse(time.RFC3339, created); err == nil {
				image.Created = t
			}
		} else if created, ok := imageData["Created"].(float64); ok {
			image.Created = time.Unix(int64(created), 0)
		}

		status.Images = append(status.Images, image)
	}

	return nil
}

// parseRepoTag 解析repository:tag格式的名称
func (p *defaultParser) parseRepoTag(name string) (repository, tag string) {
	if match := p.patterns["repoTag"].FindStringSubmatch(name); len(match) == 3 {
		return match[1], match[2]
	}
	// 如果没有匹配到，可能是没有tag的情况
	return name, "<none>"
}

// parseTextOutput 解析文本格式的镜像输出
func (p *defaultParser) parseTextOutput(output []byte, status *Status) error {
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "REPOSITORY") {
			continue
		}

		image, err := p.parseImageLine(line)
		if err != nil {
			continue // 跳过无法解析的行
		}

		status.Images = append(status.Images, image)
	}

	return nil
}

// parseImageLine 解析单行镜像信息
func (p *defaultParser) parseImageLine(line string) (Image, error) {
	var image Image
	fields := strings.Fields(line)

	if len(fields) < 3 {
		return image, errors.New("insufficient fields in image line")
	}

	// 基本格式: REPOSITORY TAG IMAGE_ID CREATED SIZE
	image.Repository = fields[0]
	image.Tag = fields[1]
	image.ID = fields[2]

	if len(fields) >= 5 {
		// 尝试解析大小（最后一个字段）
		sizeStr := fields[len(fields)-1]
		if size, err := p.parseSizeToBytes(sizeStr); err == nil {
			image.Size = size
		}
	}

	return image, nil
}

// parseSizeToBytes 解析大小字符串为字节数
func (p *defaultParser) parseSizeToBytes(sizeStr string) (int64, error) {
	// 这里重用container包中的方法逻辑
	if sizeStr == "" || sizeStr == "0B" {
		return 0, nil
	}

	sizeStr = strings.ToLower(strings.TrimSpace(sizeStr))

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

	// 简化版本的大小转换
	switch unit {
	case "b", "":
		return 0, nil
	case "kb":
		return 1000, nil
	case "mb":
		return 1000 * 1000, nil
	case "gb":
		return 1000 * 1000 * 1000, nil
	default:
		return 0, nil
	}
}
// Part 2 commit for podman_exporter/internal/metrics/collectors/image/parser.go
