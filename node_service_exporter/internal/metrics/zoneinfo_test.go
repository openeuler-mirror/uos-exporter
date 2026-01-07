package metrics

import (
	"testing"
)

func TestZoneinfoCollectorWrapper(t *testing.T) {
	wrapper := NewZoneinfoCollectorWrapper()
	if wrapper == nil {
		t.Fatal("wrapper is nil")
	}
	if wrapper.collector == nil {
		t.Fatal("collector is nil")
	}
}
