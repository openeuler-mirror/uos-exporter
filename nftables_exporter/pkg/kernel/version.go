package kernel

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	kernelVersionFile = "/proc/version"
	versionPattern    = regexp.MustCompile("Linux version ([0-9.]+)") // 缓存正则表达式
	cachedVersion     string
)

func GetVersion() (string, error) {
	if cachedVersion != "" {
		return cachedVersion, nil
	}
	content, err := getVersionFile()
	if err != nil {
		return "", fmt.Errorf("failed to read version file: %w", err)
	}

	version := extractKernelVersion(content)
	if version == "" {
		return "", fmt.Errorf("failed to parse kernel version from content")
	}
	cachedVersion = version
	return version, nil
}

func GetMajorVersion() (int, error) {
	version, err := GetVersion()
	if err != nil {
		return 0, err
	}

	majorVersion, err := parseMajorVersion(version)
	if err != nil {
		return 0, fmt.Errorf("failed to parse major version from string: %w", err)
	}
	return majorVersion, nil
}

func getVersionFile() (string, error) {
	content, err := os.ReadFile(kernelVersionFile)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", kernelVersionFile, err)
	}
	return string(content), nil
}

func extractKernelVersion(content string) string {
	found := versionPattern.FindStringSubmatch(content)
	if len(found) > 1 {
		return found[1]
	}
	return ""
}

func parseMajorVersion(version string) (int, error) {
	if version == "" {
		return 0, fmt.Errorf("version string is empty")
	}

	parts := strings.Split(version, ".")
	if len(parts) == 0 || parts[0] == "" {
		return 0, fmt.Errorf("invalid version string format")
	}

	majorVersion, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to convert major version to int: %w", err)
	}
	return majorVersion, nil
}
