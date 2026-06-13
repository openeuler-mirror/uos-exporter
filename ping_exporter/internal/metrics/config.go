package metrics

import (
	"fmt"
	"io"
	"time"

	yaml "gopkg.in/yaml.v2"
)

// Config represents configuration for the exporter.
type Config struct {
	Targets []TargetConfig `yaml:"targets"`

	Ping struct {
		Interval duration `yaml:"interval"`
		Timeout  duration `yaml:"timeout"`
		History  int      `yaml:"history-size"`
		Size     uint16   `yaml:"payload-size"`
	} `yaml:"ping"`

	DNS struct {
		Refresh    duration `yaml:"refresh"`
		Nameserver string   `yaml:"nameserver"`
	} `yaml:"dns"`

	Options struct {
		DisableIPv6 bool `yaml:"disableIPv6"` // prohibits DNS resolved IPv6 addresses
		DisableIPv4 bool `yaml:"disableIPv4"` // prohibits DNS resolved IPv4 addresses
	} `yaml:"options"`
}

// FromYAML reads YAML from reader and unmarshals it to Config.

// TODO: implement functions
