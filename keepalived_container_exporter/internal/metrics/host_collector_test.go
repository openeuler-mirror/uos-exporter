package metrics

import (
	"bytes"
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestHostHasVRRPScriptStateSupport(t *testing.T) {
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

			c := KeepalivedHostCollectorHost{version: v}
			assert.Equal(t, tc.expectedSupport, c.HasVRRPScriptStateSupport(),
				"版本 %s 的脚本状态支持结果不符合预期", tc.version)
		})
	}
}

func TestParseSigNum(t *testing.T) {
	t.Parallel()

	signum := bytes.NewBufferString("10\n")
	sigNumInt := parseSigNum(*signum, "DATA")

	if sigNumInt != 10 {
		t.Fail()
	}
}
