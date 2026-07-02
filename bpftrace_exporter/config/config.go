package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	BpftracePath    *string
	ScriptPath      *string
	VarDefs         *string
	Insecure        *bool
	DefaultSettings = Settings{
		BpftracePath: "/usr/bin/bpftrace",
		Insecure:     false,
	}
)

func init() {
	BpftracePath = kingpin.Flag("bpftrace.path",
		"Path to bpftrace executable").
		Default("/usr/bin/bpftrace").
		String()

	ScriptPath = kingpin.Flag("script.path",
		"Path to bpftrace script").
		Default("").
		String()

	VarDefs = kingpin.Flag("var.defs",
		"Variable definitions (comma-separated, e.g. \"http_latency:hist,error_count:counter\")").
		Default("").
		String()

	Insecure = kingpin.Flag("insecure",
		"Ignore server certificate if using https, Default: false.").
		Bool()

	if *Insecure {
		logrus.Warn("Insecure mode enabled, this is not recommended for production use.")
	}
}

type Settings struct {
	BpftracePath string `yaml:"bpftrace_path"`
	ScriptPath   string `yaml:"script_path"`
	VarDefs      string `yaml:"var_defs"`
	Insecure     bool   `yaml:"insecure"`
}

// LoadConfigFile 从配置文件中加载设置
func LoadConfigFile(configPath string) (*Settings, error) {
	settings := &Settings{}
	cleanPath := filepath.Clean(configPath)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}

	content, err := os.ReadFile(cleanPath)
	if err != nil {
		return settings, err
	}

	// 尝试解析完整配置
	var fullConfig map[string]interface{}
	err = yaml.Unmarshal(content, &fullConfig)
	if err != nil {
		return settings, err
	}

	// 从配置中提取bpftrace相关设置
	if bpftracePath, ok := fullConfig["bpftrace_path"].(string); ok {
		settings.BpftracePath = bpftracePath
	}

	if scriptPath, ok := fullConfig["script_path"].(string); ok {
		settings.ScriptPath = scriptPath
	}

	if varDefs, ok := fullConfig["var_defs"].(string); ok {
		settings.VarDefs = varDefs
	}

	if insecure, ok := fullConfig["insecure"].(bool); ok {
		settings.Insecure = insecure
	}

	logrus.Infof("Loaded settings from config file: BpftracePath=%s, ScriptPath=%s, VarDefs=%s",
		settings.BpftracePath, settings.ScriptPath, settings.VarDefs)

	return settings, nil
}
// Part 2 commit for bpftrace_exporter/config/config.go
