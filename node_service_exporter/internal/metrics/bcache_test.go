package metrics

import (
	"testing"
)

func TestBcacheCollectors(t *testing.T) {
	collectors := []interface{}{
		NewBcacheIOErrors(),
		NewBcacheMetadataWritten(),
		NewBcacheWritten(),
	}
	for _, c := range collectors {
		if c == nil {
			t.Error("collector is nil")
		}
	}
}
