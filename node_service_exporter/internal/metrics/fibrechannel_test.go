package metrics

import (
	"testing"
)

func TestFibreChannelCollectors(t *testing.T) {
	collectors := []interface{}{
		NewFibreChannelInfo(),
		NewFibreChannelDumpedFrames(),
		NewFibreChannelLossOfSignal(),
	}
	for _, c := range collectors {
		if c == nil {
			t.Error("collector is nil")
		}
	}
}
