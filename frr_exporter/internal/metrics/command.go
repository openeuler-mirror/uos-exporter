package metrics

import (
	"bytes"
	"context"
	"fmt"
	"frr_exporter/pkg/utils"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kingpin/v2"
)

var (
	vtyshEnable     = kingpin.Flag("frr.vtysh", "Use vtysh to query FRR instead of each daemon's Unix socket (default: disabled, recommended: disabled).").Default("false").Bool()
	vtyshPath       = kingpin.Flag("frr.vtysh.path", "Path of vtysh.").Default("/usr/bin/vtysh").String()
	vtyshTimeout    = kingpin.Flag("frr.vtysh.timeout", "The timeout when running vtysh commands (default: 20s).").Default("20s").Duration()
	vtyshSudo       = kingpin.Flag("frr.vtysh.sudo", "Enable sudo when executing vtysh commands.").Bool()
	frrVTYSHOptions = kingpin.Flag("frr.vtysh.options", "Additional options passed to vtysh.").Default("").String()
)

func executeBFDCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecBFDCmd(cmd)
}

func executeBGPCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecBGPCmd(cmd)
}

func executeOSPFMultiInstanceCommand(cmd string, instanceID int) ([]byte, error) {
	return socketConn.ExecOSPFMultiInstanceCmd(cmd, instanceID)
}

func executeOSPFCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecOSPFCmd(cmd)
}

func executePIMCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecPIMCmd(cmd)
}

func executeZebraCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecZebraCmd(cmd)
}

func executeVRRPCommand(cmd string) ([]byte, error) {
	if *vtyshEnable {
		return execVtyshCommand(cmd)
	}
	return socketConn.ExecVRRPCmd(cmd)
}

func execVtyshCommand(vtyshCmd string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), *vtyshTimeout)
	defer cancel()

	var a []string
	var executable string

	if *vtyshSudo {
		a = []string{*vtyshPath}
		executable = "/usr/bin/sudo"
	} else {
		a = []string{}
		executable = *vtyshPath
	}

	if *frrVTYSHOptions != "" {
		frrOptions := strings.Split(*frrVTYSHOptions, " ")
		a = append(a, frrOptions...)
	}

	a = append(a, "-c", vtyshCmd)
	cmd := utils.GetCommandWithContext(ctx, executable, a...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return stdout.Bytes(), fmt.Errorf("command %s failed: %w: stderr: %s: stdout: %s", cmd, err, strings.ReplaceAll(stderr.String(), "\n", " "), strings.ReplaceAll(stdout.String(), "\n", " "))
	}

	return stdout.Bytes(), nil
}

// 检查路径是否包含危险字符或模式
func isPathSafe(path string) bool {
	// 检查路径遍历攻击
	if strings.Contains(path, "..") || strings.Contains(path, "./") {
		return false
	}

	// 检查空字节注入
	if strings.Contains(path, "\x00") {
		return false
	}

	// 检查命令注入字符
	dangerousChars := []string{"|", "&", ";", "`", "$", "(", ")", "<", ">", "\\n"}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	return true
}

// 规范化并验证路径
func validatePath(baseDir, userPath string) (string, error) {
	// 清理路径
	cleanPath := filepath.Clean(userPath)

	// 检查路径安全性
	if !isPathSafe(cleanPath) {
		return "", fmt.Errorf("路径包含危险字符")
	}

	// 获取绝对路径
	absPath, err := filepath.Abs(filepath.Join(baseDir, cleanPath))
	if err != nil {
		return "", err
	}

	// 检查是否在允许的目录内
	if !strings.HasPrefix(absPath, baseDir) {
		return "", fmt.Errorf("路径超出允许范围")
	}

	return absPath, nil
}
