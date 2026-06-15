package ping

import (
	"time"
	"network_exporter/pkg/common"
	"network_exporter/pkg/icmp"
)

// Ping 使用真实ICMP执行ping操作
func Ping(addr string, ip string, timeout time.Duration, count int) *PingResult {
	result := &PingResult{
		DestAddr: addr,
		DestIp:   ip,
	}

	// 使用ICMP ID
	icmpID := &common.IcmpID{}
	pid := int(icmpID.Get())
	
	var allTimes []time.Duration
	successCount := 0
	
	// 执行多次ping
	for seq := 0; seq < count; seq++ {
		icmpReturn, err := icmp.Icmp(ip, "", 128, pid, timeout, seq, false)
		
		if err == nil && icmpReturn.Success && common.IsEqualIP(ip, icmpReturn.Addr) {
			allTimes = append(allTimes, icmpReturn.Elapsed)
			successCount++
		}
	}

	result.SntSummary = count
	result.SntFailSummary = count - successCount
	result.Success = successCount > 0

	if result.Success {
		result.DropRate = float64(count-successCount) / float64(count)
		
		// 计算时间统计
		if len(allTimes) > 0 {
			result.BestTime = allTimes[0]
			result.WorstTime = allTimes[0]
			
			sumTime := time.Duration(0)
			for _, t := range allTimes {
				if t < result.BestTime {
					result.BestTime = t
				}
				if t > result.WorstTime {
					result.WorstTime = t
				}
				sumTime += t
			}
			
			result.SumTime = sumTime
			result.AvgTime = time.Duration(int64(sumTime) / int64(len(allTimes)))
			result.SquaredDeviationTime = time.Duration(common.TimeSquaredDeviation(allTimes))
			result.UncorrectedSDTime = time.Duration(common.TimeUncorrectedDeviation(allTimes))
			result.CorrectedSDTime = time.Duration(common.TimeCorrectedDeviation(allTimes))
			result.RangeTime = common.TimeRange(allTimes)
			result.SntTimeSummary = sumTime
		}
	} else {
		result.DropRate = 1.0
		result.SntFailSummary = count
	}

	return result
} 