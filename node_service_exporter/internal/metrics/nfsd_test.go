package metrics

import (
	"testing"
)

func TestNFSdCollectors(t *testing.T) {
	collectors := []interface{}{
		NewNFSdReplyCacheHits(),
		NewNFSdReplyCacheMisses(),
		NewNFSdFileHandlesStale(),
	}
	for _, c := range collectors {
		if c == nil {
			t.Error("collector is nil")
		}
	}
}
