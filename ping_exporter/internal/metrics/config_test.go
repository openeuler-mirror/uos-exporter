package metrics

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestFromYAML(t *testing.T) {
	t.Run("Valid YAML", func(t *testing.T) {
		yamlData := `
targets:
  - addr1
ping:
  interval: 1s
  timeout: 2s
  history-size: 10
  payload-size: 64
dns:
  refresh: 5m
  nameserver: "8.8.8.8"
options:
  disableIPv6: true
  disableIPv4: false
`
		r := bytes.NewBufferString(yamlData)
		cfg, err := FromYAML(r)
		assert.NoError(t, err)
		assert.Equal(t, "addr1", cfg.Targets[0].Addr)
		assert.Equal(t, time.Second, cfg.Ping.Interval.Duration())
		assert.Equal(t, 5*time.Minute, cfg.DNS.Refresh.Duration())
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		yamlData := "invalid_yaml"
		r := bytes.NewBufferString(yamlData)
		_, err := FromYAML(r)
		assert.Error(t, err)
	})
}

func TestToYAML(t *testing.T) {
	t.Run("Valid Config", func(t *testing.T) {
		cfg := &Config{
			Targets: []TargetConfig{{Addr: "addr1"}},
			Ping: struct {
				Interval duration `yaml:"interval"`
				Timeout  duration `yaml:"timeout"`
				History  int      `yaml:"history-size"`
				Size     uint16   `yaml:"payload-size"`
			}{
				Interval: duration(time.Second),
				Timeout:  duration(2 * time.Second),
				History:  10,
				Size:     64,
			},
			DNS: struct {
				Refresh    duration `yaml:"refresh"`
				Nameserver string   `yaml:"nameserver"`
			}{
				Refresh:    duration(5 * time.Minute),
				Nameserver: "8.8.8.8",
			},
			Options: struct {
				DisableIPv6 bool `yaml:"disableIPv6"`
				DisableIPv4 bool `yaml:"disableIPv4"`
			}{
				DisableIPv6: true,
				DisableIPv4: false,
			},
		}
		var buf bytes.Buffer
		err := ToYAML(&buf, cfg)
		assert.NoError(t, err)
		var decoded Config
		err = yaml.Unmarshal(buf.Bytes(), &decoded)
		assert.NoError(t, err)
		assert.Equal(t, cfg.Targets[0].Addr, decoded.Targets[0].Addr)
	})

	t.Run("Write Error", func(t *testing.T) {
		cfg := &Config{}
		errWriter := &errorWriter{}
		err := ToYAML(errWriter, cfg)
		assert.Error(t, err)
	})
}

func TestTargetConfigByAddr(t *testing.T) {
	cfg := &Config{
		Targets: []TargetConfig{
			{Addr: "addr1"},
			{Addr: "addr2", Labels: map[string]string{"key": "value"}},
		},
	}

	t.Run("Found Target", func(t *testing.T) {
		result := cfg.TargetConfigByAddr("addr2")
		assert.Equal(t, "addr2", result.Addr)
		assert.Equal(t, "value", result.Labels["key"])
	})

	t.Run("NotFound Target", func(t *testing.T) {
		result := cfg.TargetConfigByAddr("addr3")
		assert.Equal(t, "addr3", result.Addr)
		assert.Nil(t, result.Labels)
	})
}

func TestTargetConfig_UnmarshalYAML(t *testing.T) {
	t.Run("String Input", func(t *testing.T) {
		data := "addr1"
		var tc TargetConfig
		err := yaml.Unmarshal([]byte(data), &tc)
		assert.NoError(t, err)
		assert.Equal(t, "addr1", tc.Addr)
		assert.Nil(t, tc.Labels)
	})

	t.Run("Map Input", func(t *testing.T) {
		data := "addr1:\n  key: value"
		var tc TargetConfig
		err := yaml.Unmarshal([]byte(data), &tc)
		assert.NoError(t, err)
		assert.Equal(t, "addr1", tc.Addr)
		assert.Equal(t, "value", tc.Labels["key"])
	})

}

func TestTargetConfig_MarshalYAML(t *testing.T) {
	t.Run("No Labels", func(t *testing.T) {
		tc := TargetConfig{Addr: "addr1"}
		result, err := tc.MarshalYAML()
		assert.NoError(t, err)
		assert.Equal(t, "addr1", result)
	})

	t.Run("With Labels", func(t *testing.T) {
		tc := TargetConfig{Addr: "addr1", Labels: map[string]string{"key": "value"}}
		result, err := tc.MarshalYAML()
		assert.NoError(t, err)
		expected := map[string]map[string]string{"addr1": {"key": "value"}}
		assert.Equal(t, expected, result)
	})
}

func TestDuration_UnmarshalYAML(t *testing.T) {
	t.Run("Valid Duration", func(t *testing.T) {
		data := "10s"
		var d duration
		err := yaml.Unmarshal([]byte(data), &d)
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, d.Duration())
	})

	t.Run("Invalid Duration", func(t *testing.T) {
		data := "invalid_duration"
		var d duration
		err := yaml.Unmarshal([]byte(data), &d)
		assert.Error(t, err)
	})
}

func TestDuration_MarshalYAML(t *testing.T) {
	d := duration(10 * time.Second)
	result, err := d.MarshalYAML()
	assert.NoError(t, err)
	assert.Equal(t, "10s", result)
}

// errorWriter simulates a write error.
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}
