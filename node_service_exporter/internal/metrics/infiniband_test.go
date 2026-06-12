package metrics

import (
	"testing"
)

func TestInfiniBandCollectors(t *testing.T) {
	collectors := []interface{}{
		NewInfiniBandInfo(),
		NewInfiniBandStateID(),
		NewInfiniBandPhysicalStateID(),
	}
	for _, c := range collectors {
		if c == nil {
			t.Error("collector is nil")
		}
	}
}
