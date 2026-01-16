package metrics

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"samba_exporter/internal/exporter"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func init() {
	exporter.Register(
		InitSambaExporter())
}

// The version of this program, will be set at compile time by the ./build.sh build script
var version = "1.0.0"

type Parmeters struct {
	Test                        bool
	Test_pipe_mode              bool
	Request_timeout             int
	Do_not_expose_encryption    bool
	Do_not_expose_client        bool
	Do_not_expose_user          bool
	Do_not_expose_pid           bool
	Do_not_expose_share_details bool
}

var parms = Parmeters{
	Test:                        false,
	Test_pipe_mode:              false,
	Request_timeout:             20,
	Do_not_expose_encryption:    false,
	Do_not_expose_client:        false,
	Do_not_expose_user:          false,
	Do_not_expose_pid:           false,
	Do_not_expose_share_details: false,
}

// var (
// 	RequestHandler  *PipeHandler
// 	ResponseHandler *PipeHandler
// )

func InitSambaExporter() *SambaExporter {
	config, err := LoadConfig("/etc/uos-exporter/samba-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error parse samba config %v\n", err)
	} else {
		parms.Test = config.Test
		parms.Test_pipe_mode = config.TestPipeMode
		parms.Request_timeout = config.RequestTimeOut
		parms.Do_not_expose_encryption = config.DoNotExportEncryption
		parms.Do_not_expose_client = config.DoNotExportClient
		parms.Do_not_expose_user = config.DoNotExportUser
		parms.Do_not_expose_pid = config.DoNotExportPid
		parms.Do_not_expose_share_details = config.DoNotExportShareDetails
	}

	RequestHandler := NewPipeHandler(parms.Test, RequestPipe)
	ResponseHandler := NewPipeHandler(parms.Test, ResposePipe)

	logrus.Info(fmt.Sprintf("Named pipe for requests: %s", RequestHandler.GetPipeFilePath()))
	logrus.Info(fmt.Sprintf("Named pipe for response: %s", ResponseHandler.GetPipeFilePath()))

	if parms.Do_not_expose_user {
		logrus.Info("-not-expose-user-data set, will not export user data")
	}

	if parms.Do_not_expose_client {
		logrus.Info("-not-expose-client-data set, will not export client data")
	}

	if parms.Do_not_expose_encryption {
		logrus.Info("-not-expose-encryption-data set, will not export encryption data")
	}

	if parms.Do_not_expose_share_details {
		logrus.Info("-not-expose-share-details set, will not export share details")
	}

	if parms.Test_pipe_mode {
		errTest := testPipeMode(RequestHandler, ResponseHandler)
		if errTest != nil {
			logrus.Error(errTest)
			return nil
		}
		return nil
	}

	// Ensure we exit clean on term and kill signals
	go waitforKillSignalAndExit()
	go waitforTermSignalAndExit()

	logrus.Info("Setup prometheus exporter")

	exporter := NewSambaExporter(RequestHandler, ResponseHandler, version, parms.Request_timeout, parms)

	return exporter
}

func testPipeMode(requestHandler *PipeHandler, responseHandler *PipeHandler) error {
	var processes []ProcessData
	var shares []ShareData
	var locks []LockData
	var psData []PsUtilPidData
	var errGet error

	logrus.Info("Request samba_statusd to get metrics for test-pipe mode")
	locks, processes, shares, psData, errGet = GetSambaStatus(requestHandler, responseHandler, parms.Request_timeout)
	if errGet != nil {
		return errGet
	}

	handleTestResponse(processes, shares, locks, psData)

	return nil
}

func handleTestResponse(processes []ProcessData, shares []ShareData, locks []LockData, psData []PsUtilPidData) {
	logrus.Info("Handle samba_statusd  response in test-pipe mode")

	for _, share := range shares {
		fmt.Fprintln(os.Stdout, share.String())
	}
	for _, process := range processes {
		fmt.Fprintln(os.Stdout, process.String())
	}
	for _, lock := range locks {
		fmt.Fprintln(os.Stdout, lock.String())
	}

	for _, ps := range psData {
		fmt.Fprintln(os.Stdout, ps.String())
	}

	stats := GetSmbStatistics(locks, processes, shares, parms)
	stats = append(stats, GetSmbdMetrics(psData, parms.Do_not_expose_pid)...)
	for _, stat := range stats {
		fmt.Fprintf(os.Stdout, "%s_%s: %f", EXPORTER_LABEL_PREFIX, stat.Name, stat.Value)
	}
}

func waitforKillSignalAndExit() {
	killSignal := make(chan os.Signal, syscall.SIGKILL)
	signal.Notify(killSignal, os.Interrupt)
	<-killSignal

	logrus.Info(fmt.Sprintf("End %s due to kill signal", os.Args[0]))

	os.Exit(0)
}

func waitforTermSignalAndExit() {
	termSignal := make(chan os.Signal, syscall.SIGTERM)
	signal.Notify(termSignal, os.Interrupt)
	<-termSignal

	logrus.Info(fmt.Sprintf("End %s due to terminate signal", os.Args[0]))

	os.Exit(0)
}

type SambaExporterConfig struct {
	Test                    bool `yaml:"Test"`
	TestPipeMode            bool `yaml:"Test_pipe_mode"`
	RequestTimeOut          int  `yaml:"Request_timeout"`
	DoNotExportEncryption   bool `yaml:"Do_not_expose_encryption"`
	DoNotExportClient       bool `yaml:"Do_not_expose_client"`
	DoNotExportUser         bool `yaml:"Do_not_expose_user"`
	DoNotExportPid          bool `yaml:"Do_not_expose_pid"`
	DoNotExportShareDetails bool `yaml:"Do_not_expose_share_details"`
}

func LoadConfig(path string) (*SambaExporterConfig, error) {
	// 清理路径并验证
	cleanPath := filepath.Clean(path)
	// 防止路径遍历攻击
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path: path traversal not allowed")
	}
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return nil, fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}

	// 可选：限制配置文件必须在特定目录
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		cleanPath = filepath.Join(configDir, cleanPath)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var config SambaExporterConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
