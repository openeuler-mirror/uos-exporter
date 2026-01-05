package config

import (
	"fmt"
	"net"
	"network_exporter/pkg/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	ScrapeUrl       *string
	Insecure        *bool
	DefaultSettings = Settings{
		//ScrapeUri: "http://127.0.0.1:24220/api/plugins.json",
		Insecure: false,
	}
)

func init() {
	ScrapeUrl = kingpin.Flag("scrape_uri",
		"Scrape URI").
		Short('s').
		String()
	Insecure = kingpin.Flag("insecure",
		"Ignore server certificate if using https, Default: false.").
		Bool()
	if *ScrapeUrl != "" {
		if err := utils.ValidateURI(*ScrapeUrl); err != nil {
			logrus.Warnf("Invalid scrape uri: %s", err)
			logrus.Warnf("Use default scrape uri: %s", DefaultSettings.ScrapeUri)
			*ScrapeUrl = DefaultSettings.ScrapeUri
		}
	}

	if *Insecure {
		logrus.Warn("Insecure mode enabled, this is not recommended for production use.")
	}
}

type Settings struct {
	ScrapeUri string `yaml:"scrape_uri"`
	Insecure  bool   `yaml:"insecure"`
}

// 网络监控相关配置

type Targets []struct {
	Name     string   `yaml:"name" json:"name"`
	Host     string   `yaml:"host" json:"host"`
	Port     string   `yaml:"port" json:"port"`
	Type     string   `yaml:"type" json:"type"`
	Proxy    string   `yaml:"proxy" json:"proxy"`
	Probe    []string `yaml:"probe" json:"probe"`
	SourceIp string   `yaml:"source_ip" json:"source_ip"`
	Labels   extraKV  `yaml:"labels,omitempty" json:"labels,omitempty"`
}

type HTTPGet struct {
	Interval duration `yaml:"interval" json:"interval" default:"15s"`
	Timeout  duration `yaml:"timeout" json:"timeout" default:"14s"`
}

type TCP struct {
	Interval duration `yaml:"interval" json:"interval" default:"5s"`
	Timeout  duration `yaml:"timeout" json:"timeout" default:"4s"`
}

type MTR struct {
	Interval duration `yaml:"interval" json:"interval" default:"5s"`
	Timeout  duration `yaml:"timeout" json:"timeout" default:"4s"`
	MaxHops  int      `yaml:"max-hops" json:"max-hops" default:"30"`
	Count    int      `yaml:"count" json:"count" default:"10"`
}

type ICMP struct {
	Interval duration `yaml:"interval" json:"interval" default:"5s"`
	Timeout  duration `yaml:"timeout" json:"timeout" default:"4s"`
	Count    int      `yaml:"count" json:"count" default:"10"`
}

type Conf struct {
	Refresh           duration `yaml:"refresh" json:"refresh" default:"0s"`
	Nameserver        string   `yaml:"nameserver" json:"nameserver"`
	NameserverTimeout duration `yaml:"nameserver_timeout" json:"nameserver_timeout" default:"250ms"`
}

type NetworkConfig struct {
	Conf    `yaml:"conf" json:"conf"`
	ICMP    `yaml:"icmp" json:"icmp"`
	MTR     `yaml:"mtr" json:"mtr"`
	TCP     `yaml:"tcp" json:"tcp"`
	HTTPGet `yaml:"http_get" json:"http_get"`
	Targets `yaml:"targets" json:"targets"`
}

type duration time.Duration

type extraKV struct {
	Kv map[string]string `yaml:"kv,omitempty" json:"kv,omitempty"`
}

// UnmarshalYAML is used to unmarshal into map[string]string
func (b *extraKV) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&b.Kv)
}

// Resolver DNS解析器
type Resolver struct {
	Resolver *net.Resolver
	Timeout  time.Duration
}

// SafeConfig 安全配置重载
type SafeConfig struct {
	Cfg *NetworkConfig
	sync.RWMutex
}

// Duration is a convenience getter.
func (d duration) Duration() time.Duration {
	return time.Duration(d)
}

// Set updates the underlying duration.
func (d *duration) Set(dur time.Duration) {
	*d = duration(dur)
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (d *duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = duration(dur)
	return nil
}

// ReloadConfig 安全配置重载
func (sc *SafeConfig) ReloadConfig(confFile string) (err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	var c = &NetworkConfig{}

	cleanPath := filepath.Clean(confFile)
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return fmt.Errorf("config file must be located within %s", configDir)
	}
	content, err := os.ReadFile(confFile)
	if err != nil {
		return fmt.Errorf("reading config file: %s", err)
	}

	if err = yaml.Unmarshal(content, c); err != nil {
		return fmt.Errorf("parsing config file: %s", err)
	}

	// Validate and Filter config
	targets := Targets{}
	re := regexp.MustCompile("^ICMP|MTR|ICMP+MTR|TCP|HTTPGet$")
	for _, t := range c.Targets {
		found := re.MatchString(t.Type)
		if !found {
			logrus.Errorf("Unknown check type, target: %s, check_type: %s, allowed: (ICMP|MTR|ICMP+MTR|TCP|HTTPGet)", t.Name, t.Type)
			continue
		}

		// Filter out the targets that are not assigned to the running host, if the `probe` is not specified don't filter
		if t.Probe == nil {
			targets = append(targets, t)
		} else {
			for _, p := range t.Probe {
				if p == hostname {
					targets = append(targets, t)
					continue
				}
			}
		}
	}

	// Remap the filtered targets
	c.Targets = targets

	// Config precheck
	if c.ICMP.Interval <= 0 || c.MTR.Interval <= 0 || c.TCP.Interval <= 0 || c.HTTPGet.Interval <= 0 {
		return fmt.Errorf("intervals (icmp,mtr,tcp,http_get) must be >0")
	}
	if c.MTR.MaxHops < 0 || c.MTR.MaxHops > 65500 {
		return fmt.Errorf("mtr.max-hops must be between 0 and 65500")
	}
	if c.MTR.Count < 0 || c.MTR.Count > 65500 {
		return fmt.Errorf("mtr.count must be between 0 and 65500")
	}

	sc.Lock()
	sc.Cfg = c
	sc.Unlock()

	return nil
}
