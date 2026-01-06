package metrics

import (
	"fmt"
	"keepalived_container_exporter/internal/exporter"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func init() {
	exporter.Register(
		NewKeepalivedExporter())
}

func NewKeepalivedExporter() *KeepalivedCollector {
	var keepalivedJSON = false
	var keepalivedPID = "/var/run/keepalived.pid"
	var keepalivedCheckScript = ""
	var keepalivedContainerName = ""
	var keepalivedContainerTmpDir = "/tmp"

	config, err := LoadConfig("/etc/uos-exporter/keepalived-container-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error get BIND URI %v\n", err)
	} else {
		keepalivedJSON = config.KeepalivedJSON
		keepalivedPID = config.KeepalivedPID
		keepalivedCheckScript = config.KeepalivedCheckScript
		keepalivedContainerName = config.KeepalivedContainerName
		keepalivedContainerTmpDir = config.KeepalivedContainerTmpDir
	}

	var c Collector
	if keepalivedContainerName != "" {
		c = NewKeepalivedContainerCollectorHost(
			keepalivedJSON,
			keepalivedContainerName,
			keepalivedContainerTmpDir,
			keepalivedPID,
		)
	} else {
		c = NewKeepalivedHostCollectorHost(keepalivedJSON, keepalivedPID)
	}
	return NewKeepalivedCollector(keepalivedJSON, keepalivedCheckScript, c)
}

type KeepalivedConfig struct {
	KeepalivedJSON            bool   `yaml:"keepalived_json"`
	KeepalivedPID             string `yaml:"keepalived_pid"`
	KeepalivedCheckScript     string `yaml:"keepalived_check_script"`
	KeepalivedContainerName   string `yaml:"keepalived_container_name"`
	KeepalivedContainerTmpDir string `yaml:"keepalived_container_tmp_dir"`
}

func LoadConfig(path string) (*KeepalivedConfig, error) {
	cleanPath := filepath.Clean(path)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return nil, fmt.Errorf("config file must be located within %s", configDir)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var config KeepalivedConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
