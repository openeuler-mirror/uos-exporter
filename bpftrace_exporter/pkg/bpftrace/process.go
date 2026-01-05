package bpftrace

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type Process struct {
	cmd           *exec.Cmd
	StdoutScanner *bufio.Scanner
	NumProbes     int
}


// TODO: implement functions
