package drbd

import (
	"fmt"
	"hacluster_exporter/internal/metrics/collectors/core"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


// TODO: implement functions
