package cmdline

import (
	"github.com/alecthomas/kingpin/v2"
)

// 全局 kingpin 应用实例
var (
	App = kingpin.New("newrelic_exporter", "NewRelic metrics exporter for Prometheus")
)

// 初始化函数
func init() {
	App.HelpFlag.Short('h')
} 