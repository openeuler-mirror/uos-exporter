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


// TODO: implement
