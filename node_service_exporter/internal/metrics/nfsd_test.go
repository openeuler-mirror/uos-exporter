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
// Part 2 commit for node_service_exporter/internal/metrics/nfsd_test.go
