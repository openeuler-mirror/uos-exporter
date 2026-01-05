package cib

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCibAdminParser_Parse(t *testing.T) {
	tests := []struct {
		name          string
		cibAdminPath  string
		expectedError string
	}{
		{
			name:          "invalid cibadmin path",
			cibAdminPath:  "/nonexistent/cibadmin",
			expectedError: "error while executing cibadmin",
		},
		{
			name:         "mock successful cibadmin",
			cibAdminPath: createMockCibAdmin(t),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewCibAdminParser(tt.cibAdminPath)
			result, err := parser.Parse()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				// 验证解析结果
				assert.NotEmpty(t, result)
				// 验证一些基本字段
				assert.Len(t, result.Configuration.Resources.Primitives, 1)
				assert.Equal(t, "test-primitive", result.Configuration.Resources.Primitives[0].Id)
			}
		})
	}
}

func createMockCibAdmin(t *testing.T) string {
	content := `#!/bin/sh
cat << 'EOF'
<?xml version="1.0" ?>
<cib>
  <configuration>
    <resources>
      <primitive id="test-primitive" class="ocf" provider="heartbeat" type="IPaddr2">
        <operations>
          <op id="test-primitive-monitor" name="monitor" interval="10s"/>
        </operations>
      </primitive>
    </resources>
  </configuration>
</cib>
EOF`

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "mock-cibadmin")
	err := os.WriteFile(scriptPath, []byte(content), 0755)
	if err != nil {
		t.Fatalf("Failed to create mock script: %v", err)
	}
	return scriptPath
}
