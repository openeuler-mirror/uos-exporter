package cpu

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	procStatPath = "/proc/stat"
)

// ProcStat 结构体表示 CPU 使用情况
type ProcStat struct {
	User       float64
	System     float64
	Nice       float64
	Idle       float64
	Wait       float64
	Irq        float64
	Srq        float64
	Zero       float64
	prevUser   float64
	prevSystem float64
	prevNice   float64
	prevIdle   float64
	prevWait   float64
	prevIrq    float64
	prevSrq    float64
	prevZero   float64
}

// GetProcStat 读取 `/proc/stat` 并返回解析后的 CPU 统计数据

// TODO: implement functions
