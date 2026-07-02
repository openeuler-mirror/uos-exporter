package metrics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewHwmonCollector(t *testing.T) {
	collector := NewHwmonCollector()
	if collector == nil {
		t.Fatal("NewHwmonCollector returned nil")
	}
	if collector.baseMetrics == nil {
		t.Error("baseMetrics should not be nil")
	}
}

func TestCleanMetricName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic name",
			input:    "temp1",
			expected: "temp1",
		},
		{
			name:     "uppercase to lowercase",
			input:    "TEMP1",
			expected: "temp1",
		},
		{
			name:     "special characters",
			input:    "temp-1.input",
			expected: "temp_1_input",
		},
		{
			name:     "spaces",
			input:    "temp 1 input",
			expected: "temp_1_input",
		},
		{
			name:     "leading and trailing underscores",
			input:    "_temp1_",
			expected: "temp1",
		},
		{
			name:     "multiple special chars",
			input:    "temp@#$1%^&input",
			expected: "temp___1___input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanMetricName(tt.input)
			if result != tt.expected {
				t.Errorf("cleanMetricName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExplodeSensorFilename(t *testing.T) {
	tests := []struct {
		name             string
		filename         string
		expectedOk       bool
		expectedType     string
		expectedNum      int
		expectedProperty string
	}{
		{
			name:             "basic temp sensor",
			filename:         "temp1_input",
			expectedOk:       true,
			expectedType:     "temp",
			expectedNum:      1,
			expectedProperty: "input",
		},
		{
			name:             "fan sensor",
			filename:         "fan2_input",
			expectedOk:       true,
			expectedType:     "fan",
			expectedNum:      2,
			expectedProperty: "input",
		},
		{
			name:             "sensor without number",
			filename:         "beep_enable",
			expectedOk:       true,
			expectedType:     "beep_enable",
			expectedNum:      0,
			expectedProperty: "",
		},
		{
			name:             "sensor without property",
			filename:         "temp1",
			expectedOk:       true,
			expectedType:     "temp",
			expectedNum:      1,
			expectedProperty: "",
		},
		{
			name:             "invalid filename",
			filename:         "123invalid",
			expectedOk:       false,
			expectedType:     "",
			expectedNum:      0,
			expectedProperty: "",
		},
		{
			name:             "vrm sensor",
			filename:         "vrm",
			expectedOk:       true,
			expectedType:     "vrm",
			expectedNum:      0,
			expectedProperty: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, sensorType, sensorNum, sensorProperty := explodeSensorFilename(tt.filename)
			
			if ok != tt.expectedOk {
				t.Errorf("explodeSensorFilename(%q) ok = %v, want %v", tt.filename, ok, tt.expectedOk)
			}
			if sensorType != tt.expectedType {
				t.Errorf("explodeSensorFilename(%q) type = %q, want %q", tt.filename, sensorType, tt.expectedType)
			}
			if sensorNum != tt.expectedNum {
				t.Errorf("explodeSensorFilename(%q) num = %d, want %d", tt.filename, sensorNum, tt.expectedNum)
			}
			if sensorProperty != tt.expectedProperty {
				t.Errorf("explodeSensorFilename(%q) property = %q, want %q", tt.filename, sensorProperty, tt.expectedProperty)
			}
		})
	}
}

func TestSysFilePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic path",
			input:    "hwmon",
			expected: "/sys/hwmon",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "/sys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sysFilePath(tt.input)
			if result != tt.expected {
				t.Errorf("sysFilePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestAddValueFile(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_sensor")
	
	// Write test data to the file
	testData := "12345\n"
	err := os.WriteFile(testFile, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Test adding value from file
	data := make(map[string]map[string]string)
	addValueFile(data, "temp1", "input", testFile)
	
	// Check if data was added correctly
	if val, ok := data["temp1"]["input"]; !ok {
		t.Error("Value was not added to data map")
	} else if strings.TrimSpace(val) != "12345" {
		t.Errorf("Expected value '12345', got '%s'", val)
	}
}

func TestAddValueFileNonExistent(t *testing.T) {
	// Test with non-existent file
	data := make(map[string]map[string]string)
	addValueFile(data, "temp1", "input", "/non/existent/file")
	
	// Should not add anything to the map
	if len(data) != 0 {
		t.Error("Data map should be empty when file doesn't exist")
	}
}

func TestCollectSensorDataEmptyDir(t *testing.T) {
	// Create a temporary empty directory
	tmpDir := t.TempDir()
	
	data := make(map[string]map[string]string)
	err := collectSensorData(tmpDir, data)
	
	if err != nil {
		t.Errorf("collectSensorData should not return error for empty dir: %v", err)
	}
	
	if len(data) != 0 {
		t.Error("Data map should be empty for empty directory")
	}
}

func TestCollectSensorDataWithValidFiles(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir := t.TempDir()
	
	// Create valid sensor files
	validFiles := map[string]string{
		"temp1_input": "25000",
		"fan1_input":  "1500",
		"in0_input":   "1200",
	}
	
	for filename, content := range validFiles {
		err := os.WriteFile(filepath.Join(tmpDir, filename), []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}
	
	// Also create an invalid file that should be ignored
	err := os.WriteFile(filepath.Join(tmpDir, "invalid_file"), []byte("ignored"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid test file: %v", err)
	}
	
	data := make(map[string]map[string]string)
	err = collectSensorData(tmpDir, data)
	
	if err != nil {
		t.Errorf("collectSensorData should not return error: %v", err)
	}
	
	// Check that valid sensor data was collected
	expectedSensors := []string{"temp1", "fan1", "in0"}
	for _, sensor := range expectedSensors {
		if _, ok := data[sensor]; !ok {
			t.Errorf("Expected sensor %s not found in collected data", sensor)
		}
	}
}

func TestCollectSensorDataNonExistentDir(t *testing.T) {
	data := make(map[string]map[string]string)
	err := collectSensorData("/non/existent/directory", data)
	
	if err == nil {
		t.Error("collectSensorData should return error for non-existent directory")
	}
}

func TestHwmonCollectorCollect(t *testing.T) {
	collector := NewHwmonCollector()
	if collector == nil {
		t.Fatal("NewHwmonCollector returned nil")
	}
	
	// Create a channel to collect metrics
	ch := make(chan prometheus.Metric, 100)
	
	// This should not panic even if hwmon is not available
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Collect should not panic: %v", r)
			}
		}()
		collector.Collect(ch)
	}()
	
	close(ch)
	
	// Count collected metrics
	count := 0
	for range ch {
		count++
	}
	
	// We don't assert on the exact count because it depends on system capabilities
	// Just ensure it doesn't crash
	t.Logf("Collected %d hwmon metrics", count)
}

func TestHwmonSensorTypes(t *testing.T) {
	// Test that hwmonSensorTypes contains expected types
	expectedTypes := []string{"temp", "fan", "in", "curr", "power"}
	
	for _, expectedType := range expectedTypes {
		found := false
		for _, sensorType := range hwmonSensorTypes {
			if sensorType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected sensor type %q not found in hwmonSensorTypes", expectedType)
		}
	}
	
	if len(hwmonSensorTypes) == 0 {
		t.Error("hwmonSensorTypes should not be empty")
	}
}

func TestHwmonRegexes(t *testing.T) {
	// Test hwmonInvalidMetricChars regex
	if hwmonInvalidMetricChars == nil {
		t.Error("hwmonInvalidMetricChars should not be nil")
	}
	
	// Test hwmonFilenameFormat regex
	if hwmonFilenameFormat == nil {
		t.Error("hwmonFilenameFormat should not be nil")
	}
	
	// Test that the filename regex works with expected patterns
	testCases := []string{"temp1_input", "fan2", "beep_enable", "vrm"}
	for _, testCase := range testCases {
		if !hwmonFilenameFormat.MatchString(testCase) {
			t.Errorf("hwmonFilenameFormat should match %q", testCase)
		}
	}
}
// Part 2 commit for node_hardware_exporter/internal/metrics/hwmon_test.go
