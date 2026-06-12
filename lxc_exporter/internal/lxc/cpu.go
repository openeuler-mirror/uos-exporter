package lxc

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	CPUStatFile = "cpu.stat"
)

type CPUStat struct {
	Usage         float64
	User          float64
	System        float64
	ForceIdle     float64
	NrPeriods     float64
	NrThrottled   float64
	ThrottledUsec float64
	Nrbursts      float64
	BurstUsec     float64
}


// TODO: implement
