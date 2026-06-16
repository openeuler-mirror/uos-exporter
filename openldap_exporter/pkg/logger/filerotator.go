package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"openldap_exporter/pkg/utils"
)

var defaultMaxFiles = 5

type FileRotator struct {
	basePath  string
	maxSize   int64
	maxAge    time.Duration
	current   *os.File
	size      int64
	startTime time.Time
	keepFiles int
}

func NewFileRotator(basePath string, maxSize int64, maxAge time.Duration) *FileRotator {
	dir := filepath.Dir(basePath)
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0o750)
	}
	return &FileRotator{
		basePath:  basePath,
		maxSize:   maxSize,
		maxAge:    maxAge,
		keepFiles: defaultMaxFiles,
	}
}

func (fr *FileRotator) Write(p []byte) (n int, err error) {
	err = fr.setupCurrent()
	if err != nil {
		return 0, err
	}
	if fr.shouldRotate() {
		err = fr.rotate()
		if err != nil {
			return 0, err
		}
	}
	n, err = fr.current.Write(p)
	if err != nil {
		return n, err
	}
	fr.size += int64(n)
	return n, nil
}

func (fr *FileRotator) setupCurrent() error {
	if fr.current == nil {
		fileinfo, err := os.Stat(fr.basePath)
		if err == nil {
			fr.current, err = os.OpenFile(fr.basePath, os.O_APPEND|os.O_WRONLY, 0o600)
			if err != nil {
				return err
			}
			fr.size = fileinfo.Size()
			fr.startTime = fileinfo.ModTime()
		} else if os.IsNotExist(err) {
			fr.current, err = os.Create(fr.basePath)
			if err != nil {
				return err
			}
			fr.startTime = time.Now()
		} else {
			return err
		}
	}
	return nil
}

func (fr *FileRotator) Close() error {
	if fr.current != nil {
		return fr.current.Close()
	}
	return nil
}

func (fr *FileRotator) shouldRotate() bool {
	if fr.size > fr.maxSize || time.Now().Sub(fr.startTime) > fr.maxAge {
		return true
	}
	return false
}

func (fr *FileRotator) rotate() error {
	if fr.current != nil {
		err := fr.current.Close()
		if err != nil {
			return err
		}
	}

	for i := fr.keepFiles - 1; i > 0; i-- {
		if !utils.FileExists(fr.getLogPath(i)) {
			continue
		}
		if i == fr.keepFiles-1 {
			err := os.Remove(fr.getLogPath(i))
			if err != nil {
				return err
			}
			continue
		}
		err := os.Rename(fr.getLogPath(i), fr.getLogPath(i+1))
		if err != nil {
			return err
		}
	}
	err := os.Rename(fr.basePath, fr.getLogPath(1))
	if err != nil {
		return err
	}
	fr.current, err = os.Create(fr.basePath)
	if err != nil {
		return err
	}
	fr.size = 0
	fr.startTime = time.Now()
	return nil
}

func (fr *FileRotator) getLogPath(number int) string {
	return fmt.Sprintf("%s.%d", fr.basePath, number)
}
