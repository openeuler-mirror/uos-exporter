package metrics

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var timeParse = time.Parse

type errorReader struct {
	err error
}

func createTempFile(t *testing.T, content string, suffix string) string {
	t.Helper()
	pattern := "test-*"
	if suffix != "" {
		pattern += suffix
	}
	
	file, err := os.CreateTemp("", pattern)
	require.NoError(t, err)
	defer file.Close()

	_, err = file.WriteString(content)
	require.NoError(t, err)

	return file.Name()
}

func createTestCollector(t *testing.T, files []string) *OSReleaseCollector {
	t.Helper()
	c := &OSReleaseCollector{
		osReleaseFiles:  files,
		refreshInterval: 1 * time.Millisecond,
		logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	c.initializeDescriptors()
	return c
}

func mockTimeParse(layout, value string) (time.Time, error) {
	if value == "invalid" {
		return time.Time{}, errors.New("invalid date")
	}
	return time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), nil
}

func TestOSReleaseCollector_LoadOSData(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		fileType    string
		expected    OSRelease
		expectError bool
	}{
		{
			name:     "env file basic",
			fileType: "env",
			content: `NAME="TestOS"
ID=testos
VERSION="1.2.3"
VERSION_ID="1.2.3"`,
			expected: OSRelease{
				Name:      "TestOS",
				ID:        "testos",
				Version:   "1.2.3",
				VersionID: "1.2.3",
			},
		},
		{
			name:     "env file with quotes",
			fileType: "env",
			content: `NAME="Pretty OS"
PRETTY_NAME="Pretty OS"
VERSION="Version 1.0"
HOME_URL="https://example.com"`,
			expected: OSRelease{
				Name:       "Pretty OS",
				PrettyName: "Pretty OS",
				Version:    "Version 1.0",
				HomeURL:    "https://example.com",
			},
		},
		{
			name:     "plist file",
			fileType: "plist",
			content: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ProductName</key>
	<string>MacOS</string>
	<key>ProductVersion</key>
	<string>14.0</string>
	<key>ProductBuildVersion</key>
	<string>23A344</string>
</dict>
</plist>`,
			expected: OSRelease{
				Name:      "MacOS",
				Version:   "14.0",
				VersionID: "14.0",
				BuildID:   "23A344",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suffix := ""
			switch tt.fileType {
			case "env":
				suffix = "os-release"
			case "plist":
				suffix = "SystemVersion.plist"
			}
			
			filename := createTempFile(t, tt.content, suffix)
			defer os.Remove(filename)

			c := createTestCollector(t, []string{filename})
			err := c.loadOSData()
			
			if tt.expectError {
				require.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			require.NotNil(t, c.osData, "osData should not be nil")

			c.dataMutex.RLock()
			defer c.dataMutex.RUnlock()

			assert.Equal(t, tt.expected.Name, c.osData.Name)
			assert.Equal(t, tt.expected.Version, c.osData.Version)
		})
	}

	t.Run("file not found", func(t *testing.T) {
		c := createTestCollector(t, []string{"/non/existent/file"})
		err := c.loadOSData()
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrNoDataFound) || strings.Contains(err.Error(), "all attempts"))
	})

	t.Run("unsupported file type", func(t *testing.T) {
		filename := createTempFile(t, "invalid content", "unknown.txt")
		defer os.Remove(filename)

		c := createTestCollector(t, []string{filename})
		c.osReleaseFiles = []string{filename} 
		err := c.loadOSData()
		require.Error(t, err)
		assert.Equal(t, ErrUnsupportedFileFormat, errors.Unwrap(err))
	})
}

func TestOSReleaseCollector_RefreshOSData(t *testing.T) {
	t.Run("refresh when stale", func(t *testing.T) {
		filename := createTempFile(t, `NAME="Initial"`, "os-release")
		defer os.Remove(filename)

		c := createTestCollector(t, []string{filename})
		c.refreshInterval = 10 * time.Millisecond
		require.NoError(t, c.loadOSData())
		newContent := `NAME="Updated"
ID=updated
VERSION="2.0"`
		require.NoError(t, os.WriteFile(filename, []byte(newContent), 0644))

		time.Sleep(50 * time.Millisecond)

		err := c.refreshOSData()
		require.NoError(t, err)

		c.dataMutex.RLock()
		defer c.dataMutex.RUnlock()
		assert.Equal(t, "Updated", c.osData.Name)
	})

	t.Run("no refresh when not stale", func(t *testing.T) {
		filename := createTempFile(t, `NAME="Initial"`, "os-release")
		defer os.Remove(filename)

		c := createTestCollector(t, []string{filename})
		c.refreshInterval = time.Hour 
		require.NoError(t, c.loadOSData())

		newContent := `NAME="Updated"
ID=updated
VERSION="2.0"`
		require.NoError(t, os.WriteFile(filename, []byte(newContent), 0644))
		ch := make(chan prometheus.Metric, 10)
		go c.Collect(ch)
		time.Sleep(10 * time.Millisecond)

		c.dataMutex.RLock()
		defer c.dataMutex.RUnlock()
		assert.Equal(t, "Initial", c.osData.Name)
	})
}

func TestOSReleaseCollector_Update(t *testing.T) {
	t.Run("with valid data", func(t *testing.T) {
		filename := createTempFile(t, `NAME="TestOS"
ID=testos
VERSION="1.0"
VERSION_ID="1.0.0"
BUILD_ID="20240101"
HOME_URL="https://testos.example"
BUG_REPORT_URL="https://bugs.testos.example"
PLATFORM_ID="testos:1.0"
ID_LIKE="debian"
IMAGE_ID="testos-image"
IMAGE_VERSION="1.0"
PRETTY_NAME="TestOS 1.0"
VARIANT="Server"
VARIANT_ID="server"
VERSION_CODENAME="stable"
SUPPORT_END="2025-12-31"`, "os-release")
		defer os.Remove(filename)

		c := createTestCollector(t, []string{filename})
		require.NoError(t, c.loadOSData())

		ch := make(chan prometheus.Metric, 5)
		err := c.Update(ch)
		require.NoError(t, err)
		close(ch)

		count := 0
		for range ch {
			count++
		}

		assert.Equal(t, 3, count, "should produce three metrics")
	})

	t.Run("with no data", func(t *testing.T) {
		c := createTestCollector(t, []string{})
		c.osData = nil

		ch := make(chan prometheus.Metric)
		err := c.Update(ch)
		require.Error(t, err)
		assert.Equal(t, ErrNoDataFound, err)
	})

	t.Run("version extraction", func(t *testing.T) {
		tests := []struct {
			versionID string
			expected  float64
		}{
			{"1.2.3", 1.2},
			{"5", 5.0},
			{"invalid", 0},
			{"", 0},
		}

		for _, tt := range tests {
			t.Run(tt.versionID, func(t *testing.T) {
				c := createTestCollector(t, []string{})
				c.osData = &OSRelease{VersionID: tt.versionID}
				c.extractVersion()
				assert.Equal(t, tt.expected, c.versionValue)
			})
		}
	})

	t.Run("support end time parsing", func(t *testing.T) {
		originalTimeParse := timeParse
		defer func() { timeParse = originalTimeParse }()

		timeParse = mockTimeParse

		tests := []struct {
			date     string
			expected time.Time
		}{
			{"2025-12-31", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},
			{"invalid", time.Time{}},
			{"", time.Time{}},
		}

		for _, tt := range tests {
			t.Run(tt.date, func(t *testing.T) {
				c := createTestCollector(t, []string{})
				c.osData = &OSRelease{SupportEnd: tt.date}
				c.parseSupportEndTime()
				assert.Equal(t, tt.expected, c.supportEndTime)
			})
		}
	})
}

func TestEnvFileParser(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected OSRelease
	}{
		{
			name: "basic values",
			content: `NAME="TestOS"
ID=testos
VERSION_ID=1.0`,
			expected: OSRelease{
				Name:      "TestOS",
				ID:        "testos",
				VersionID: "1.0",
			},
		},
		{
			name: "quoted values",
			content: `PRETTY_NAME="Pretty OS"
VERSION="Version 1.0"
HOME_URL="https://example.com"`,
			expected: OSRelease{
				PrettyName: "Pretty OS",
				Version:    "Version 1.0",
				HomeURL:    "https://example.com",
			},
		},
		{
			name: "mixed quoted and unquoted",
			content: `ID=testos
VERSION_ID="1.0"
BUILD_ID=20240101`,
			expected: OSRelease{
				ID:        "testos",
				VersionID: "1.0",
				BuildID:   "20240101",
			},
		},
		{
			name: "empty values",
			content: `NAME=""
ID=
VERSION_ID=`,
			expected: OSRelease{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &EnvFileParser{}
			reader := strings.NewReader(tt.content)
			result, err := parser.Parse(reader)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, *result)
		})
	}

	t.Run("scanner error", func(t *testing.T) {
		errorReader := &errorReader{err: errors.New("read error")}
		parser := &EnvFileParser{}
		_, err := parser.Parse(errorReader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "scanning failed")
	})
}

func TestPlistFileParser(t *testing.T) {
	t.Run("valid plist", func(t *testing.T) {
		content := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ProductName</key>
	<string>MacOS</string>
	<key>ProductVersion</key>
	<string>14.0</string>
	<key>ProductBuildVersion</key>
	<string>23A344</string>
</dict>
</plist>`
		parser := &PlistFileParser{}
		reader := strings.NewReader(content)
		result, err := parser.Parse(reader)
		require.NoError(t, err)
		assert.Equal(t, "MacOS", result.Name)
		assert.Equal(t, "14.0", result.Version)
		assert.Equal(t, "23A344", result.BuildID)
	})

	t.Run("invalid xml", func(t *testing.T) {
		content := `invalid xml content`
		parser := &PlistFileParser{}
		reader := strings.NewReader(content)
		_, err := parser.Parse(reader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "XML unmarshalling failed")
	})

	t.Run("read error", func(t *testing.T) {
		errorReader := &errorReader{err: errors.New("read error")}
		parser := &PlistFileParser{}
		_, err := parser.Parse(errorReader)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading failed")
	})
}

func TestAccessorMethods(t *testing.T) {
	c := createTestCollector(t, []string{})
	c.osData = &OSRelease{Name: "TestOS", Version: "1.0"}
	c.versionValue = 1.0
	c.supportEndTime = time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, "TestOS", c.GetOSName())
	assert.Equal(t, "1.0", c.GetOSVersion())
	assert.Equal(t, 1.0, c.GetOSVersionValue())
	assert.Equal(t, time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), c.GetSupportEndTime())
}

func TestConcurrentAccess(t *testing.T) {
	filename := createTempFile(t, `NAME="TestOS"`, "os-release")
	defer os.Remove(filename)

	c := createTestCollector(t, []string{filename})
	require.NoError(t, c.loadOSData())

	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		ch := make(chan prometheus.Metric, 10)
		c.Collect(ch)
	}()

	go func() {
		defer wg.Done()
		ch := make(chan *prometheus.Desc, 5)
		c.Describe(ch)
	}()

	go func() {
		defer wg.Done()
		_ = c.GetOSName()
	}()

	go func() {
		defer wg.Done()
		c.dataMutex.Lock()
		c.osData = &OSRelease{Name: "Updated"}
		c.dataMutex.Unlock()
	}()

	wg.Wait()
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}