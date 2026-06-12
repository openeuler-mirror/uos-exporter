package metrics

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// Type for numeric statistic values from the samba server
type SmbStatisticsNumeric struct {
	Name   string
	Value  float64
	Help   string
	Labels map[string]string
}

type StatisticsGeneratorSettings struct {
	DoNotExportClient       bool
	DoNotExportUser         bool
	DoNotExportEncryption   bool
	DoNotExportPid          bool
	DoNotExportShareDetails bool
}

type lockCreationEntry struct {
	UserID       int
	CreationTime time.Time
	Share        string
}

// GetSmbStatistics - Get the statistic data for prometheus out of the response data arrays
func GetSmbStatistics(lockData []LockData, processData []ProcessData, shareData []ShareData, parms Parmeters) []SmbStatisticsNumeric {
	var ret []SmbStatisticsNumeric

	var users []int
	var pids []int
	var shares []string
	var clients []string
	var sambaVersion string
	var cluserNodeIds []int
	var lockCreationEntries []lockCreationEntry
	locksPerShare := make(map[string]int, 0)
	processPerClient := make(map[string]int, 0)
	protocolVersionCount := make(map[string]int, 0)
	signingMethodCount := make(map[string]int, 0)
	encryptionMethodCount := make(map[string]int, 0)
	clientConnectionTime := make(map[string]int64, 0)
	pidsPerNode := make(map[int][]int, 0)
	locksPerNode := make(map[int]int)
	processPerNode := make(map[int]int)
	sharesPerNode := make(map[int]int)

	for _, lock := range lockData {
		if !intArrContains(users, lock.UserID) {
			users = append(users, lock.UserID)
		}

		if !intArrContains(pids, lock.PID) {
			pids = append(pids, lock.PID)
		}

		if !intArrContains(cluserNodeIds, lock.ClusterNodeId) {
			cluserNodeIds = append(cluserNodeIds, lock.ClusterNodeId)
		}

		if lock.ClusterNodeId > -1 {
			pidsOfNode, foundPids := pidsPerNode[lock.ClusterNodeId]
			if !foundPids {
				if !intArrContains(pidsOfNode, lock.PID) {
					pidsPerNode[lock.ClusterNodeId] = append(pidsPerNode[lock.ClusterNodeId], lock.PID)
				}
			}

			locksCount, foundLocks := locksPerNode[lock.ClusterNodeId]
			if foundLocks {
				locksPerNode[lock.ClusterNodeId] = locksCount + 1
			} else {
				locksPerNode[lock.ClusterNodeId] = 1
			}
		}

		locksOfShare, found := locksPerShare[lock.SharePath]
		if !found {
			locksPerShare[lock.SharePath] = 1
		} else {
			locksPerShare[lock.SharePath] = locksOfShare + 1
		}

		newEntry := lockCreationEntry{lock.UserID, lock.Time, lock.SharePath}
		if !lockArrContainsEntry(lockCreationEntries, newEntry) {
			lockCreationEntries = append(lockCreationEntries, newEntry)
		}
	}

	for _, process := range processData {
		if !intArrContains(users, process.UserID) {
			users = append(users, process.UserID)
		}

		if !intArrContains(pids, process.PID) {
			pids = append(pids, process.PID)
		}
		sambaVersion = process.SambaVersion
		if !intArrContains(cluserNodeIds, process.ClusterNodeId) {
			cluserNodeIds = append(cluserNodeIds, process.ClusterNodeId)
		}

		if process.ClusterNodeId > -1 {
			pidsOfNode, found := pidsPerNode[process.ClusterNodeId]
			if !found {
				if !intArrContains(pidsOfNode, process.PID) {
					pidsPerNode[process.ClusterNodeId] = append(pidsPerNode[process.ClusterNodeId], process.PID)
				}
			}

			processCount, processFound := processPerNode[process.ClusterNodeId]
			if processFound {
				processPerNode[process.ClusterNodeId] = processCount + 1
			} else {
				processPerNode[process.ClusterNodeId] = 1
			}
		}

		processOnShare, foundC := processPerClient[process.Machine]
		if !foundC {
			processPerClient[process.Machine] = 1
		} else {
			processPerClient[process.Machine] = processOnShare + 1
		}

		versionCount, foundV := protocolVersionCount[process.ProtocolVersion]
		if !foundV {
			protocolVersionCount[process.ProtocolVersion] = 1
		} else {
			protocolVersionCount[process.ProtocolVersion] = versionCount + 1
		}

		signingCount, foundS := signingMethodCount[process.Signing]
		if !foundS {
			signingMethodCount[process.Signing] = 1
		} else {
			signingMethodCount[process.Signing] = signingCount + 1
		}

		encryptionCount, foundE := encryptionMethodCount[process.Encryption]
		if !foundE {
			encryptionMethodCount[process.Encryption] = 1
		} else {
			encryptionMethodCount[process.Encryption] = encryptionCount + 1
		}
	}

	for _, share := range shareData {
		if !intArrContains(pids, share.PID) {
			pids = append(pids, share.PID)
		}

		if !intArrContains(cluserNodeIds, share.ClusterNodeId) {
			cluserNodeIds = append(cluserNodeIds, share.ClusterNodeId)
		}

		if share.ClusterNodeId > -1 {
			pidsOfNode, found := pidsPerNode[share.ClusterNodeId]
			if !found {
				if !intArrContains(pidsOfNode, share.PID) {
					pidsPerNode[share.ClusterNodeId] = append(pidsPerNode[share.ClusterNodeId], share.PID)
				}
			}

			sharesCount, foundShares := sharesPerNode[share.ClusterNodeId]
			if foundShares {
				sharesPerNode[share.ClusterNodeId] = sharesCount + 1
			} else {
				sharesPerNode[share.ClusterNodeId] = 1
			}
		}

		if !strArrContains(shares, share.Service) {
			shares = append(shares, share.Service)
		}

		if !strArrContains(clients, share.Machine) {
			clients = append(clients, share.Machine)
		}

		_, foundC := clientConnectionTime[share.Machine]
		if !foundC {
			clientConnectionTime[share.Machine] = share.ConnectedAt.Unix()
		}
	}

	clusterMode := false
	if len(cluserNodeIds) > 1 || (len(cluserNodeIds) == 1 && cluserNodeIds[0] != -1) {
		clusterMode = true
	}

	// Sanity check, if running in cluster mode, no ClusterNodeId should be -1
	if clusterMode {
		if intArrContains(cluserNodeIds, -1) {
			fmt.Println("warning: ClusterNodeId -1 detected while running in cluster mode")
		}
	}

	// TODO: Generate more metrics
	ret = append(ret, SmbStatisticsNumeric{"individual_user_count", float64(len(users)), "The number of users connected to this samba server", nil})
	ret = append(ret, SmbStatisticsNumeric{"locked_file_count", float64(len(lockData)), "Number of files locked by the samba server", nil})
	ret = append(ret, SmbStatisticsNumeric{"share_count", float64(len(shares)), "Number of shares servered by the samba server", nil})
	ret = append(ret, SmbStatisticsNumeric{"client_count", float64(len(clients)), "Number of clients using the samba server", nil})

	if clusterMode {
		ret = append(ret, SmbStatisticsNumeric{"cluster_node_count", float64(len(cluserNodeIds)), "Number of cluster nodes running the samba cluster", nil})
		for node, pids := range pidsPerNode {
			ret = append(ret, SmbStatisticsNumeric{"pids_per_node_count", float64(len(pids)), "Number of PIDs per cluster node", map[string]string{"node": fmt.Sprint(node)}})
		}

		for node, locks := range locksPerNode {
			ret = append(ret, SmbStatisticsNumeric{"locks_per_node_count", float64(locks), "Number of Locks per cluster node", map[string]string{"node": fmt.Sprint(node)}})
		}

		for node, processes := range processPerNode {
			ret = append(ret, SmbStatisticsNumeric{"processes_per_node_count", float64(processes), "Number of Locks per cluster node", map[string]string{"node": fmt.Sprint(node)}})
		}

		for node, shares := range sharesPerNode {
			ret = append(ret, SmbStatisticsNumeric{"shares_per_node_count", float64(shares), "Number of Shares per cluster node", map[string]string{"node": fmt.Sprint(node)}})
		}

	} else {
		ret = append(ret, SmbStatisticsNumeric{"pid_count", float64(len(pids)), "Number of processes running by the samba server", nil})
	}

	ret = append(ret, SmbStatisticsNumeric{"server_information", 1, "Version of the samba server", map[string]string{"version": sambaVersion}})

	if !parms.Do_not_expose_share_details {
		if len(locksPerShare) > 0 {
			for share, locks := range locksPerShare {
				ret = append(ret, SmbStatisticsNumeric{"locks_per_share_count", float64(locks), "Number of locks on share", map[string]string{"share": share}})
			}
		} else {
			// Add this value even if no locks found, so prometheus description will be created
			ret = append(ret, SmbStatisticsNumeric{"locks_per_share_count", float64(0), "Number of locks on share", map[string]string{"share": ""}})
		}
	}

	if !parms.Do_not_expose_encryption {
		if len(protocolVersionCount) > 0 {
			for version, count := range protocolVersionCount {
				ret = append(ret, SmbStatisticsNumeric{"protocol_version_count", float64(count), "Number of processes on the server using the protocol", map[string]string{"protocol_version": version}})
			}
		} else {
			ret = append(ret, SmbStatisticsNumeric{"protocol_version_count", float64(0), "Number of processes on the server using the protocol", map[string]string{"protocol_version": ""}})
		}

		if len(signingMethodCount) > 0 {
			for method, count := range signingMethodCount {
				ret = append(ret, SmbStatisticsNumeric{"signing_method_count", float64(count), "Number of processes on the server using the signing", map[string]string{"signing": method}})
			}
		} else {
			ret = append(ret, SmbStatisticsNumeric{"signing_method_count", float64(0), "Number of processes on the server using the signing", map[string]string{"signing": ""}})
		}

		if len(encryptionMethodCount) > 0 {
			for method, count := range encryptionMethodCount {
				ret = append(ret, SmbStatisticsNumeric{"encryption_method_count", float64(count), "Number of processes on the server using the encryption", map[string]string{"encryption": method}})
			}
		} else {
			ret = append(ret, SmbStatisticsNumeric{"encryption_method_count", float64(0), "Number of processes on the server using the encryption", map[string]string{"encryption": ""}})
		}
	}

	if !parms.Do_not_expose_client {
		if len(processPerClient) > 0 {
			for client, count := range processPerClient {
				ret = append(ret, SmbStatisticsNumeric{"process_per_client_count", float64(count), "Number of processes on the server used by one client", map[string]string{"client": client}})
			}
		} else {
			ret = append(ret, SmbStatisticsNumeric{"process_per_client_count", float64(0), "Number of processes on the server used by one client", map[string]string{"client": ""}})
		}

		if len(clientConnectionTime) > 0 {
			for client, connectTime := range clientConnectionTime {
				ret = append(ret, SmbStatisticsNumeric{"client_connected_at", float64(connectTime), "Unix time stamp a client connected", map[string]string{"client": client}})
				now := time.Now()
				connected_since := now.Sub(time.Unix(connectTime, 0))
				ret = append(ret, SmbStatisticsNumeric{"client_connected_since_seconds", connected_since.Seconds(), "Seconds since a client connected", map[string]string{"client": client}})
			}
		} else {
			// Add this values even if no locks found, so prometheus description will be created
			ret = append(ret, SmbStatisticsNumeric{"client_connected_at", float64(0), "Unix time stamp a client connected", map[string]string{"client": ""}})
			ret = append(ret, SmbStatisticsNumeric{"client_connected_since_seconds", float64(0), "Seconds since a client connected", map[string]string{"client": ""}})
		}
	}

	if !(parms.Do_not_expose_user || parms.Do_not_expose_share_details) {
		if len(lockCreationEntries) > 0 {
			for _, lockEntry := range lockCreationEntries {
				ret = append(ret, SmbStatisticsNumeric{"lock_created_at", float64(lockEntry.CreationTime.Unix()),
					"Unix time stamp a lock was created",
					map[string]string{"user": strconv.Itoa(lockEntry.UserID), "share": lockEntry.Share}})

				ret = append(ret, SmbStatisticsNumeric{"lock_created_since_seconds", float64(time.Since(lockEntry.CreationTime).Seconds()),
					"Seconds since a lock was created",
					map[string]string{"user": strconv.Itoa(lockEntry.UserID), "share": lockEntry.Share}})
			}
		} else {
			ret = append(ret, SmbStatisticsNumeric{"lock_created_at", float64(0), "Unix time stamp a lock was created", map[string]string{"user": "", "share": ""}})
			ret = append(ret, SmbStatisticsNumeric{"lock_created_since_seconds", float64(0), "Seconds since a lock was created", map[string]string{"user": "", "share": ""}})
		}
	}

	return ret
}

func intArrContains(arr []int, value int) bool {
	for _, field := range arr {
		if field == value {
			return true
		}
	}

	return false
}

func lockArrContainsEntry(arr []lockCreationEntry, value lockCreationEntry) bool {
	for _, field := range arr {
		if field.Share == value.Share && field.UserID == value.UserID {
			return true
		}
	}

	return false
}

func strArrContains(arr []string, value string) bool {
	for _, field := range arr {
		if field == value {
			return true
		}
	}

	return false
}

const smbd_image_name = "smbd"

func GetSmbdMetrics(pidDataList []PsUtilPidData, notExportPid bool) []SmbStatisticsNumeric {

	var ret []SmbStatisticsNumeric

	ret = append(ret, SmbStatisticsNumeric{"smbd_unique_process_id_count", float64(len(pidDataList)), fmt.Sprintf("Count of unique process IDs for '%s'", smbd_image_name), nil})

	if len(pidDataList) > 0 {
		cpuPercentageSum := float64(0)
		vmBytesSum := uint64(0)
		vmPercentSum := float64(0)
		readCountSum := uint64(0)
		writeCountSum := uint64(0)
		readBytesSum := uint64(0)
		writeBytesSum := uint64(0)
		openFilesCountSum := uint64(0)
		threadCountSum := uint64(0)
		for _, pidData := range pidDataList {

			cpuPercentageSum += pidData.CpuUsagePercent
			vmBytesSum += pidData.VirtualMemoryUsageBytes
			vmPercentSum += pidData.VirtualMemoryUsagePercent
			readCountSum += pidData.IoCounterReadCount
			writeCountSum += pidData.IoCounterWriteCount
			readBytesSum += pidData.IoCounterReadBytes
			writeBytesSum += pidData.IoCounterWriteBytes
			openFilesCountSum += pidData.OpenFilesCount
			threadCountSum += pidData.ThreadCount

			if !notExportPid {
				// Metrics with PID label
				ret = append(ret, SmbStatisticsNumeric{"smbd_cpu_usage_percentage",
					pidData.CpuUsagePercent, fmt.Sprintf("CPU usage of the '%s' process with pid in percent", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_virtual_memory_usage_bytes",
					float64(pidData.VirtualMemoryUsageBytes), fmt.Sprintf("Virtual memory usage of the '%s' process with pid in bytes", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_virtual_memory_usage_percent",
					pidData.VirtualMemoryUsagePercent, fmt.Sprintf("Virtual memory usage of the '%s' process with pid in percent", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_read_count",
					float64(pidData.IoCounterReadCount), fmt.Sprintf("IO counter read count of the process '%s'", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_write_count",
					float64(pidData.IoCounterWriteCount), fmt.Sprintf("IO counter write count of the process '%s'", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_read_bytes",
					float64(pidData.IoCounterReadBytes), fmt.Sprintf("IO counter reads of the process '%s' in byte", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_write_bytes",
					float64(pidData.IoCounterWriteBytes), fmt.Sprintf("IO counter writes of the process '%s' in byte", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_open_file_count",
					float64(pidData.OpenFilesCount), fmt.Sprintf("Open file handles by process '%s'", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
				ret = append(ret, SmbStatisticsNumeric{"smbd_thread_count",
					float64(pidData.ThreadCount), fmt.Sprintf("Threads used by process '%s'", smbd_image_name),
					map[string]string{"pid": strconv.Itoa(int(pidData.PID))}})
			}
		}

		// Add sum metrics (without label)
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_cpu_usage_percentage",
			cpuPercentageSum, fmt.Sprintf("Sum CPU usage of all '%s' processes in percent", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_virtual_memory_usage_bytes",
			float64(vmBytesSum), fmt.Sprintf("Virtual memory usage of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_virtual_memory_usage_percent",
			vmPercentSum, fmt.Sprintf("Virtual memory usage of all '%s' processes in percent", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_read_count",
			float64(readCountSum), fmt.Sprintf("IO counter read count of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_write_count",
			float64(writeCountSum), fmt.Sprintf("IO counter write count of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_read_bytes",
			float64(readBytesSum), fmt.Sprintf("IO counter reads of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_write_bytes",
			float64(writeBytesSum), fmt.Sprintf("IO counter writes of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_open_file_count",
			float64(openFilesCountSum), fmt.Sprintf("Open file handles of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_thread_count",
			float64(threadCountSum), fmt.Sprintf("Threads used by all '%s' processes", smbd_image_name), nil})

	} else {
		// Give back empty metrics, when smbd is not running
		if !notExportPid {
			// Metrics with PID labels
			ret = append(ret, SmbStatisticsNumeric{"smbd_cpu_usage_percentage",
				0, fmt.Sprintf("CPU usage of the '%s' process with pid in percent", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_virtual_memory_usage_bytes",
				0, fmt.Sprintf("Virtual memory usage of the '%s' process with pid in bytes", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_virtual_memory_usage_percent",
				0, fmt.Sprintf("Virtual memory usage of the '%s' process with pid in percent", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_read_count",
				0, fmt.Sprintf("IO counter read count of the process '%s'", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_write_count",
				0, fmt.Sprintf("IO counter write count of the process '%s'", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_read_bytes",
				0, fmt.Sprintf("IO counter reads of the process '%s' in byte", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_io_counter_write_bytes",
				0, fmt.Sprintf("IO counter writes of the process '%s' in byte", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_open_file_count",
				0, fmt.Sprintf("Open file handles by process '%s'", smbd_image_name),
				map[string]string{"pid": ""}})
			ret = append(ret, SmbStatisticsNumeric{"smbd_thread_count",
				0, fmt.Sprintf("Threads used by process '%s'", smbd_image_name),
				map[string]string{"pid": ""}})
		}

		// Metrics without labels (sum metrics)
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_cpu_usage_percentage",
			0, fmt.Sprintf("Sum CPU usage of all '%s' processes in percent", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_virtual_memory_usage_bytes",
			0, fmt.Sprintf("Virtual memory usage of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_virtual_memory_usage_percent",
			0, fmt.Sprintf("Virtual memory usage of all '%s' processes in percent", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_read_count",
			0, fmt.Sprintf("IO counter read count of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_write_count",
			0, fmt.Sprintf("IO counter write count of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_read_bytes",
			0, fmt.Sprintf("IO counter reads of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_io_counter_write_bytes",
			0, fmt.Sprintf("IO counter writes of all '%s' processes in bytes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_open_file_count",
			0, fmt.Sprintf("Open file handles of all '%s' processes", smbd_image_name), nil})
		ret = append(ret, SmbStatisticsNumeric{"smbd_sum_thread_count",
			0, fmt.Sprintf("Threads used by all '%s' processes", smbd_image_name), nil})
	}

	return ret
}

// Class to get commonbl.PsUtilPidData for a Process
type PsDataGenerator struct {
	ProcessToRequest string
	pgrepPath        string
}

// Get a new instance of PsDataGenerator
func NewPsDataGenerator(processToRequest string) (*PsDataGenerator, error) {
	ret := PsDataGenerator{}
	var errLookPath error
	ret.ProcessToRequest = processToRequest
	ret.pgrepPath, errLookPath = exec.LookPath("pgrep")
	if errLookPath != nil {
		return nil, errLookPath
	}

	return &ret, nil
}

// Get the commonbl.PsUtilPidData data of the ProcessToRequest.
// - In case this process is not running an empty list is returned
// - In case an error, other then not finding the process, occurs during gathering data it is returned
func (generator *PsDataGenerator) GetPsUtilPidData() ([]PsUtilPidData, error) {
	ret := []PsUtilPidData{}

	pidList, pidListErr := generator.getPidList()
	if (pidListErr != nil) || (len(pidList) == 0) {
		return ret, nil
	}

	for _, pid := range pidList {

		proc, errProc := process.NewProcess(pid)
		if errProc != nil {
			return nil, errProc
		}

		cpuPercent, errPer := proc.CPUPercent()
		if errPer != nil {
			return nil, errPer
		}
		vmBytes, errVmBytes := proc.MemoryInfo()
		if errVmBytes != nil {
			return nil, errVmBytes
		}
		vmPercent, errVmPercent := proc.MemoryPercent()
		if errVmPercent != nil {
			return nil, errVmPercent
		}
		ioCounters, errIoCounters := proc.IOCounters()
		if errIoCounters != nil {
			return nil, errIoCounters
		}
		openFileStats, errOpenFileStats := proc.OpenFiles()
		if errOpenFileStats != nil {
			return nil, errOpenFileStats
		}
		threadStats, errThreadStats := proc.Threads()
		if errThreadStats != nil {
			return nil, errThreadStats
		}

		entry := PsUtilPidData{
			int64(pid),
			cpuPercent,
			vmBytes.VMS,
			float64(vmPercent),
			ioCounters.ReadCount,
			ioCounters.ReadBytes,
			ioCounters.WriteCount,
			ioCounters.WriteBytes,
			uint64(len(openFileStats)),
			uint64(len(threadStats)),
		}

		ret = append(ret, entry)
	}

	return ret, nil
}

func isValidProcessName(name string) bool {
	// 更严格的验证：只允许常见的进程名字符
	// 不允许空格、引号、分号、管道等特殊字符
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.][a-zA-Z0-9_.-]*$`, name)
	return matched && len(name) > 0 && len(name) < 64
}

func (generator *PsDataGenerator) getPidList() ([]int32, error) {
	var pidList []int32

	if !isValidProcessName(generator.ProcessToRequest) {
		return nil, fmt.Errorf("invalid process name: %s", generator.ProcessToRequest)
	}

	if !isValidProcessName(generator.pgrepPath) {
		return nil, fmt.Errorf("invalid pgrep path: %s", generator.pgrepPath)
	}
	//gosec G204 - Process name is rigorously validated and pgrep path is from trusted source

	pidListInByte, errGetPids := exec.Command("pgrep", PROCESS_TO_MONITOR).Output()
	if errGetPids != nil {
		return nil, errGetPids
	}

	pidListLines := strings.Split(string(pidListInByte), "\n")
	for _, line := range pidListLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, errConv := strconv.ParseInt(line, 10, 32)
		if errConv != nil {
			return nil, errConv
		} else {
			pidList = append(pidList, int32(pid))
		}
	}

	return pidList, nil
}
