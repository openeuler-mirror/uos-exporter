package metrics

import (
	// "fmt"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPIMConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := NewPIMConfig(logger)
	
	assert.Equal(t, defaultCommandTimeout, config.CommandTimeout)
	assert.Equal(t, logger, config.Logger)
}

func TestNewPIMCollector(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	t.Run("ValidCreation", func(t *testing.T) {
		collector, err := NewPIMCollector(logger)
		require.NoError(t, err)
		assert.NotNil(t, collector)
		
		pimCol := collector.(*pimCollector)
		assert.Equal(t, logger, pimCol.logger)
		assert.NotNil(t, pimCol.descriptions)
		assert.IsType(t, &DefaultCommandRunner{}, pimCol.commandRunner)
		assert.IsType(t, &DefaultTimeParser{}, pimCol.timeParser)
		assert.NotNil(t, pimCol.config)
		assert.Equal(t, defaultCommandTimeout, pimCol.config.CommandTimeout)
	})
}

func TestPIMCollector_Update(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Success", func(t *testing.T) {
		collector := &pimCollector{
			commandRunner: &mockCommandRunner{data: validPIMNeighborJSON},
			descriptions:  getPIMDesc(),
			logger:        logger,
			timeParser:    &DefaultTimeParser{},
			config:        NewPIMConfig(logger),
		}
		
		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.NoError(t, err)
		assert.Greater(t, len(ch), 0)
	})
	
	t.Run("CommandExecutionError", func(t *testing.T) {
		collector := &pimCollector{
			commandRunner: &failingCommandRunner{},
			descriptions:  getPIMDesc(),
			logger:        logger,
			timeParser:    &DefaultTimeParser{},
			config:        NewPIMConfig(logger),
		}
		
		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command failed")
	})
	
	t.Run("JSONProcessingError", func(t *testing.T) {
		collector := &pimCollector{
			commandRunner: &mockCommandRunner{data: []byte("{invalid json")},
			descriptions:  getPIMDesc(),
			logger:        logger,
			timeParser:    &DefaultTimeParser{},
			config:        NewPIMConfig(logger),
		}
		
		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal PIM neighbors data")
	})
}

func TestPIMNeighborProcessor_Process(t *testing.T) {
	processor := NewPIMNeighborProcessor(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		getPIMDesc(),
	)
	
	t.Run("ValidData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		err := processor.Process(ch, validPIMNeighborJSON)
		require.NoError(t, err)
		assert.Equal(t, 5, len(ch)) 
	})
	
	t.Run("EmptyData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		err := processor.Process(ch, []byte("{}"))
		require.NoError(t, err)
		assert.Equal(t, 0, len(ch))
	})
	
	t.Run("InvalidVRFData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		err := processor.Process(ch, []byte(`{"vrf1": "invalid"}`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal VRF instance")
	})
	
	t.Run("InvalidInterfaceData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		data := []byte(`{"vrf1": {"eth0": "invalid"}}`)
		err := processor.Process(ch, data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal neighbor data")
	})
}

func TestPIMNeighborProcessor_ProcessNeighbor(t *testing.T) {
	processor := NewPIMNeighborProcessor(
		slog.New(slog.NewTextHandler(os.Stdout, nil)),
		getPIMDesc(),
	)
	
	t.Run("ValidUptime", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		neighbor := pimNeighbor{
			Interface: "eth0",
			Neighbor:  "192.168.1.1",
			UpTime:    "01:23:45",
		}
		
		err := processor.processNeighbor(ch, "vrf1", "eth0", "192.168.1.1", neighbor)
		require.NoError(t, err)
		assert.Equal(t, 1, len(ch))
	})
	
	t.Run("InvalidUptime", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		neighbor := pimNeighbor{
			Interface: "eth0",
			Neighbor:  "192.168.1.1",
			UpTime:    "invalid-time",
		}
		
		err := processor.processNeighbor(ch, "vrf1", "eth0", "192.168.1.1", neighbor)
		require.NoError(t, err) // Should log error but not return it
		assert.Equal(t, 0, len(ch))
	})
}

func TestNeighborCounter(t *testing.T) {
	counter := NewNeighborCounter()
	assert.Equal(t, 0.0, counter.Count())
	
	counter.Increment()
	assert.Equal(t, 1.0, counter.Count())
	
	counter.Increment()
	counter.Increment()
	assert.Equal(t, 3.0, counter.Count())
}

func TestParseHMS(t *testing.T) {
	t.Run("ValidTime", func(t *testing.T) {
		sec, err := parseHMS("01:02:03")
		require.NoError(t, err)
		assert.Equal(t, uint64(3723), sec) // 1*3600 + 2*60 + 3
	})
	
	t.Run("InvalidFormat", func(t *testing.T) {
		_, err := parseHMS("01-02-03")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid time format")
	})
	
	t.Run("NonNumeric", func(t *testing.T) {
		_, err := parseHMS("aa:bb:cc")
		require.Error(t, err)
	})
	
	t.Run("Incomplete", func(t *testing.T) {
		_, err := parseHMS("01:02")
		require.Error(t, err)
	})
}

func TestDefaultCommandRunner(t *testing.T) {
	runner := &DefaultCommandRunner{}
	
	t.Run("ValidCommand", func(t *testing.T) {
		// This test requires the actual command to be available
		// Skip in environments where vtysh isn't available
		if os.Getenv("SKIP_LIVE_COMMANDS") == "1" {
			t.Skip("Skipping live command execution")
		}
		
		output, err := runner.Execute("show ip pim vrf all neighbor json", 5*time.Second)
		if err != nil {
			t.Logf("Command execution failed (may be expected): %v", err)
			return
		}
		
		// Should at least be valid JSON
		var data interface{}
		assert.NoError(t, json.Unmarshal(output, &data))
	})
}

func TestGetPIMDesc(t *testing.T) {
	descs := getPIMDesc()
	
	assert.Contains(t, descs, "neighborCount")
	assert.Contains(t, descs, "upTime")
	
	neighborCountDesc := descs["neighborCount"]
	assert.Contains(t, neighborCountDesc.String(), "pim_neighbor_count_total")
	
	upTimeDesc := descs["upTime"]
	assert.Contains(t, upTimeDesc.String(), "pim_neighbor_uptime_seconds")
}

// Mock implementations for testing
var validPIMNeighborJSON = []byte(`{
	"vrf1": {
		"eth0": {
			"192.168.1.1": {
				"interface": "eth0",
				"neighbor": "192.168.1.1",
				"uptime": "01:23:45"
			},
			"192.168.1.2": {
				"interface": "eth0",
				"neighbor": "192.168.1.2",
				"uptime": "02:34:56"
			}
		}
	},
	"vrf2": {
		"eth1": {
			"10.0.0.1": {
				"interface": "eth1",
				"neighbor": "10.0.0.1",
				"uptime": "00:05:00"
			}
		}
	}
}`)

type mockCommandRunner struct {
	data []byte
	err  error
}

func (m *mockCommandRunner) Execute(command string, timeout time.Duration) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

type failingCommandRunner struct{}

func (f *failingCommandRunner) Execute(command string, timeout time.Duration) ([]byte, error) {
	return nil, errors.New("command failed")
}

type mockTimeParser struct {
	result uint64
	err    error
}

func (m *mockTimeParser) ParseHMS(timeStr string) (uint64, error) {
	return m.result, m.err
}

// Test helper to capture log output
func captureLogs(f func()) string {
	var buf bytes.Buffer
	original := slog.Default()
	defer slog.SetDefault(original)
	
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)
	
	f()
	return buf.String()
}