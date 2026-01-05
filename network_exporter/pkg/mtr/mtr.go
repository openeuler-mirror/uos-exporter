package mtr

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"network_exporter/pkg/common"
	"network_exporter/pkg/icmp"
)

// RunMTR 执行真实的MTR路由跟踪（并发版本）
func RunMTR(destAddr, srcAddr string, timeout time.Duration, maxHops, count int) *MtrResult {
	result := &MtrResult{
		DestAddr:      destAddr,
		Hops:          []common.IcmpHop{},
		HopSummaryMap: make(map[string]*common.IcmpSummary),
	}

	// 使用ICMP ID
	icmpID := &common.IcmpID{}
	pid := int(icmpID.Get())
	
	mtrReturns := make([]*MtrReturn, maxHops+1)
	var mtrMutex sync.Mutex // 保护mtrReturns的并发访问
	
	// 原子计数器用于生成唯一的seq
	var seqCounter int64
	
	// 创建工作任务通道
	type task struct {
		ttl int
		snt int
	}
	
	taskChan := make(chan task, maxHops*count)
	
	// 生成所有任务
	for snt := 0; snt < count; snt++ {
		for ttl := 1; ttl < maxHops; ttl++ {
			taskChan <- task{ttl: ttl, snt: snt}
		}
	}
	close(taskChan)
	
	// 并发工作器数量，根据maxHops调整
	workerCount := maxHops
	if workerCount > 20 {
		workerCount = 20 // 限制最大并发数
	}
	
	var wg sync.WaitGroup
	hasPermissionError := false
	var permissionMutex sync.Mutex
	
	// 启动工作器
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			for task := range taskChan {
				ttl := task.ttl
				seq := int(atomic.AddInt64(&seqCounter, 1))
				
				// 初始化mtrReturn结构
				mtrMutex.Lock()
				if mtrReturns[ttl] == nil {
					mtrReturns[ttl] = &MtrReturn{
						ttl:       ttl,
						host:      "unknown",
						succSum:   0,
						success:   false,
						lastTime:  time.Duration(0),
						sumTime:   time.Duration(0),
						bestTime:  time.Duration(0),
						worstTime: time.Duration(0),
						avgTime:   time.Duration(0),
						allTime:   make([]time.Duration, 0),
					}
				}
				mtrMutex.Unlock()
				
				// 执行ICMP测试
				hopReturn, err := icmp.Icmp(destAddr, srcAddr, ttl, pid, timeout, seq, false)
				
				if err != nil {
					// 检查是否是权限错误
					permissionMutex.Lock()
					if !hasPermissionError {
						hasPermissionError = true
					}
					permissionMutex.Unlock()
					continue
				}
				
				if !hopReturn.Success {
					continue
				}
				
				// 更新结果（需要加锁）
				mtrMutex.Lock()
				mtrReturn := mtrReturns[ttl]
				mtrReturn.host = hopReturn.Addr
				mtrReturn.lastTime = hopReturn.Elapsed
				mtrReturn.allTime = append(mtrReturn.allTime, hopReturn.Elapsed)
				mtrReturn.succSum = mtrReturn.succSum + 1
				
				if mtrReturn.worstTime == time.Duration(0) || hopReturn.Elapsed > mtrReturn.worstTime {
					mtrReturn.worstTime = hopReturn.Elapsed
				}
				if mtrReturn.bestTime == time.Duration(0) || hopReturn.Elapsed < mtrReturn.bestTime {
					mtrReturn.bestTime = hopReturn.Elapsed
				}
				mtrReturn.sumTime += hopReturn.Elapsed
				mtrReturn.avgTime = time.Duration((int64)(mtrReturn.sumTime/time.Microsecond)/(int64)(mtrReturn.succSum)) * time.Microsecond
				mtrReturn.success = true
				mtrMutex.Unlock()
				
				// 如果到达目标地址，可以考虑提前结束某些测试
				if common.IsEqualIP(hopReturn.Addr, destAddr) {
					// 这里可以添加提前结束的逻辑，但为了保持统计准确性，我们继续执行
				}
			}
		}()
	}
	
	// 等待所有工作器完成
	wg.Wait()

	// 构建结果
	for index, mtrReturn := range mtrReturns {
		if index == 0 {
			continue
		}

		if mtrReturn == nil {
			break
		}

		hop := common.IcmpHop{TTL: mtrReturn.ttl, Snt: count}
		if index != 1 {
			hop.AddressFrom = mtrReturns[index-1].host
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
		hop.SquaredDeviationTime = time.Duration(common.TimeSquaredDeviation(mtrReturn.allTime))
		hop.UncorrectedSDTime = time.Duration(common.TimeUncorrectedDeviation(mtrReturn.allTime))
		hop.CorrectedSDTime = time.Duration(common.TimeCorrectedDeviation(mtrReturn.allTime))
		hop.RangeTime = time.Duration(common.TimeRange(mtrReturn.allTime))

		failSum := count - mtrReturn.succSum
		hop.SntFail = failSum
		loss := (float64)(failSum) / (float64)(count)
		hop.Loss = float64(loss)

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

	return result
}

// RunMTRSequential 执行顺序版本的MTR路由跟踪（保留原始实现作为备选）
func RunMTRSequential(destAddr, srcAddr string, timeout time.Duration, maxHops, count int) *MtrResult {
	result := &MtrResult{
		DestAddr:      destAddr,
		Hops:          []common.IcmpHop{},
		HopSummaryMap: make(map[string]*common.IcmpSummary),
	}

	// 使用ICMP ID
	icmpID := &common.IcmpID{}
	pid := int(icmpID.Get())
	
	mtrReturns := make([]*MtrReturn, maxHops+1)

	// 执行多轮测试
	seq := 0
	hasPermissionError := false
	
	for snt := 0; snt < count; snt++ {
		for ttl := 1; ttl < maxHops; ttl++ {
			if mtrReturns[ttl] == nil {
				mtrReturns[ttl] = &MtrReturn{
					ttl:       ttl,
					host:      "unknown",
					succSum:   0,
					success:   false,
					lastTime:  time.Duration(0),
					sumTime:   time.Duration(0),
					bestTime:  time.Duration(0),
					worstTime: time.Duration(0),
					avgTime:   time.Duration(0),
				}
			}

			hopReturn, err := icmp.Icmp(destAddr, srcAddr, ttl, pid, timeout, seq, false)
			seq++ // 确保每次ICMP调用都有不同的seq
			
			if err != nil {
				// 检查是否是权限错误
				if !hasPermissionError {
					hasPermissionError = true
				}
				continue
			}
			
			if !hopReturn.Success {
				continue
			}

			mtrReturns[ttl].host = hopReturn.Addr
			mtrReturns[ttl].lastTime = hopReturn.Elapsed
			mtrReturns[ttl].allTime = append(mtrReturns[ttl].allTime, hopReturn.Elapsed)
			mtrReturns[ttl].succSum = mtrReturns[ttl].succSum + 1
			
			if mtrReturns[ttl].worstTime == time.Duration(0) || hopReturn.Elapsed > mtrReturns[ttl].worstTime {
				mtrReturns[ttl].worstTime = hopReturn.Elapsed
			}
			if mtrReturns[ttl].bestTime == time.Duration(0) || hopReturn.Elapsed < mtrReturns[ttl].bestTime {
				mtrReturns[ttl].bestTime = hopReturn.Elapsed
			}
			mtrReturns[ttl].sumTime += hopReturn.Elapsed
			mtrReturns[ttl].avgTime = time.Duration((int64)(mtrReturns[ttl].sumTime/time.Microsecond)/(int64)(mtrReturns[ttl].succSum)) * time.Microsecond
			mtrReturns[ttl].success = true

			if common.IsEqualIP(hopReturn.Addr, destAddr) {
				break
			}
		}
	}

	// 构建结果
	for index, mtrReturn := range mtrReturns {
		if index == 0 {
			continue
		}

		if mtrReturn == nil {
			break
		}

		hop := common.IcmpHop{TTL: mtrReturn.ttl, Snt: count}
		if index != 1 {
			hop.AddressFrom = mtrReturns[index-1].host
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
		hop.SquaredDeviationTime = time.Duration(common.TimeSquaredDeviation(mtrReturn.allTime))
		hop.UncorrectedSDTime = time.Duration(common.TimeUncorrectedDeviation(mtrReturn.allTime))
		hop.CorrectedSDTime = time.Duration(common.TimeCorrectedDeviation(mtrReturn.allTime))
		hop.RangeTime = time.Duration(common.TimeRange(mtrReturn.allTime))

		failSum := count - mtrReturn.succSum
		hop.SntFail = failSum
		loss := (float64)(failSum) / (float64)(count)
		hop.Loss = float64(loss)

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

	return result
} 