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


// TODO: implement functions
