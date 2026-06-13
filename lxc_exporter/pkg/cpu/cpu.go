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
func GetProcStat() (ProcStat, error) {
	content, err := fetchProcStat()
	if err != nil {
		return ProcStat{}, err
	}

	return parseProcStat(content)
}

// Refresh 更新 CPU 统计信息并计算增量
func (p *ProcStat) Refresh() error {
	content, err := fetchProcStat()
	if err != nil {
		return err
	}

	newProc, err := parseProcStat(content)
	if err != nil {
		return err
	}

	p.User = newProc.User - p.prevUser
	p.System = newProc.System - p.prevSystem
	p.Nice = newProc.Nice - p.prevNice
	p.Idle = newProc.Idle - p.prevIdle
	p.Wait = newProc.Wait - p.prevWait
	p.Irq = newProc.Irq - p.prevIrq
	p.Srq = newProc.Srq - p.prevSrq
	p.Zero = newProc.Zero - p.prevZero

	p.prevUser = newProc.User
	p.prevSystem = newProc.System
	p.prevNice = newProc.Nice
	p.prevIdle = newProc.Idle
	p.prevWait = newProc.Wait
	p.prevIrq = newProc.Irq
	p.prevSrq = newProc.Srq
	p.prevZero = newProc.Zero

	return nil
}

// 读取 `/proc/stat` 第一行内容
func fetchProcStat() (string, error) {
	file, err := os.Open(procStatPath)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", procStatPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read %s: %w", procStatPath, err)
	}
	return "", fmt.Errorf("empty %s file", procStatPath)
}

// 解析 CPU 统计信息
func parseProcStat(line string) (ProcStat, error) {
	fields := strings.Fields(line)
	if len(fields) < 9 || fields[0] != "cpu" {
		return ProcStat{}, fmt.Errorf("invalid /proc/stat format")
	}

	// 转换 CPU 统计数据
	values := make([]float64, 8)
	for i := 0; i < 8; i++ {
		v, err := strconv.ParseFloat(fields[i+1], 64)
		if err != nil {
			return ProcStat{}, fmt.Errorf("failed to parse cpu value: %w", err)
		}
		values[i] = v
	}

	return ProcStat{
		User:       values[0],
		System:     values[1],
		Nice:       values[2],
		Idle:       values[3],
		Wait:       values[4],
		Irq:        values[5],
		Srq:        values[6],
		Zero:       values[7],
		prevUser:   values[0],
		prevSystem: values[1],
		prevNice:   values[2],
		prevIdle:   values[3],
		prevWait:   values[4],
		prevIrq:    values[5],
		prevSrq:    values[6],
		prevZero:   values[7],
	}, nil
}
