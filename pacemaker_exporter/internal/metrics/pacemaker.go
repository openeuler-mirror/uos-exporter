package metrics

import (
	"pacemaker_exporter/internal/exporter"

	"github.com/sirupsen/logrus"
)

func init() {
	exporter.Register(
		NewKeepalivedExporter())
}

func NewKeepalivedExporter() *PacemakerCollector {
	c, err := NewPacemakerCollector()
	if err != nil {
		logrus.Warnln("Couldn't create", err)
		return nil
	}

	return c
}

type KeepalivedConfig struct {
	KeepalivedJSON            bool   `yaml:"keepalived_json"`
	KeepalivedPID             string `yaml:"keepalived_pid"`
	KeepalivedCheckScript     string `yaml:"keepalived_check_script"`
	KeepalivedContainerName   string `yaml:"keepalived_container_name"`
	KeepalivedContainerTmpDir string `yaml:"keepalived_container_tmp_dir"`
}

// func LoadConfig(path string) (*KeepalivedConfig, error) {
// 	data, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var config KeepalivedConfig
// 	if err := yaml.Unmarshal(data, &config); err != nil {
// 		return nil, err
// 	}

// 	return &config, nil
// }
