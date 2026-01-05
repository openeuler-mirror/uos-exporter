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


// TODO: implement functions
