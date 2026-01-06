package metrics

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestInitPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tmpDir   string
		expected struct {
			json  string
			stats string
			data  string
		}
	}{
		{
			name:   "default tmp dir",
			tmpDir: "/tmp",
			expected: struct {
				json  string
				stats string
				data  string
			}{
				json:  "/tmp/keepalived.json",
				stats: "/tmp/keepalived.stats",
				data:  "/tmp/keepalived.data",
			},
		},
		{
			name:   "custom tmp dir",
			tmpDir: "/custom-tmp",
			expected: struct {
				json  string
				stats string
				data  string
			}{
				json:  "/custom-tmp/keepalived.json",
				stats: "/custom-tmp/keepalived.stats",
				data:  "/custom-tmp/keepalived.data",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			k := KeepalivedContainerCollectorHost{}
			k.initPaths(tt.tmpDir)

			assert.Equal(t, tt.expected.json, k.jsonPath)
			assert.Equal(t, tt.expected.stats, k.statsPath)
			assert.Equal(t, tt.expected.data, k.dataPath)
		})
	}
}

func TestContainerHasVRRPScriptStateSupport(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		version         string
		expectedSupport bool
	}{
		{name: "nil version", version: "", expectedSupport: true},
		{name: "supported version", version: "1.4.0", expectedSupport: true},
		{name: "unsupported version", version: "1.3.5", expectedSupport: false},
		{name: "newer version", version: "2.0.0", expectedSupport: true},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var v *version.Version
			if tc.version != "" {
				v = version.Must(version.NewVersion(tc.version))
			}

			c := KeepalivedContainerCollectorHost{version: v}
			assert.Equal(t, tc.expectedSupport, c.HasVRRPScriptStateSupport())
		})
	}
}
