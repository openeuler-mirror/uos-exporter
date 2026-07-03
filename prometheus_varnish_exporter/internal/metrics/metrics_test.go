package metrics

import (
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStartParamsInitialization tests the initialization of StartParams.
func TestStartParamsInitialization(t *testing.T) {
	// Backup original command-line arguments
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set test command-line arguments
	os.Args = []string{
		"test",
		"-web.listen-address=:8080",
		"-web.telemetry-path=/custom-metrics",
		"-web.health-path=/health",
		"-varnishstat-path=/usr/bin/varnishstat",
		"-n=instance1",
		"-N=vsm1",
		"-docker-container-name=container1",
		"-exit-on-errors=true",
		"-verbose=true",
		"-test=true",
		"-raw=true",
		"-with-go-metrics=true",
	}

	// Reset flag.CommandLine to avoid "flag redefined" errors
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Call init to initialize StartParams
	//initialize()

	// Assert StartParams values
	assert.Equal(t, ":8080", StartParams.ListenAddress)
	assert.Equal(t, "/custom-metrics", StartParams.Path)
	assert.Equal(t, "/health", StartParams.HealthPath)
	assert.Equal(t, "/usr/bin/varnishstat", StartParams.VarnishstatExe)
	assert.Equal(t, "instance1", StartParams.Params.Instance)
	assert.Equal(t, "vsm1", StartParams.Params.VSM)
	assert.Equal(t, "container1", StartParams.VarnishDockerContainer)
	assert.True(t, StartParams.ExitOnErrors)
	assert.True(t, StartParams.Verbose)
	assert.True(t, StartParams.Test)
	assert.True(t, StartParams.Raw)
	assert.True(t, StartParams.WithGoMetrics)
}

// TestExitHandlerSet tests the Set method of exitHandler.
func TestExitHandlerSet(t *testing.T) {
	eh := &exitHandler{
		exitOnError: true,
		logger:     &mockLogger{},
	}

	// Test setting an error
	err := fmt.Errorf("test error")
	result := eh.Set(err)
	assert.Equal(t, err, result)
	assert.True(t, eh.HasError())

	// Test clearing an error
	result = eh.Set(nil)
	assert.Nil(t, result)
	assert.False(t, eh.HasError())
}

// TestExitHandlerErrorf tests the Errorf method of exitHandler.
func TestExitHandlerErrorf(t *testing.T) {
	eh := &exitHandler{
		exitOnError: false,
		logger:     &mockLogger{},
	}

	// Test formatting an error
	err := eh.Errorf("test error: %s", "details")
	assert.Error(t, err)
	assert.Equal(t, "test error: details", err.Error())
}

// TestGetVersion tests the getVersion function.
func TestGetVersion(t *testing.T) {
	// Test development version
	Version = ""
	version := getVersion(false)
	assert.Equal(t, "dev", version)

	// Test version with hash
	Version = "1.0.0"
	VersionHash = "abc123"
	version = getVersion(false)
	assert.Equal(t, "v1.0.0 (abc123)", version)

	// Test version with date
	VersionDate = "2023-01-01"
	version = getVersion(true)
	assert.Equal(t, "v1.0.0 (abc123) 2023-01-01", version)
}

// TestVarnishstatParamsMake tests the make method of varnishstatParams.
func TestVarnishstatParamsMake(t *testing.T) {
	params := &varnishstatParams{
		Instance: "instance1",
		VSM:      "vsm1",
	}

	// Test with both instance and VSM
	result := params.make()
	assert.Equal(t, []string{"-n", "instance1", "-N", "vsm1"}, result)

	// Test with only instance
	params.VSM = ""
	result = params.make()
	assert.Equal(t, []string{"-n", "instance1"}, result)

	// Test with only VSM
	params.Instance = ""
	params.VSM = "vsm1"
	result = params.make()
	assert.Equal(t, []string{"-N", "vsm1"}, result)

	// Test with neither instance nor VSM
	params.Instance = ""
	params.VSM = ""
	result = params.make()
	assert.Empty(t, result)
}

// TestVarnishstatParamsIsEmpty tests the isEmpty method of varnishstatParams.
func TestVarnishstatParamsIsEmpty(t *testing.T) {
	params := &varnishstatParams{}

	// Test empty params
	assert.True(t, params.isEmpty())

	// Test non-empty instance
	params.Instance = "instance1"
	assert.False(t, params.isEmpty())

	// Test non-empty VSM
	params.Instance = ""
	params.VSM = "vsm1"
	assert.False(t, params.isEmpty())

	// Test non-empty instance and VSM
	params.Instance = "instance1"
	params.VSM = "vsm1"
	assert.False(t, params.isEmpty())
}

// mockLogger is a mock implementation of the Logger interface for testing.
type mockLogger struct{}

func (m *mockLogger) Fatal(format string, args ...interface{}) {}
func (m *mockLogger) Error(format string, args ...interface{}) {}
// Part 2 commit for prometheus_varnish_exporter/internal/metrics/metrics_test.go
