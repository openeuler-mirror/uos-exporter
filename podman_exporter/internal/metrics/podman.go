package metrics

import (
	"os"
	"path/filepath"
	"podman_exporter/internal/exporter"
	"podman_exporter/internal/metrics/collectors/container"
	"podman_exporter/internal/metrics/collectors/core"
	"podman_exporter/internal/metrics/collectors/image"
	"podman_exporter/internal/metrics/collectors/network"
	"podman_exporter/internal/metrics/collectors/pod"
	"podman_exporter/internal/metrics/collectors/system"
	"podman_exporter/internal/metrics/collectors/volume"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var config Config

func init() {
	err := LoadConfigFromFile("/etc/uos-exporter/podman-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error loading config file: %v", err)
		os.Exit(1)
	}

	logger := logrus.New()
	PodmanCollector := NewPodmanCollector(logger)
	ScrapeCollector := core.NewScrapeCollector(PodmanCollector, logger)
	exporter.Register(ScrapeCollector)
}

type Config struct {
}

// validateConfigPath validates that the config file path is safe to read
func validateConfigPath(path string) error {
	if path == "" {
		return errors.New("config path cannot be empty")
	}

	// Resolve the path to prevent directory traversal
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return errors.Wrap(err, "failed to resolve config path")
	}

	// Only allow paths under /etc/uos-exporter/ for security
	allowedPrefix := "/etc/uos-exporter/"
	if !strings.HasPrefix(cleanPath, allowedPrefix) {
		return errors.Errorf("config path must be under %s", allowedPrefix)
	}

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return errors.New("config path contains directory traversal")
	}

	return nil
}

func LoadConfigFromFile(path string) error {
	// // Validate the path before reading
	// if err := validateConfigPath(path); err != nil {
	// 	return errors.Wrap(err, "invalid config path")
	// }
	// cleanPath, err := filepath.Abs(path)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to resolve config path")
	// }
	// // Only allow paths under /etc/uos-exporter/ for security
	// allowedPrefix := "/etc/uos-exporter/"
	// if !strings.HasPrefix(cleanPath, allowedPrefix) {
	// 	return errors.Errorf("config path must be under %s", allowedPrefix)
	// }
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, "/etc/uos-exporter/") {
		return errors.Errorf("config path must be under %s", "/etc/uos-exporter/")
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file")
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return errors.Wrap(err, "failed to parse config file")
	}
	return nil
}

type PodmanCollector struct {
	containerCollector *container.ContainerCollector
	imageCollector     *image.Collector
	podCollector       *pod.Collector
	networkCollector   *network.Collector
	volumeCollector    *volume.Collector
	systemCollector    *system.Collector
}

func NewPodmanCollector(logger *logrus.Logger) *PodmanCollector {
	collector := &PodmanCollector{}

	containerCollector := container.NewCollector(
		"/usr/bin/podman",
		true)

	imageCollector := image.NewCollector(logger, 30*time.Second)
	podCollector := pod.NewCollector(logger, 30*time.Second)
	networkCollector := network.NewCollector(logger, 30*time.Second)
	volumeCollector := volume.NewCollector(logger, 30*time.Second)
	systemCollector := system.NewCollector(logger, 30*time.Second)

	collector.containerCollector = containerCollector
	collector.imageCollector = imageCollector
	collector.podCollector = podCollector
	collector.networkCollector = networkCollector
	collector.volumeCollector = volumeCollector
	collector.systemCollector = systemCollector

	return collector
}

func (c *PodmanCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 收集容器指标
	if err := c.containerCollector.CollectWithError(ch); err != nil {
		return err
	}

	// 收集镜像指标
	if err := c.imageCollector.CollectWithError(ch); err != nil {
		return err
	}

	// 收集Pod指标
	if err := c.podCollector.CollectWithError(ch); err != nil {
		return err
	}

	// 收集网络指标
	if err := c.networkCollector.CollectWithError(ch); err != nil {
		return err
	}

	// 收集Volume指标
	if err := c.volumeCollector.CollectWithError(ch); err != nil {
		return err
	}

	// 收集系统指标
	if err := c.systemCollector.CollectWithError(ch); err != nil {
		return err
	}

	return nil
}

func (c *PodmanCollector) GetSubsystem() string {
	return "podman"
}

func (c *PodmanCollector) Describe(ch chan<- *prometheus.Desc) {
	c.containerCollector.Describe(ch)
	c.imageCollector.Describe(ch)
	c.podCollector.Describe(ch)
	c.networkCollector.Describe(ch)
	c.volumeCollector.Describe(ch)
	c.systemCollector.Describe(ch)
}

func (c *PodmanCollector) Collect(ch chan<- prometheus.Metric) {
	c.containerCollector.Collect(ch)
	c.imageCollector.Collect(ch)
	c.podCollector.Collect(ch)
	c.networkCollector.Collect(ch)
	c.volumeCollector.Collect(ch)
	c.systemCollector.Collect(ch)
}
