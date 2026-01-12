// +build !test

package metrics

import (
	"fmt"
	"os"
	"strings"
	//_ "net/http/pprof"

	"frr_exporter/internal/exporter"
	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/common/promslog/flag"
	versioncollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/common/promslog"
	// "github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
)

func isTest() bool {
    return strings.HasSuffix(os.Args[0], ".test")
}

func init() {
	promslogConfig := &promslog.Config{}

	if !isTest() {
    
		flag.AddFlags(kingpin.CommandLine, promslogConfig)
        kingpin.Version(version.Print("frr_exporter"))
        kingpin.HelpFlag.Short('h')
        kingpin.Parse()
    }

	logger := promslog.New(promslogConfig)

	exporter.Register(versioncollector.NewCollector("frr_exporter"))

	nc, err := NewFrrExporter(logger)
	if err != nil {
		panic(fmt.Errorf("could not create collector: %w", err))
	}

	exporter.Register(nc)
}
