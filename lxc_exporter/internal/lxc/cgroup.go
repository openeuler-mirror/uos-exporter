package lxc

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	CgroupStatFile = "cgroup.stat"
)

type CgroupStat struct {
	NrDescendants      float64
	NrDyingDescendants float64
}

func (l *Lxc) GetCgroupStat(containerName string) (CgroupStat, error) {
	if !l.containerExists(containerName) {
		return CgroupStat{}, ErrorContainerNotFound
	}
	statContent, err := l.readMemoryStatFile(containerName)
	if err != nil {
		return CgroupStat{}, err
	}
	return parseCgroupStat(statContent)
}
func (l *Lxc) GetCgroupStatAll() ([]CgroupStat, error) {
	css := make([]CgroupStat, 0)
	for _, containerName := range l.containerPaths {
		stat, err := l.GetCgroupStat(containerName)
		if err != nil {
			logrus.Warn("read cgroup stat error:", err)
			continue
		}
		css = append(css, stat)
	}
	return css, nil
}

func (l *Lxc) readCgroupStatFile(containerName string) ([]byte, error) {
	cgroupStatPath, err := l.getMemoryStatFilePath(containerName)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(cgroupStatPath)
	if !strings.HasPrefix(cleanPath, cgroupPath) {
		return nil, fmt.Errorf("config file must be located within %s", cgroupPath)
	}
	return os.ReadFile(cleanPath)
}

func (l *Lxc) getCgroupStatFilePath(containerName string) (string, error) {
	cgroupStatPath := path.Join(cgroupPath,
		//lxcPrefix,
		l.getContainerPathName(containerName),
		CgroupStatFile)
	_, err := os.Stat(cgroupStatPath)
	if err != nil {
		return "", err
	}
	return cgroupStatPath, nil
}

func parseCgroupStat(content []byte) (CgroupStat, error) {
	var (
		nrDescendants      float64
		nrDyingDescendants float64
	)
	lines := strings.Split(
		strings.TrimSpace(
			string(content)), "\n")
	if len(lines) < 2 {
		return CgroupStat{},
			errors.New("invalid cgroup.stat format")
	}
	nrDescendants, err := parseCgroupStatLine(lines[0])
	if err != nil {
		return CgroupStat{}, err
	}
	nrDyingDescendants, err = parseCPUStatLine(lines[1])
	if err != nil {
		return CgroupStat{}, err
	}

	return CgroupStat{
			NrDescendants:      nrDescendants,
			NrDyingDescendants: nrDyingDescendants,
		},
		nil
}

func parseCgroupStatLine(line string) (float64, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return 0,
			errors.New("invalid cpu.stat line format")
	}
	return parseFloat64(fields[1])
}
