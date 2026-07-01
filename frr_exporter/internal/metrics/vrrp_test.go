package metrics

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVRRPConfig(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	config := NewVRRPConfig(logger)

	assert.Equal(t, defaultVrrpCommandTimeout, config.CommandTimeout)
	assert.Equal(t, logger, config.Logger)
}

func TestNewVRRPCollector(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("ValidCreation", func(t *testing.T) {
		collector, err := NewVRRPCollector(logger)
		require.NoError(t, err)
		assert.NotNil(t, collector)

		vrrpCol := collector.(*vrrpCollector)
		assert.Equal(t, logger, vrrpCol.logger)
		assert.NotNil(t, vrrpCol.descriptions)
		assert.IsType(t, &DefaultVrrpCommandExecutor{}, vrrpCol.executor)
		assert.IsType(t, &VRRPProcessorImpl{}, vrrpCol.processor)
		assert.Equal(t, defaultVrrpCommandTimeout, vrrpCol.config.CommandTimeout)
	})
}

func TestVRRPCollector_Update(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	t.Run("Success", func(t *testing.T) {
		collector := &vrrpCollector{
			executor:     &mockVrrpCommandExecutor{data: validVRRPJSON},
			descriptions: getVRRPDesc(),
			logger:       logger,
			processor:    NewVRRPProcessor(logger),
			config:       NewVRRPConfig(logger),
		}

		ch := make(chan prometheus.Metric, 20)
		err := collector.Update(ch)
		require.NoError(t, err)
		assert.Greater(t, len(ch), 0)
	})

	t.Run("CommandExecutionError", func(t *testing.T) {
		collector := &vrrpCollector{
			executor:     &failingVrrpCommandExecutor{},
			descriptions: getVRRPDesc(),
			logger:       logger,
			processor:    NewVRRPProcessor(logger),
			config:       NewVRRPConfig(logger),
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command failed")
	})

	t.Run("JSONProcessingError", func(t *testing.T) {
		collector := &vrrpCollector{
			executor:     &mockVrrpCommandExecutor{data: []byte("{invalid json")},
			descriptions: getVRRPDesc(),
			logger:       logger,
			processor:    NewVRRPProcessor(logger),
			config:       NewVRRPConfig(logger),
		}

		ch := make(chan prometheus.Metric, 10)
		err := collector.Update(ch)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot process output of show vrrp json")
		assert.Contains(t, err.Error(), "invalid character 'i' looking for beginning of object key string")
	})
}

func TestVRRPProcessor_Process(t *testing.T) {
	processor := NewVRRPProcessor(slog.New(slog.NewTextHandler(os.Stdout, nil)))

	t.Run("ValidData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 20)
		err := processor.Process(ch, validVRRPJSON, getVRRPDesc())
		require.NoError(t, err)
		assert.Equal(t, 14, len(ch))
	})

	t.Run("EmptyData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		err := processor.Process(ch, []byte("[]"), getVRRPDesc())
		require.NoError(t, err)
		assert.Equal(t, 0, len(ch))
	})

	t.Run("InvalidData", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		err := processor.Process(ch, []byte("{invalid json"), getVRRPDesc())
		require.Error(t, err)
	})
}

func TestVRRPInstanceProcessor_Process(t *testing.T) {
	processor := NewVRRPInstanceProcessor(getVRRPDesc())

	t.Run("IPv4Instance", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		instance := VrrpInstanceInfo{
			Subinterface: "eth0.100",
			Status:       vrrpStatusMaster,
			Statistics: VrrpInstanceStats{
				AdverTx:         uint32VrrpPtr(100),
				AdverRx:         uint32VrrpPtr(50),
				GarpTx:          uint32VrrpPtr(10),
				NeighborAdverTx: uint32VrrpPtr(5),
				Transitions:     uint32VrrpPtr(3),
			},
		}

		processor.Process(ch, "v4", 1, "eth0", instance)
		assert.Equal(t, 8, len(ch))
	})

	t.Run("IPv6Instance", func(t *testing.T) {
		ch := make(chan prometheus.Metric, 10)
		instance := VrrpInstanceInfo{
			Subinterface: "eth1.200",
			Status:       vrrpStatusBackup,
			Statistics: VrrpInstanceStats{
				AdverTx:         uint32Ptr(200),
				AdverRx:         uint32Ptr(100),
				Transitions:     uint32Ptr(5),
			},
		}

		processor.Process(ch, "v6", 2, "eth1", instance)
		assert.Equal(t, 6, len(ch))
	})
}

func TestVRRPLabelGenerator(t *testing.T) {
	t.Run("BaseLabels", func(t *testing.T) {
		generator := NewVRRPLabelGenerator("v4", 1, "eth0", "eth0.100")
		labels := generator.GetBaseLabels()
		assert.Equal(t, []string{"v4", "1", "eth0", "eth0.100"}, labels)
	})

	t.Run("StateLabels", func(t *testing.T) {
		generator := NewVRRPLabelGenerator("v6", 2, "eth1", "eth1.200")
		labels := generator.GetStateLabels(vrrpStatusMaster)
		assert.Equal(t, []string{"v6", "2", "eth1", "eth1.200", vrrpStatusMaster}, labels)
	})
}

func TestGetVRRPDesc(t *testing.T) {
	descs := getVRRPDesc()

	assert.Contains(t, descs, "vrrpState")
	assert.Contains(t, descs, "adverTx")
	assert.Contains(t, descs, "adverRx")
	assert.Contains(t, descs, "garpTx")
	assert.Contains(t, descs, "neighborAdverTx")
	assert.Contains(t, descs, "transitions")

	stateDesc := descs["vrrpState"]
	assert.Contains(t, stateDesc.String(), "vrrp_state")
	assert.Contains(t, stateDesc.String(), "Status of the VRRP state machine")
}

// Helper functions and mock implementations
var validVRRPJSON = []byte(`[
    {
        "vrid": 1,
        "interface": "eth0",
        "v4": {
            "interface": "eth0.100",
            "status": "Master",
            "stats": {
                "adverTx": 100,
                "adverRx": 50,
                "garpTx": 10,
                "neighborAdverTx": 5,
                "transitions": 3
            }
        },
        "v6": {
            "interface": "eth0.100",
            "status": "Backup",
            "stats": {
                "adverTx": 200,
                "adverRx": 100,
                "transitions": 5
            }
        }
    }
]`)

type mockVrrpCommandExecutor struct {
	data []byte
	err  error
}

func (m *mockVrrpCommandExecutor) Execute(command string, timeout time.Duration) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}

type failingVrrpCommandExecutor struct{}

func (f *failingVrrpCommandExecutor) Execute(command string, timeout time.Duration) ([]byte, error) {
	return nil, errors.New("command failed")
}

func uint32VrrpPtr(val uint32) *uint32 {
	return &val
}

// Test helper to capture log output
func captureVrrpLogs(f func()) string {
	var buf bytes.Buffer
	original := slog.Default()
	defer slog.SetDefault(original)

	logger := slog.New(slog.NewTextHandler(&buf, nil))
	slog.SetDefault(logger)

	f()
	return buf.String()
}
// Part 2 commit for frr_exporter/internal/metrics/vrrp_test.go
