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


// TODO: implement functions
