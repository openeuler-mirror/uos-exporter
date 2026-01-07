package logger

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestFileRotator(t *testing.T) {
	basePath := "test.log"
	maxSize := int64(100) // 100 bytes
	maxAge := time.Second * 10

	fr := NewFileRotator(basePath, maxSize, maxAge)

	data := []byte("Test data that should cause a rotate")
	for i := 0; i < 10; i++ {
		n, err := fr.Write(data)
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
		if n != len(data) {
			t.Errorf("Write wrote %d bytes; want %d", n, len(data))
		}
	}

	newFiles, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	logFiles := false
	for _, file := range newFiles {
		if file.IsDir() {
			continue
		}
		if file.Name() == basePath || strings.HasPrefix(file.Name(), basePath+".20") {
			logFiles = true
			break
		}
	}

	if !logFiles {
		t.Errorf("No log files created")
	}

	for _, file := range newFiles {
		if file.IsDir() {
			continue
		}
		if file.Name() == basePath || strings.HasPrefix(file.Name(), basePath+".") {
			os.Remove(file.Name())
		}
	}
}
// Part 2 commit for tc_exporter/pkg/logger/filerotator_test.go
