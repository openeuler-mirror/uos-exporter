package metrics

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
	"bytes"

	"github.com/google/go-cmp/cmp"
	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type remoteVersionCollector struct {
	cl     clientInterface
	logger *zap.Logger
}

type clientInterface interface {
	GetRemoteVersion(ctx context.Context) (*client.RemoteVersion, *client.Response, error)
}

func newRemoteVersionCollector(cl clientInterface) *remoteVersionCollector {
	return &remoteVersionCollector{
		cl:     cl,
		logger: zaptest.NewLogger(&testing.T{}),
	}
}

func (c *remoteVersionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc(
		"paperless_remote_version_update_available",
		"Whether an update is available.",
		[]string{"version"}, nil,
	)
	ch <- prometheus.NewDesc(
		"paperless_warnings_total",
		"Number of warnings generated while scraping metrics.",
		[]string{"category"}, nil,
	)
}

func (c *remoteVersionCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.collect(ctx, ch); err != nil {
		c.logger.Error("Collection failed", zap.Error(err))
	}
}

func (c *remoteVersionCollector) collect(ctx context.Context, ch chan<- prometheus.Metric) error {
	version, _, err := c.cl.GetRemoteVersion(ctx)
	
	// Always report both metrics
	updateValue := 0.0
	versionStr := ""
	if version != nil {
		if version.UpdateAvailable {
			updateValue = 1.0
		}
		versionStr = version.Version
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			"paperless_remote_version_update_available",
			"Whether an update is available.",
			[]string{"version"}, nil,
		),
		prometheus.GaugeValue,
		updateValue,
		versionStr,
	)

	warningValue := 0.0
	if err != nil {
		warningValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			"paperless_warnings_total",
			"Number of warnings generated while scraping metrics.",
			[]string{"category"}, nil,
		),
		prometheus.GaugeValue,
		warningValue,
		"get_remote_version",
	)

	if err != nil {
		return fmt.Errorf("fetching remote version: %w", err)
	}

	return nil
}

type fakeRemoteVersionClient struct {
	result client.RemoteVersion
	err    error
}

func (c *fakeRemoteVersionClient) GetRemoteVersion(ctx context.Context) (*client.RemoteVersion, *client.Response, error) {
	return &c.result, &client.Response{}, c.err
}

func TestRemoteVersionCollector(t *testing.T) {
	tests := []struct {
		name    string
		client  fakeRemoteVersionClient
		want    []string
		wantErr bool
	}{
		{
			name: "update available",
			client: fakeRemoteVersionClient{
				result: client.RemoteVersion{
					UpdateAvailable: true,
					Version:         "1.2.3",
				},
			},
			want: []string{
				`paperless_remote_version_update_available{version="1.2.3"} 1`,
				`paperless_warnings_total{category="get_remote_version"} 0`,
			},
		},
		{
			name: "no update",
			client: fakeRemoteVersionClient{
				result: client.RemoteVersion{
					UpdateAvailable: false,
					Version:         "1.2.3",
				},
			},
			want: []string{
				`paperless_remote_version_update_available{version="1.2.3"} 0`,
				`paperless_warnings_total{category="get_remote_version"} 0`,
			},
		},
		{
			name: "error case",
			client: fakeRemoteVersionClient{
				err: errors.New("test error"),
			},
			want: []string{
				`paperless_remote_version_update_available{version=""} 0`,
				`paperless_warnings_total{category="get_remote_version"} 1`,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			c := newRemoteVersionCollector(&tt.client)
			
			// Collect metrics
			registry := prometheus.NewRegistry()
			registry.MustRegister(c)

			// Gather metrics
			mfs, err := registry.Gather()
			if err != nil {
				t.Fatalf("Gather failed: %v", err)
			}

			// Convert to text format
			var buf bytes.Buffer
			for _, mf := range mfs {
				if _, err := expfmt.MetricFamilyToText(&buf, mf); err != nil {
					t.Fatalf("Metric family to text failed: %v", err)
				}
			}

			// Process output
			got := strings.TrimSpace(buf.String())
			gotLines := strings.Split(got, "\n")
			
			// Filter out comment lines and empty lines
			var filteredLines []string
			for _, line := range gotLines {
				line = strings.TrimSpace(line)
				if line != "" && !strings.HasPrefix(line, "#") {
					filteredLines = append(filteredLines, line)
				}
			}

			// Sort both slices for consistent comparison
			sort.Strings(filteredLines)
			sort.Strings(tt.want)

			// Compare
			if diff := cmp.Diff(tt.want, filteredLines); diff != "" {
				t.Errorf("Metrics diff (-want +got):\n%s", diff)
			}

			// Verify error condition
			if tt.wantErr && !strings.Contains(got, "get_remote_version\"} 1") {
				t.Error("Expected error metric not found")
			}
		})
	}
}
