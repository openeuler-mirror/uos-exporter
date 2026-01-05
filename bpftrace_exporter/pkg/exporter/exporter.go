package exporter

import (
	"encoding/json"
	"fmt"
	"log"
	"path"
	"strings"
	"sync"

	"github.com/andreasgerstmayr/bpftrace_exporter/pkg/bpftrace"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "bpftrace"
)

type Exporter struct {
	mutex         sync.RWMutex
	numProbesDesc *prometheus.Desc

	process    *bpftrace.Process
	scriptName string
	vars       map[string]*VarDef
}

type VarDef struct {
	VarType  int
	IsMap    bool
	Desc     *prometheus.Desc
	PromType prometheus.ValueType
}


// TODO: implement functions
