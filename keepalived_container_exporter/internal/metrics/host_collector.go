package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"keepalived_container_exporter/pkg/utils"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// KeepalivedHostCollectorHost implements Collector for when Keepalived and Keepalived Exporter are both on a same host.
type KeepalivedHostCollectorHost struct {
	pidPath string
	version *version.Version
	useJSON bool

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedHostCollectorHost is creating new instance of KeepalivedHostCollectorHost.
func NewKeepalivedHostCollectorHost(useJSON bool, pidPath string) *KeepalivedHostCollectorHost {
	k := &KeepalivedHostCollectorHost{
		useJSON: useJSON,
		pidPath: pidPath,
	}

	var err error
	if k.version, err = k.getKeepalivedVersion(); err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}

	k.initSignals()

	return k
}

func (k *KeepalivedHostCollectorHost) Refresh() error {
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

func (k *KeepalivedHostCollectorHost) initSignals() {
	if k.useJSON {
		k.SIGJSON = k.sigNum("JSON")
	}

	k.SIGDATA = k.sigNum("DATA")
	k.SIGSTATS = k.sigNum("STATS")
}

// GetKeepalivedVersion returns Keepalived version.
func (k *KeepalivedHostCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("bash", "-c", "keepalived -v")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).
			WithError(err).
			Error("Error getting keepalived version")

		return nil, errors.New("error getting keepalived version")
	}

	return ParseVersion(stderr.String())
}

// Signal sends signal to Keepalived process.
func (k *KeepalivedHostCollectorHost) signal(signal os.Signal) error {
	pid, err := k.getKeepalivedPID()
	if err != nil {
		logrus.WithField("path", k.pidPath).WithError(err).Error("Unknown pid found for keepalived")

		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to find process")

		return err
	}

	err = proc.Signal(signal)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to send signal")

		return err
	}

	return nil
}

// getKeepalivedPID 获取Keepalived进程PID
func (k *KeepalivedHostCollectorHost) getKeepalivedPID() (int, error) {
	data, err := os.ReadFile(k.pidPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// SigNum returns signal number for given signal name.
func (k *KeepalivedHostCollectorHost) sigNum(sigString string) syscall.Signal {
	allow_sig_nums := []string{"JSON", "DATA", "STATS"}
	if !contains(allow_sig_nums, sigString) {
		logrus.WithField("signal", sigString).Fatal("Unsupported signal")
	}
	if !HasSigNumSupport(k.version) {
		return GetDefaultSignal(sigString)
	}

	var stdout, stderr bytes.Buffer

	sigNumCommand := "keepalived --signum=" + sigString
	cmd := utils.GetCommand("bash", "-c", sigNumCommand)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "stderr": stderr.String()}).
			WithError(err).
			Fatal("Error getting signum")
	}

	return syscall.Signal(parseSigNum(stdout, sigString))
}

func (k *KeepalivedHostCollectorHost) JSONVrrps() ([]VRRP, error) {
	const fileName = "/tmp/keepalived.json"
	tp := JSONParser{}

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open JSON VRRP file")

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

func (k *KeepalivedHostCollectorHost) StatsVrrps() (map[string]*VRRPStats, error) {
	const fileName = "/tmp/keepalived.stats"

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Stats VRRP file")

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

func (k *KeepalivedHostCollectorHost) DataVrrps() (map[string]*VRRPData, error) {
	const fileName = "/tmp/keepalived.data"
	// tp := TextParser{}

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Data VRRP file")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return ParseVRRPData(f)
}

func (k *KeepalivedHostCollectorHost) ScriptVrrps() ([]VRRPScript, error) {
	const fileName = "/tmp/keepalived.data"
	tp := TextParser{}

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Script VRRP file")

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
func (k *KeepalivedHostCollectorHost) HasVRRPScriptStateSupport() bool {
	return HasVRRPScriptStateSupport(k.version)
}

func parseSigNum(sigNum bytes.Buffer, sigString string) int64 {
	var signum int64
	if err := json.Unmarshal(sigNum.Bytes(), &signum); err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": sigNum.String()}).
			WithError(err).
			Fatal("Error parsing signum result")
	}

	return signum
}
