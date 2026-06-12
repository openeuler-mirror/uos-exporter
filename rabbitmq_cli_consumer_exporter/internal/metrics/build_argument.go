package metrics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"rabbitmq_cli_consumer_exporter/pkg/utils"
	"strings"

	"github.com/bketelsen/logr"
)

type ArgumentBuilder struct {
	Builder
	Compressed   bool
	WithMetadata bool
	log          logr.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
}

// SetLogger is part of Builder.

// TODO: implement
