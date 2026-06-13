package mtr

import (
	"context"
	"fmt"
	"math"
	"network_exporter/pkg/common"
	"network_exporter/pkg/icmp"
	"sync"
	"sync/atomic"
	"time"
)

// ConcurrentMTROptions 并发MTR的配置选项
type ConcurrentMTROptions struct {
	MaxWorkers     int           // 最大工作器数量
	BatchSize      int           // 批处理大小
	EarlyStop      bool          // 是否启用提前停止
	ProgressReport bool          // 是否启用进度报告
	Timeout        time.Duration // 总体超时时间
}

// DefaultConcurrentMTROptions 返回默认的并发MTR配置
func DefaultConcurrentMTROptions() *ConcurrentMTROptions {
	return &ConcurrentMTROptions{
		MaxWorkers:     10,
		BatchSize:      5,
		EarlyStop:      true,
		ProgressReport: false,
		Timeout:        30 * time.Second,
	}
}

// RunMTRConcurrent 执行高性能并发MTR路由跟踪
func RunMTRConcurrent(destAddr, srcAddr string, timeout time.Duration, maxHops, count int, options *ConcurrentMTROptions) *MtrResult {
	if options == nil {
		options = DefaultConcurrentMTROptions()
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	result := &MtrResult{
		DestAddr:      destAddr,
		Hops:          []common.IcmpHop{},
		HopSummaryMap: make(map[string]*common.IcmpSummary),
	}

	// 使用ICMP ID
	icmpID := &common.IcmpID{}
	pid := int(icmpID.Get())

	// 线程安全的结果存储
	mtrReturns := make([]*safeReturn, maxHops+1)
	for i := 1; i < maxHops+1; i++ {
		mtrReturns[i] = &safeReturn{
			mtrReturn: &MtrReturn{
				ttl:       i,
				host:      "unknown",
				succSum:   0,
				success:   false,
				lastTime:  time.Duration(0),
				sumTime:   time.Duration(0),
				bestTime:  time.Duration(0),
				worstTime: time.Duration(0),
				avgTime:   time.Duration(0),
				allTime:   make([]time.Duration, 0, count),
			},
		}
	}

	// 任务生成器
	taskChan := make(chan task, options.BatchSize*options.MaxWorkers)
	go func() {
		defer close(taskChan)
		for snt := 0; snt < count; snt++ {
			for ttl := 1; ttl < maxHops; ttl++ {
				select {
				case taskChan <- task{ttl: ttl, snt: snt}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// 工作器池
	var wg sync.WaitGroup
	var seqCounter int64
	reachedDestination := int32(0)

	// 调整工作器数量
	workerCount := options.MaxWorkers
	if workerCount > maxHops {
		workerCount = maxHops
	}

	// 启动工作器
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case task, ok := <-taskChan:
					if !ok {
						return
					}

					// 如果启用提前停止且已到达目标，跳过高TTL的测试
					if options.EarlyStop && atomic.LoadInt32(&reachedDestination) == 1 && task.ttl > int(atomic.LoadInt32(&reachedDestination)) {
						continue
					}

					processTask(ctx, task, destAddr, srcAddr, pid, timeout, &seqCounter, mtrReturns, &reachedDestination)

				case <-ctx.Done():
					return
				}
			}
		}(i)
	}

	// 等待完成
	wg.Wait()

	// 构建最终结果
	buildResult(result, mtrReturns, destAddr, count)

	return result
}

// safeReturn 线程安全的返回结构
type safeReturn struct {
	mu        sync.RWMutex
	mtrReturn *MtrReturn
}

// task 表示一个ICMP测试任务
type task struct {
	ttl int
	snt int
}

// processTask 处理单个任务
func processTask(ctx context.Context, t task, destAddr, srcAddr string, pid int, timeout time.Duration, seqCounter *int64, mtrReturns []*safeReturn, reachedDestination *int32) {
	seq := int(atomic.AddInt64(seqCounter, 1))

	// 检查上下文是否已取消
	select {
	case <-ctx.Done():
		return
	default:
	}

	// 执行ICMP测试
	hopReturn, err := icmp.Icmp(destAddr, srcAddr, t.ttl, pid, timeout, seq, false)
	if err != nil || !hopReturn.Success {
		return
	}

	// 更新结果
	safeRet := mtrReturns[t.ttl]
	safeRet.mu.Lock()
	defer safeRet.mu.Unlock()

	ret := safeRet.mtrReturn
	ret.host = hopReturn.Addr
	ret.lastTime = hopReturn.Elapsed
	ret.allTime = append(ret.allTime, hopReturn.Elapsed)
	ret.succSum++

	if ret.worstTime == time.Duration(0) || hopReturn.Elapsed > ret.worstTime {
		ret.worstTime = hopReturn.Elapsed
	}
	if ret.bestTime == time.Duration(0) || hopReturn.Elapsed < ret.bestTime {
		ret.bestTime = hopReturn.Elapsed
	}
	ret.sumTime += hopReturn.Elapsed
	ret.avgTime = time.Duration((int64)(ret.sumTime/time.Microsecond)/(int64)(ret.succSum)) * time.Microsecond
	ret.success = true

	// 检查是否到达目标地址
	if common.IsEqualIP(hopReturn.Addr, destAddr) {
		atomic.StoreInt32(reachedDestination, safeIntToInt32(t.ttl))
	}
}

// buildResult 构建最终结果
func buildResult(result *MtrResult, mtrReturns []*safeReturn, destAddr string, count int) {
	for index := 1; index < len(mtrReturns); index++ {
		safeRet := mtrReturns[index]
		if safeRet == nil {
			break
		}

		safeRet.mu.RLock()
		mtrReturn := safeRet.mtrReturn
		if mtrReturn == nil {
			safeRet.mu.RUnlock()
			break
		}

		hop := common.IcmpHop{TTL: mtrReturn.ttl, Snt: count}
		if index != 1 && index-1 < len(mtrReturns) && mtrReturns[index-1] != nil {
			mtrReturns[index-1].mu.RLock()
			hop.AddressFrom = mtrReturns[index-1].mtrReturn.host
			mtrReturns[index-1].mu.RUnlock()
		} else {
			hop.AddressFrom = mtrReturn.host
		}

		hop.AddressTo = mtrReturn.host
		hop.Success = mtrReturn.success
		hop.LastTime = mtrReturn.lastTime
		hop.SumTime = mtrReturn.sumTime
		hop.AvgTime = mtrReturn.avgTime
		hop.BestTime = mtrReturn.bestTime
		hop.WorstTime = mtrReturn.worstTime

		// 计算统计信息
		if len(mtrReturn.allTime) > 0 {
			hop.SquaredDeviationTime = time.Duration(common.TimeSquaredDeviation(mtrReturn.allTime))
			hop.UncorrectedSDTime = time.Duration(common.TimeUncorrectedDeviation(mtrReturn.allTime))
			hop.CorrectedSDTime = time.Duration(common.TimeCorrectedDeviation(mtrReturn.allTime))
			hop.RangeTime = time.Duration(common.TimeRange(mtrReturn.allTime))
		}

		failSum := count - mtrReturn.succSum
		hop.SntFail = failSum
		loss := (float64)(failSum) / (float64)(count)
		hop.Loss = float64(loss)

		safeRet.mu.RUnlock()

		result.Hops = append(result.Hops, hop)

		// 只有成功的hop才添加到HopSummaryMap
		if hop.Success {
			summaryKey := fmt.Sprintf("%d_%s", hop.TTL, hop.AddressTo)
			result.HopSummaryMap[summaryKey] = &common.IcmpSummary{
				AddressFrom: hop.AddressFrom,
				AddressTo:   hop.AddressTo,
				Snt:         hop.Snt,
				SntFail:     hop.SntFail,
				SntTime:     hop.SumTime,
			}
		}

		if common.IsEqualIP(hop.AddressTo, destAddr) {
			break
		}
	}
}

// RunMTRWithBatching 执行批处理版本的并发MTR
func RunMTRWithBatching(destAddr, srcAddr string, timeout time.Duration, maxHops, count int, batchSize int) *MtrResult {
	options := &ConcurrentMTROptions{
		MaxWorkers:     maxHops / 2,
		BatchSize:      batchSize,
		EarlyStop:      true,
		ProgressReport: false,
		Timeout:        time.Duration(count) * timeout * 2, // 动态调整超时时间
	}

	if options.MaxWorkers < 1 {
		options.MaxWorkers = 1
	}
	if options.MaxWorkers > 15 {
		options.MaxWorkers = 15
	}

	return RunMTRConcurrent(destAddr, srcAddr, timeout, maxHops, count, options)
}

func safeIntToInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
