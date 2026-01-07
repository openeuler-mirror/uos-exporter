package metrics

import (
	"log"

	"github.com/prometheus/client_golang/prometheus"
)

type PvSegExporter struct {
	pvseg_start *prometheus.Desc
	pvseg_size  *prometheus.Desc
}

var pvsegnamespace = "pvseg"


// TODO: implement functions
