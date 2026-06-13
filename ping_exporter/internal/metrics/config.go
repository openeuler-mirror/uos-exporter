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
func FromYAML(r io.Reader) (*Config, error) {
	c := &Config{}
	err := yaml.NewDecoder(r).Decode(c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode YAML: %w", err)
	}

	return c, nil
}

// ToYAML encodes the given configuration to the writer as YAML
func ToYAML(w io.Writer, cfg *Config) error {
	err := yaml.NewEncoder(w).Encode(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}

	return nil
}

func (cfg *Config) TargetConfigByAddr(addr string) TargetConfig {
	for _, t := range cfg.Targets {
		if t.Addr == addr {
			return t
		}
	}

	return TargetConfig{Addr: addr}
}

type TargetConfig struct {
	Addr   string
	Labels map[string]string
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *TargetConfig) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err == nil {
		d.Addr = s
		return nil
	}

	var x map[string]map[string]string
	if err := unmashal(&x); err != nil {
		return err
	}

	for addr, l := range x {
		d.Addr = addr
		d.Labels = l
	}

	return nil
}

func (d TargetConfig) MarshalYAML() (interface{}, error) {
	if d.Labels == nil {
		return d.Addr, nil
	}
	ret := make(map[string]map[string]string)
	ret[d.Addr] = d.Labels
	return ret, nil
}

type duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *duration) UnmarshalYAML(unmashal func(interface{}) error) error {
	var s string
	if err := unmashal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("failed to decode duration: %w", err)
	}
	*d = duration(dur)

	return nil
}

func (d duration) MarshalYAML() (interface{}, error) {
	return d.Duration().String(), nil
}

// Duration is a convenience getter.
func (d duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set updates the underlying duration.
func (d *duration) Set(dur time.Duration) {
	*d = duration(dur)
}
