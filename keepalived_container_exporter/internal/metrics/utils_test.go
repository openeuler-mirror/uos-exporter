package metrics

import (
	"syscall"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestHasSigNumSupport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{name: "nil version", version: "", expected: true},
		{name: "supported version", version: "1.3.8", expected: true},
		{name: "unsupported version", version: "1.3.5", expected: false},
		{name: "newer version", version: "2.0.0", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var v *version.Version
			if tt.version != "" {
				v = version.Must(version.NewVersion(tt.version))
			}

			assert.Equal(t, tt.expected, HasSigNumSupport(v),
				"版本 %s 的信号编号支持结果不符合预期", tt.version)
		})
	}
}

func TestGetDefaultSignal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		sigType  string
		expected syscall.Signal
	}{
		{name: "DATA signal", sigType: "DATA", expected: syscall.SIGUSR1},
		{name: "STATS signal", sigType: "STATS", expected: syscall.SIGUSR2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, GetDefaultSignal(tt.sigType),
				"信号类型 %s 的默认值不符合预期", tt.sigType)
		})
	}
}

func TestParseVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		versionOutput string
		expectError   bool
		expectedVer   string
	}{
		{
			name: "valid version output",
			versionOutput: `Keepalived v2.0.20 (05/04,2020)
				Copyright(C) 2001-2020 Alexandre Cassen`,
			expectError: false,
			expectedVer: "2.0.20",
		},
		{
			name:          "invalid format - no version",
			versionOutput: "Keepalived",
			expectError:   true,
		},
		{
			name:          "invalid format - malformed version",
			versionOutput: "Keepalived keepalived",
			expectError:   true,
		},
		{
			name: "short version string",
			versionOutput: `Keepalived v2.0.20
			test string`,
			expectError: false,
			expectedVer: "2.0.20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			v, err := ParseVersion(tt.versionOutput)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			expected := version.Must(version.NewVersion(tt.expectedVer))
			assert.Equal(t, 0, v.Compare(expected),
				"解析版本 %s 结果不符合预期", tt.expectedVer)
		})
	}
}

func TestHasVRRPScriptStateSupport(t *testing.T) {
	t.Parallel()

	testCaseses := []struct {
		name            string
		version         *version.Version
		expectedSupport bool
	}{
		{name: "nil", version: nil, expectedSupport: true},
		{name: "1.4.0", version: version.Must(version.NewVersion("1.4.0")), expectedSupport: true},
		{name: "1.3.5", version: version.Must(version.NewVersion("1.3.5")), expectedSupport: false},
	}

	for _, tc := range testCaseses {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if HasSigNumSupport(tc.version) != tc.expectedSupport {
				t.Fail()
			}
		})
	}
}
