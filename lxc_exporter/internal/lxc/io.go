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


// TODO: implement
