package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

// warning is a special form of a metric and suitable for reporting non-fatal
// errors during a scrape. Warnings are only logged and not forwarded to the
// registry.
type warning struct {
	err error
}

var _ prometheus.Metric = (*warning)(nil)


// TODO: implement functions
