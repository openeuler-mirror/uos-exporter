package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/hansmi/prometheus-paperless-exporter/internal/testutil"
)

type fakeStatusClient struct {
	err error
}

func (c *fakeStatusClient) GetStatus(ctx context.Context) (*client.SystemStatus, *client.Response, error) {
	return &client.SystemStatus{
		PNGXVersion: "2.14.7",
		ServerOS:    "Linux-6.8.12-8-pve-x86_64-with-glibc2.36",
		InstallType: "bare-metal",
		Storage: client.SystemStatusStorage{
			Total:     21474836480,
			Available: 13406437376,
		},
		Database: client.SystemStatusDatabase{
			Type:   "postgresql",
			URL:    "paperlessdb",
			Status: "OK",
			Error:  "",
			MigrationStatus: client.SystemStatusDatabaseMigration{
				LatestMigration:     "mfa.0003_authenticator_type_uniq",
				UnappliedMigrations: []string{},
			},
		},
		Tasks: client.SystemStatusTasks{
			RedisURL:              "redis://localhost:6379",
			RedisStatus:           "OK",
			RedisError:            "",
			CeleryStatus:          "OK",
			IndexStatus:           "OK",
			IndexLastModified:     time.Date(2025, time.February, 21, 0, 1, 54, 773392000, time.UTC),
			IndexError:            "",
			ClassifierStatus:      "OK",
			ClassifierLastTrained: time.Date(2025, time.February, 21, 20, 5, 1, 589548000, time.UTC),
			ClassifierError:       "",
		},
	}, &client.Response{}, c.err
}


// TODO: implement functions
