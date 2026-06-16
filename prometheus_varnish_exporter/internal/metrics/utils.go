package metrics

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	logger *log.Logger = log.New(os.Stdout, "", log.LstdFlags)
)

type LoggingConfig struct {
	RawOutputEnabled bool
}

var LogConfig LoggingConfig = LoggingConfig{RawOutputEnabled: false}

func logRaw(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func logTitle(format string, args ...interface{}) {
	logInfo(format, args...)

	formattedTitle := fmt.Sprintf(format, args...)
	titleUnderline := strings.Repeat("-", len(formattedTitle))
	if len(titleUnderline) > 0 {
		logInfo(titleUnderline)
	}
}

func logInfo(format string, args ...interface{}) {
	if LogConfig.RawOutputEnabled {
		logRaw(format, args...)
	} else {
		logger.Printf(format, args...)
	}
}

func logWarn(format string, args ...interface{}) {
	warningMessage := "[WARN] " + format
	if LogConfig.RawOutputEnabled {
		logRaw(warningMessage, args...)
	} else {
		logger.Printf(warningMessage, args...)
	}
}

func logError(format string, args ...interface{}) {
	errorMessage := "[ERROR] " + format
	if LogConfig.RawOutputEnabled {
		logRaw(errorMessage, args...)
	} else {
		logger.Printf(errorMessage, args...)
	}
}

func logFatal(format string, args ...interface{}) {
	fatalMessage := "[FATAL] " + format
	if LogConfig.RawOutputEnabled {
		logRaw(fatalMessage, args...)
	} else {
		logger.Printf(fatalMessage, args...)
	}
	os.Exit(1)
}

func logFatalError(err error) {
	if err != nil {
		logFatal(err.Error())
	}
}

type CaseSensitivity int

const (
	CaseSensitive   CaseSensitivity = 0
	CaseInsensitive CaseSensitivity = 1
)

func startsWith(str, prefix string, cs CaseSensitivity) bool {
	n := len(prefix)
	if len(str) < n {
		return false
	}

	// 快速路径：直接比较
	if cs == CaseSensitive {
		return str[:n] == prefix
	}

	// 优化点：按 8 字节块比较
	for i := 0; i < n; i += 8 {
		end := i + 8
		if end > n {
			end = n
		}

		// 批量处理字节块
		if !compareChunk(str[i:end], prefix[i:end], cs) {
			return false
		}
	}
	return true
}

func compareChunk(a, b string, cs CaseSensitivity) bool {
	if cs == CaseSensitive {
		return a == b
	}

	// 向量化处理 (实际实现可能需要汇编)
	for i := 0; i < len(a); i++ {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

func toLower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func startsWithAny(str string, prefixes []string, cs CaseSensitivity) bool {
	for _, prefix := range prefixes {
		if startsWith(str, prefix, cs) {
			return true
		}
	}
	return false
}

func endsWith(str, postfix string, cs CaseSensitivity) bool {
	switch cs {
	case CaseSensitive:
		return strings.HasSuffix(str, postfix)
	case CaseInsensitive:
		return strings.HasSuffix(strings.ToLower(str), strings.ToLower(postfix))
	default:
		return false
	}
}

func endsWithAny(str string, postfixes []string, cs CaseSensitivity) bool {
	for _, postfix := range postfixes {
		if endsWith(str, postfix, cs) {
			return true
		}
	}
	return false
}

// Returns if file/dir in path exists.
func fileExists(path string) bool {
	if len(path) == 0 {
		return false
	}
	fileInfo, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return fileInfo != nil
}

// data
func stringProperty(data map[string]interface{}, key string) (string, error) {
	value, exists := data[key]
	if !exists || value == nil {
		return "", nil
	}

	strValue, isString := value.(string)
	if !isString {
		return "", fmt.Errorf("%s is not a string", key)
	}

	return strValue, nil
}

// Additional helper functions
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}

func isFile(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fileInfo.IsDir()
}

// func readFileContent(path string) (string, error) {
// 	if !fileExists(path) {
// 		return "", fmt.Errorf("file does not exist: %s", path)
// 	}

// 	content, err := os.ReadFile(path)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read file: %s", err)
// 	}

// 	return string(content), nil
// }

// func writeFileContent(path string, content string) error {
// 	err := os.WriteFile(path, []byte(content), 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to write file: %s", err)
// 	}
// 	return nil
// }

// func appendToFile(path string, content string) error {
// 	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
// 	if err != nil {
// 		return fmt.Errorf("failed to open file for appending: %s", err)
// 	}
// 	defer file.Close()

// 	_, err = file.WriteString(content)
// 	if err != nil {
// 		return fmt.Errorf("failed to append to file: %s", err)
// 	}

// 	return nil
// }

func deleteFile(path string) error {
	if !fileExists(path) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	err := os.Remove(path)
	if err != nil {
		return fmt.Errorf("failed to delete file: %s", err)
	}

	return nil
}

// func createDirectory(path string) error {
// 	if fileExists(path) {
// 		return fmt.Errorf("directory already exists: %s", path)
// 	}

// 	err := os.MkdirAll(path, 0755)
// 	if err != nil {
// 		return fmt.Errorf("failed to create directory: %s", err)
// 	}

// 	return nil
// }

func deleteDirectory(path string) error {
	if !fileExists(path) {
		return fmt.Errorf("directory does not exist: %s", path)
	}

	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("failed to delete directory: %s", err)
	}

	return nil
}
