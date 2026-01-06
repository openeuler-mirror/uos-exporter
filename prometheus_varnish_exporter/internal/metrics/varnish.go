package metrics

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"prometheus_varnish_exporter/pkg/utils"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	vbeReloadPrefix = "VBE.reload_"
)

var (
	DescCache = &descCache{
		descs: make(map[string]*prometheus.Desc),
	}
)

type descCache struct {
	sync.RWMutex

	descs map[string]*prometheus.Desc
}

func (dc *descCache) Desc(key string) *prometheus.Desc {
	dc.RLock()
	desc := dc.descs[key]
	dc.RUnlock()
	return desc
}

func (dc *descCache) Set(key string, desc *prometheus.Desc) *prometheus.Desc {
	dc.Lock()
	dc.descs[key] = desc
	dc.Unlock()
	return desc
}


// TODO: implement functions
