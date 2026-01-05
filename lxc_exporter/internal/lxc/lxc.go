package lxc

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var (
	cgroupPath = "/sys/fs/cgroup/"
	lxcPrefix  = "lxc.payload"
	lxcLsCmd   = "lxc-ls"
)

var (
	ErrorContainerNotFound = errors.New("container not found")
)

type Lxc struct {
	containerPaths []string
}

func NewLxc() *Lxc {
	return &Lxc{}
}

func (l *Lxc) UpdateContainerNameAll() {
	list, err := l.getContainerNameAll()
	if err != nil {
		return
	}
	l.containerPaths = list
}

func (l *Lxc) containerExists(containerName string) bool {
	_, err := os.Stat(cgroupPath +
		l.getContainerPathName(
			containerName))
	return err == nil
}

func (l *Lxc) GetContainerNameAll() []string {
	return l.containerPaths
}

func (l *Lxc) getContainerNameAll() ([]string, error) {
	var paths []string
	//entries, err := os.ReadDir(cgroupPath)
	//if err != nil {
	//	return paths,
	//		err
	//}
	//for _, entry := range entries {
	//	if entry.IsDir() {
	//		if strings.HasPrefix(entry.Name(),
	//			lxcPrefix) {
	//			paths = append(paths,
	//				l.getContainerName(entry.Name()))
	//		}
	//	}
	//	//paths = append(paths, entry.Name())
	//}
	lxcLsRunArg := []string{"--running"}
	cmd := exec.Command(lxcLsCmd, lxcLsRunArg...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	//for _, line := range strings.Split(string(output), " ") {
	//	if line == "" {
	//		continue
	//	}
	//	paths = append(paths, line)
	//}
	paths = strings.Fields(string(output))
	return paths, nil
}

func (l Lxc) getContainerPathName(name string) string {
	//return fmt.Sprintf("%s.%s",
	//	lxcPrefix,
	//	name)
	return name
}
func (l Lxc) getContainerName(pathName string) string {
	result := strings.TrimPrefix(pathName,
		lxcPrefix+
			".")
	return result
}

func parseFloat64(str string) (float64, error) {
	return strconv.ParseFloat(str,
		64)
}
