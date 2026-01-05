package core

import (
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"hacluster_exporter/internal/clock"
	mock_core "hacluster_exporter/test/mock_collector"
)


// TODO: implement functions
