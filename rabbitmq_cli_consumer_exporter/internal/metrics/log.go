package metrics

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"

	"github.com/bketelsen/logr"
	log "github.com/corvus-ch/logr/std"
)

// NewFromConfig crates a logger according to the given config.
func NewLogFromConfig(cfg *Config) (logr.Logger, io.Writer, io.Writer, error) {
	createLogger := func(logType string, writerFunc func() (io.Writer, error), defaultWriter io.Writer) (*stdlog.Logger, io.Writer, error) {
		w, err := writerFunc()
		if err != nil {
			return nil, nil, fmt.Errorf("failed creating %s log: %s", logType, err)
		}
		return stdlog.New(w, "", flag(cfg.WithDateTime())), w, nil
	}

	errL, errW, err := createLogger("error",
		func() (io.Writer, error) { return newWriter(cfg.Logs.Error, cfg.IsVerbose(), os.Stderr) },
		os.Stderr)
	if err != nil {
		return nil, nil, nil, err
	}

	infL, outW, err := createLogger("info",
		func() (io.Writer, error) { return newWriter(cfg.Logs.Info, cfg.IsVerbose(), os.Stdout) },
		os.Stdout)
	if err != nil {
		return nil, nil, nil, err
	}

	return log.New(0, errL, infL), outW, errW, nil
}

// newWriter creates a new writer for the given file.
// If verbose is set to true, in addition to the file, the logger will also write to writer passed as the out argument.
func newWriter(filename string, verbose bool, out io.Writer) (io.Writer, error) {
	writers := make([]io.Writer, 0)
	cleanFileName := filepath.Clean(filename)
	if len(cleanFileName) > 0 || !verbose {
		file, err := os.OpenFile(cleanFileName, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)

		if err != nil {
			return nil, err
		}

		writers = append(writers, file)
	}

	if verbose {
		writers = append(writers, out)
	}

	return io.MultiWriter(writers...), nil
}

func flag(dateTime bool) int {
	if dateTime {
		return stdlog.LstdFlags
	}

	return 0
}
