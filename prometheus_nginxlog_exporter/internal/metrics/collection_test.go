package metrics

import (
	"testing"
)

func TestUniqueRelabelings(t *testing.T) {
	tests := []struct {
		name        string
		input       []*Relabeling
		expected    []*Relabeling
		expectError bool
	}{
		{
			name: "nil input",
			input: nil,
			expectError: true,
		},
		{
			name: "empty slice",
			input: []*Relabeling{},
			expected: []*Relabeling{},
			expectError: false,
		},
		{
			name: "single relabeling",
			input: []*Relabeling{
				NewTestRelabeling("label1", "value1"),
			},
			expected: []*Relabeling{
				NewTestRelabeling("label1", "value1"),
			},
			expectError: false,
		},
		{
			name: "duplicate relabelings",
			input: []*Relabeling{
				NewTestRelabeling("label1", "value1"),
				NewTestRelabeling("label1", "value2"),
				NewTestRelabeling("label2", "value3"),
				NewTestRelabeling("label1", "value4"),
			},
			expected: []*Relabeling{
				NewTestRelabeling("label1", "value1"),
				NewTestRelabeling("label2", "value3"),
			},
			expectError: false,
		},
		{
			name: "with nil relabeling",
			input: []*Relabeling{
				NewTestRelabeling("label1", "value1"),
				nil,
				NewTestRelabeling("label2", "value2"),
			},
			expectError: true,
		},
		{
			name: "with empty target label",
			input: []*Relabeling{
				NewTestRelabeling("", "value1"),
				NewTestRelabeling("label2", "value2"),
			},
			expectError: true,
		},
		{
			name: "case insensitive comparison",
			input: []*Relabeling{
				NewTestRelabeling("LABEL1", "value1"),
				NewTestRelabeling("label1", "value2"),
				NewTestRelabeling("Label2", "value3"),
			},
			expected: []*Relabeling{
				NewTestRelabeling("LABEL1", "value1"),
				NewTestRelabeling("Label2", "value3"),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UniqueRelabelings(tt.input)
			
			if (err != nil) != tt.expectError {
				t.Errorf("UniqueRelabelings() error = %v, expectError %v", err, tt.expectError)
				return
			}
			
			if !tt.expectError {
				if len(result) != len(tt.expected) {
					t.Errorf("UniqueRelabelings() result length = %d, expected %d", len(result), len(tt.expected))
					return
				}
				
				for i := range result {
					if result[i].Config().TargetLabel != tt.expected[i].Config().TargetLabel {
						t.Errorf("UniqueRelabelings() result[%d] = %v, expected %v", 
							i, result[i].Config().TargetLabel, tt.expected[i].Config().TargetLabel)
					}
				}
			}
		})
	}
}

// Helper function to create test Relabeling instances
func NewTestRelabeling(targetLabel, sourceValue string) *Relabeling {
	return &Relabeling{
		config: RelabelConfig{
			TargetLabel: targetLabel,
			SourceValue: sourceValue,
		},
	}
}

func TestUniqueRelabelings(t *testing.T) {

    testCases := []struct {
        name     string
        input    []*Relabeling
        expected []*Relabeling
    }{
        {
            name:     "empty input",
            input:    nil,
            expected: nil,
        },
        {
            name: "complete repetition",
            input: []*Relabeling{
                {TargetLabel: "label1"},
                {TargetLabel: "label1"},
                {TargetLabel: "label1"},
            },
            expected: []*Relabeling{
                {TargetLabel: "label1"},
            },
        },
        {
            name: "mixed case",
            input: []*Relabeling{
                {TargetLabel: "Label1"},
                {TargetLabel: "label1"},
                {TargetLabel: "LABEL1"},
            },
            expected: []*Relabeling{
                {TargetLabel: "Label1"},
            },
        },
        {
            name: "partial repetition",
            input: []*Relabeling{
                {TargetLabel: "label1"},
                {TargetLabel: "label2"},
                {TargetLabel: "label1"},
                {TargetLabel: "label3"},
            },
            expected: []*Relabeling{
                {TargetLabel: "label1"},
                {TargetLabel: "label2"},
                {TargetLabel: "label3"},
            },
        },
        {
            name: "retain the order",
            input: []*Relabeling{
                {TargetLabel: "label2"},
                {TargetLabel: "label1"},
                {TargetLabel: "label2"},
                {TargetLabel: "label3"},
            },
            expected: []*Relabeling{
                {TargetLabel: "label2"},
                {TargetLabel: "label1"},
                {TargetLabel: "label3"},
            },
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            actual := UniqueRelabelings(tc.input)
            assert.Equal(t, len(tc.expected), len(actual), "the lengths of the results are inconsistent")
            
            for i := range actual {
                assert.Equal(t, tc.expected[i].TargetLabel, actual[i].TargetLabel,
                    "the %d element does not match", i+1)
            }
        })
    }
}

func BenchmarkUniqueRelabelings(b *testing.B) {
    data := make([]*Relabeling, 1000)
    for i := range data {
        data[i] = &Relabeling{
            TargetLabel: fmt.Sprintf("label%d", i%100),
        }
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _ = UniqueRelabelings(data)
    }
}
// Part 2 commit for prometheus_nginxlog_exporter/internal/metrics/collection_test.go
