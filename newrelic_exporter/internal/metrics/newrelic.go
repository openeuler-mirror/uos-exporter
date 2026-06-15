package metrics

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// cryptoRandReader implements io.Reader using crypto/rand for secure random data
type cryptoRandReader struct{}

func (r *cryptoRandReader) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

// generateSecureRandomFloat64 generates a secure random float64 between min and max

// TODO: implement functions
