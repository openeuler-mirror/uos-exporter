package lxc

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ioStatFile = "io.stat"
)

type IoStat struct {
	Rbytes float64
	Wbytes float64
	Rios   float64
	Wios   float64
	Dbytes float64
	Dios   float64
}

func (l *Lxc) GetIoStat(containerName string) (IoStat, error) {
	if !l.containerExists(containerName) {
		return IoStat{}, ErrorContainerNotFound
	}
	statContent, err := l.readIoStatFile(containerName)
	if err != nil {
		return IoStat{}, err
	}
	return parseIoStat(statContent)
}
func (l *Lxc) GetIoStatAll() ([]IoStat, error) {
	return nil, nil
}

func (l *Lxc) readIoStatFile(containerName string) ([]byte, error) {
	cgroupStatPath, err := l.getIoStatFilePath(containerName)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(cgroupStatPath)
	if !strings.HasPrefix(cleanPath, cgroupPath) {
		return nil, fmt.Errorf("config file must be located within %s", cgroupPath)
	}
	return os.ReadFile(cleanPath)
}

func (l *Lxc) getIoStatFilePath(containerName string) (string, error) {
	ioStatPath := path.Join(cgroupPath,
		//lxcPrefix,
		l.getContainerPathName(containerName),
		ioStatFile)
	_, err := os.Stat(ioStatPath)
	if err != nil {
		return "", err
	}
	return ioStatPath, nil
}

func parseIoStat(content []byte) (IoStat, error) {
	var (
		rbytes float64
		wbytes float64
		rios   float64
		wios   float64
		dbytes float64
		dios   float64
	)

	line := strings.TrimSpace(
		string(content))
	fields := strings.Fields(line)
	if len(fields) != 7 {
		return IoStat{},
			errors.New("invalid io.stat line format")
	}
	rbytes, err := parseIoFields(fields[1])
	if err != nil {
		return IoStat{}, err
	}
	wbytes, err = parseIoFields(fields[2])
	if err != nil {
		return IoStat{}, err
	}
	rios, err = parseIoFields(fields[3])
	if err != nil {
		return IoStat{}, err
	}
	wios, err = parseIoFields(fields[4])
	if err != nil {
		return IoStat{}, err
	}
	dbytes, err = parseIoFields(fields[5])
	if err != nil {
		return IoStat{}, err
	}
	dios, err = parseIoFields(fields[6])
	if err != nil {
		return IoStat{}, err
	}
	return IoStat{
		Rbytes: rbytes,
		Wbytes: wbytes,
		Rios:   rios,
		Wios:   wios,
		Dbytes: dbytes,
		Dios:   dios,
	}, nil
}

func parseIoFields(s string) (float64, error) {
	iofield := strings.Split(s, "=")
	if len(iofield) != 2 {
		return 0, errors.New("invalid io.stat field")
	}
	value := iofield[1]
	return strconv.ParseFloat(value, 64)
}
