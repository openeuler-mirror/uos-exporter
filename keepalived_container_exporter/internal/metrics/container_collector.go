package metrics

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// KeepalivedContainerCollectorHost implements Collector for when Keepalived is on container and Keepalived Exporter is on a host.
type KeepalivedContainerCollectorHost struct {
	version       *version.Version
	useJSON       bool
	containerName string
	dataPath      string
	jsonPath      string
	statsPath     string
	dockerCli     *client.Client
	pidPath       string

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedContainerCollectorHost is creating new instance of KeepalivedContainerCollectorHost.
func NewKeepalivedContainerCollectorHost(
	useJSON bool,
	containerName, containerTmpDir, pidPath string,
) *KeepalivedContainerCollectorHost {
	k := &KeepalivedContainerCollectorHost{
		useJSON:       useJSON,
		containerName: containerName,
		pidPath:       pidPath,
	}

	var err error

	k.dockerCli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating docker env client")
	}

	k.initVersion()

	k.initSignals()

	k.initPaths(containerTmpDir)

	return k
}

func (c *KeepalivedContainerCollectorHost) initVersion() {
	var err error
	c.version, err = c.getKeepalivedVersion()
	if err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}
}

func (k *KeepalivedContainerCollectorHost) Refresh() error {
	if k.useJSON {
		if err := k.signal(k.SIGJSON); err != nil {
			logrus.WithError(err).Error("Failed to send JSON signal to keepalived")

			return err
		}

		return nil
	}

	if err := k.signal(k.SIGSTATS); err != nil {
		logrus.WithError(err).Error("Failed to send STATS signal to keepalived")

		return err
	}

	if err := k.signal(k.SIGDATA); err != nil {
		logrus.WithError(err).Error("Failed to send DATA signal to keepalived")

		return err
	}

	return nil
}

func (k *KeepalivedContainerCollectorHost) initPaths(containerTmpDir string) {
	k.jsonPath = filepath.Join(containerTmpDir, "keepalived.json")
	k.statsPath = filepath.Join(containerTmpDir, "keepalived.stats")
	k.dataPath = filepath.Join(containerTmpDir, "keepalived.data")
}

// GetKeepalivedVersion returns Keepalived version.
func (k *KeepalivedContainerCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	getVersionCmd := []string{"keepalived", "-v"}

	stdout, err := k.dockerExecCmd(getVersionCmd)
	if err != nil {
		return nil, err
	}

	return ParseVersion(stdout.String())
}

func (k *KeepalivedContainerCollectorHost) initSignals() {
	if k.useJSON {
		k.SIGJSON = k.sigNum("JSON")
	}

	k.SIGDATA = k.sigNum("DATA")
	k.SIGSTATS = k.sigNum("STATS")
}

// SigNum returns signal number for given signal name.
func (k *KeepalivedContainerCollectorHost) sigNum(sigString string) syscall.Signal {
	if !HasSigNumSupport(k.version) {
		return GetDefaultSignal(sigString)
	}

	sigNumCommand := []string{"keepalived", "--signum", sigString}

	stdout, err := k.dockerExecCmd(sigNumCommand)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "container": k.containerName}).
			WithError(err).
			Fatal("Error getting signum")
	}

	reg := regexp.MustCompile("[^0-9]+")
	strSigNum := reg.ReplaceAllString(stdout.String(), "")

	signum, err := strconv.ParseInt(strSigNum, 10, 32)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": stdout.String()}).
			WithError(err).
			Fatal("Error parsing signum result")
	}

	return syscall.Signal(signum)
}

// getPID 获取进程ID
func (k *KeepalivedContainerCollectorHost) getPID() (string, error) {
	data, err := os.ReadFile(k.pidPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Signal sends signal to Keepalived process.
func (k *KeepalivedContainerCollectorHost) signal(signal syscall.Signal) error {
	// data, err := os.ReadFile(k.pidPath)
	pid, err := k.getPID()
	if err != nil {
		logrus.WithField("path", k.pidPath).WithError(err).Info("Can't find keepalived pid. Falling back to the default process.")

		err := k.dockerCli.ContainerKill(context.Background(), k.containerName, strconv.Itoa(int(signal)))
		if err != nil {
			logrus.WithError(err).WithField("signal", int(signal)).Error("Failed to send signal")

			return err
		}

		return nil
	}

	// pid := strings.TrimSuffix(string(data), "\n")
	logrus.WithField("pid", pid).Info("Pid found")

	cmd := strslice.StrSlice{"kill", "-" + strconv.Itoa(int(signal)), pid}
	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create the execution instance
	execIDResp, err := k.dockerCli.ContainerExecCreate(context.Background(), k.containerName, execConfig)
	if err != nil {
		logrus.WithError(err).Error("Error creating exec instance")

		return err
	}

	// Start the execution of the created command
	err = k.dockerCli.ContainerExecStart(context.Background(), execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		logrus.WithError(err).Error("Error starting exec command")

		return err
	}

	return nil
}

// JSONVrrps send SIGJSON and parse the data to the list of collector.VRRP struct.
func (k *KeepalivedContainerCollectorHost) JSONVrrps() ([]VRRP, error) {
	tp := JSONParser{}
	f, err := os.Open(k.jsonPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.jsonPath).Error("Failed to open keepalived.json")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return tp.Parse(f)
}

// StatsVrrps send SIGSTATS and parse the stats.
func (k *KeepalivedContainerCollectorHost) StatsVrrps() (map[string]*VRRPStats, error) {
	f, err := os.Open(k.statsPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.statsPath).Error("Failed to open keepalived.stats")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return ParseStats(f)
}

// DataVrrps send SIGDATA ans parse the data.
func (k *KeepalivedContainerCollectorHost) DataVrrps() (map[string]*VRRPData, error) {
	// tp := TextParser{}
	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()
	// log.Printf("Failed to get keepalived data: %v\n", err)

	return ParseVRRPData(f)
}

// ScriptVrrps parse the script data from keepalived.data.
func (k *KeepalivedContainerCollectorHost) ScriptVrrps() ([]VRRPScript, error) {
	tp := TextParser{}
	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return tp.ParseVRRPScript(f), nil
}

// HasVRRPScriptStateSupport check if Keepalived version supports VRRP Script State in output.
func (k *KeepalivedContainerCollectorHost) HasVRRPScriptStateSupport() bool {
	return HasVRRPScriptStateSupport(k.version)
}

func (k *KeepalivedContainerCollectorHost) dockerExecCmd(cmd []string) (*bytes.Buffer, error) {
	rst, err := k.dockerCli.ContainerExecCreate(context.Background(), k.containerName, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error creating exec container")

		return nil, err
	}

	response, err := k.dockerCli.ContainerExecAttach(context.Background(), rst.ID, types.ExecStartCheck{})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error attaching a connection to an exec process")

		return nil, err
	}
	defer response.Close()

	data, err := io.ReadAll(response.Reader)
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error reading response from docker command")

		return nil, err
	}

	return bytes.NewBuffer(data), nil
}
