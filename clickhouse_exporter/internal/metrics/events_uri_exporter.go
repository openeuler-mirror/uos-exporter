package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/prometheus/client_golang/prometheus"
)

// Exporter collects clickhouse stats from the given URI and exports them using
// the prometheus metrics package.
type EventsURIExporter struct {
	Query                                                 *prometheus.Desc
	SelectQuery                                           *prometheus.Desc
	InitialQuery                                          *prometheus.Desc
	QueriesWithSubqueries                                 *prometheus.Desc
	SelectQueriesWithSubqueries                           *prometheus.Desc
	FailedQuery                                           *prometheus.Desc
	QueryTimeMicroseconds                                 *prometheus.Desc
	SelectQueryTimeMicroseconds                           *prometheus.Desc
	OtherQueryTimeMicroseconds                            *prometheus.Desc
	FileOpen                                              *prometheus.Desc
	Seek                                                  *prometheus.Desc
	ReadBufferFromFileDescriptorRead                      *prometheus.Desc
	ReadBufferFromFileDescriptorReadBytes                 *prometheus.Desc
	WriteBufferFromFileDescriptorWrite                    *prometheus.Desc
	WriteBufferFromFileDescriptorWriteBytes               *prometheus.Desc
	FileSync                                              *prometheus.Desc
	FileSyncElapsedMicroseconds                           *prometheus.Desc
	ReadCompressedBytes                                   *prometheus.Desc
	CompressedReadBufferBlocks                            *prometheus.Desc
	CompressedReadBufferBytes                             *prometheus.Desc
	OpenedFileCacheHits                                   *prometheus.Desc
	OpenedFileCacheMisses                                 *prometheus.Desc
	OpenedFileCacheMicroseconds                           *prometheus.Desc
	IOBufferAllocs                                        *prometheus.Desc
	IOBufferAllocBytes                                    *prometheus.Desc
	ArenaAllocChunks                                      *prometheus.Desc
	ArenaAllocBytes                                       *prometheus.Desc
	FunctionExecute                                       *prometheus.Desc
	TableFunctionExecute                                  *prometheus.Desc
	CreatedReadBufferOrdinary                             *prometheus.Desc
	DiskReadElapsedMicroseconds                           *prometheus.Desc
	DiskWriteElapsedMicroseconds                          *prometheus.Desc
	NetworkReceiveElapsedMicroseconds                     *prometheus.Desc
	NetworkSendElapsedMicroseconds                        *prometheus.Desc
	NetworkReceiveBytes                                   *prometheus.Desc
	NetworkSendBytes                                      *prometheus.Desc
	GlobalThreadPoolExpansions                            *prometheus.Desc
	GlobalThreadPoolThreadCreationMicroseconds            *prometheus.Desc
	GlobalThreadPoolLockWaitMicroseconds                  *prometheus.Desc
	GlobalThreadPoolJobs                                  *prometheus.Desc
	GlobalThreadPoolJobWaitTimeMicroseconds               *prometheus.Desc
	LocalThreadPoolExpansions                             *prometheus.Desc
	LocalThreadPoolShrinks                                *prometheus.Desc
	LocalThreadPoolThreadCreationMicroseconds             *prometheus.Desc
	LocalThreadPoolLockWaitMicroseconds                   *prometheus.Desc
	LocalThreadPoolJobs                                   *prometheus.Desc
	LocalThreadPoolBusyMicroseconds                       *prometheus.Desc
	LocalThreadPoolJobWaitTimeMicroseconds                *prometheus.Desc
	InsertedRows                                          *prometheus.Desc
	InsertedBytes                                         *prometheus.Desc
	CompileFunction                                       *prometheus.Desc
	CompileExpressionsMicroseconds                        *prometheus.Desc
	CompileExpressionsBytes                               *prometheus.Desc
	ExternalProcessingFilesTotal                          *prometheus.Desc
	JoinBuildTableRowCount                                *prometheus.Desc
	JoinProbeTableRowCount                                *prometheus.Desc
	JoinResultRowCount                                    *prometheus.Desc
	SelectedRows                                          *prometheus.Desc
	SelectedBytes                                         *prometheus.Desc
	RowsReadByMainReader                                  *prometheus.Desc
	LoadedDataParts                                       *prometheus.Desc
	LoadedDataPartsMicroseconds                           *prometheus.Desc
	WaitMarksLoadMicroseconds                             *prometheus.Desc
	LoadedMarksFiles                                      *prometheus.Desc
	LoadedMarksCount                                      *prometheus.Desc
	LoadedMarksMemoryBytes                                *prometheus.Desc
	Merge                                                 *prometheus.Desc
	MergeSourceParts                                      *prometheus.Desc
	MergedRows                                            *prometheus.Desc
	MergedColumns                                         *prometheus.Desc
	GatheredColumns                                       *prometheus.Desc
	MergedUncompressedBytes                               *prometheus.Desc
	MergeTotalMilliseconds                                *prometheus.Desc
	MergeExecuteMilliseconds                              *prometheus.Desc
	MergeHorizontalStageTotalMilliseconds                 *prometheus.Desc
	MergeHorizontalStageExecuteMilliseconds               *prometheus.Desc
	MergeVerticalStageTotalMilliseconds                   *prometheus.Desc
	MergeVerticalStageExecuteMilliseconds                 *prometheus.Desc
	MergeProjectionStageTotalMilliseconds                 *prometheus.Desc
	MergeProjectionStageExecuteMilliseconds               *prometheus.Desc
	MergingSortedMilliseconds                             *prometheus.Desc
	GatheringColumnMilliseconds                           *prometheus.Desc
	MergeTreeDataWriterRows                               *prometheus.Desc
	MergeTreeDataWriterUncompressedBytes                  *prometheus.Desc
	MergeTreeDataWriterCompressedBytes                    *prometheus.Desc
	MergeTreeDataWriterBlocks                             *prometheus.Desc
	MergeTreeDataWriterBlocksAlreadySorted                *prometheus.Desc
	MergeTreeDataWriterSortingBlocksMicroseconds          *prometheus.Desc
	MergeTreeDataWriterMergingBlocksMicroseconds          *prometheus.Desc
	InsertedWideParts                                     *prometheus.Desc
	InsertedCompactParts                                  *prometheus.Desc
	MergedIntoWideParts                                   *prometheus.Desc
	MergedIntoCompactParts                                *prometheus.Desc
	ContextLock                                           *prometheus.Desc
	ContextLockWaitMicroseconds                           *prometheus.Desc
	SystemLogErrorOnFlush                                 *prometheus.Desc
	RWLockAcquiredReadLocks                               *prometheus.Desc
	RWLockReadersWaitMilliseconds                         *prometheus.Desc
	PartsLockHoldMicroseconds                             *prometheus.Desc
	PartsLockWaitMicroseconds                             *prometheus.Desc
	RealTimeMicroseconds                                  *prometheus.Desc
	UserTimeMicroseconds                                  *prometheus.Desc
	SystemTimeMicroseconds                                *prometheus.Desc
	SoftPageFaults                                        *prometheus.Desc
	HardPageFaults                                        *prometheus.Desc
	OSCPUWaitMicroseconds                                 *prometheus.Desc
	OSCPUVirtualTimeMicroseconds                          *prometheus.Desc
	OSReadBytes                                           *prometheus.Desc
	OSWriteBytes                                          *prometheus.Desc
	OSReadChars                                           *prometheus.Desc
	OSWriteChars                                          *prometheus.Desc
	QueryProfilerSignalOverruns                           *prometheus.Desc
	QueryProfilerRuns                                     *prometheus.Desc
	QueryMemoryLimitExceeded                              *prometheus.Desc
	ThreadPoolReaderPageCacheHitElapsedMicroseconds       *prometheus.Desc
	ThreadPoolReaderPageCacheMiss                         *prometheus.Desc
	ThreadPoolReaderPageCacheMissBytes                    *prometheus.Desc
	ThreadPoolReaderPageCacheMissElapsedMicroseconds      *prometheus.Desc
	SynchronousReadWaitMicroseconds                       *prometheus.Desc
	MainConfigLoads                                       *prometheus.Desc
	ServerStartupMilliseconds                             *prometheus.Desc
	MergerMutatorsGetPartsForMergeElapsedMicroseconds     *prometheus.Desc
	MergerMutatorPrepareRangesForMergeElapsedMicroseconds *prometheus.Desc
	MergerMutatorSelectPartsForMergeElapsedMicroseconds   *prometheus.Desc
	MergerMutatorRangesForMergeCount                      *prometheus.Desc
	MergerMutatorPartsInRangesForMergeCount               *prometheus.Desc
	MergerMutatorSelectRangePartsCount                    *prometheus.Desc
	AsyncLoaderWaitMicroseconds                           *prometheus.Desc
	LogTrace                                              *prometheus.Desc
	LogDebug                                              *prometheus.Desc
	LogInfo                                               *prometheus.Desc
	LogWarning                                            *prometheus.Desc
	LogError                                              *prometheus.Desc
	LoggerElapsedNanoseconds                              *prometheus.Desc
	InterfaceHTTPSendBytes                                *prometheus.Desc
	InterfaceHTTPReceiveBytes                             *prometheus.Desc
	InterfaceNativeSendBytes                              *prometheus.Desc
	InterfaceNativeReceiveBytes                           *prometheus.Desc
	ConcurrencyControlSlotsGranted                        *prometheus.Desc
	ConcurrencyControlSlotsAcquired                       *prometheus.Desc
	GWPAsanAllocateFailed                                 *prometheus.Desc
	MemoryWorkerRun                                       *prometheus.Desc
	MemoryWorkerRunElapsedMicroseconds                    *prometheus.Desc
}

func NewEventsURIExporter() *EventsURIExporter {
	Query := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Query_total"),
		"Number of Query total processed",
		nil,
		nil,
	)
	SelectQuery := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SelectQuery_total"),
		"Number of SelectQuery total processed",
		nil,
		nil,
	)
	InitialQuery := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InitialQuery_total"),
		"Number of InitialQuery total processed",
		nil,
		nil,
	)
	QueriesWithSubqueries := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueriesWithSubqueries_total"),
		"Number of QueriesWithSubqueries total processed",
		nil,
		nil,
	)
	SelectQueriesWithSubqueries := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SelectQueriesWithSubqueries_total"),
		"Number of SelectQueriesWithSubqueries total processed",
		nil,
		nil,
	)
	FailedQuery := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FailedQuery_total"),
		"Number of FailedQuery total processed",
		nil,
		nil,
	)
	QueryTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryTimeMicroseconds_total"),
		"Number of QueryTimeMicroseconds total processed",
		nil,
		nil,
	)
	SelectQueryTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SelectQueryTimeMicroseconds_total"),
		"Number of SelectQueryTimeMicroseconds total processed",
		nil,
		nil,
	)
	OtherQueryTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OtherQueryTimeMicroseconds_total"),
		"Number of OtherQueryTimeMicroseconds total processed",
		nil,
		nil,
	)
	FileOpen := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FileOpen_total"),
		"Number of FileOpen total processed",
		nil,
		nil,
	)
	Seek := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Seek_total"),
		"Number of Seek total processed",
		nil,
		nil,
	)
	ReadBufferFromFileDescriptorRead := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReadBufferFromFileDescriptorRead_total"),
		"Number of ReadBufferFromFileDescriptorRead total processed",
		nil,
		nil,
	)
	ReadBufferFromFileDescriptorReadBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReadBufferFromFileDescriptorReadBytes_total"),
		"Number of ReadBufferFromFileDescriptorReadBytes total processed",
		nil,
		nil,
	)
	WriteBufferFromFileDescriptorWrite := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "WriteBufferFromFileDescriptorWrite_total"),
		"Number of WriteBufferFromFileDescriptorWrite total processed",
		nil,
		nil,
	)
	WriteBufferFromFileDescriptorWriteBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "WriteBufferFromFileDescriptorWriteBytes_total"),
		"Number of WriteBufferFromFileDescriptorWriteBytes total processed",
		nil,
		nil,
	)
	FileSync := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FileSync_total"),
		"Number of FileSync total processed",
		nil,
		nil,
	)
	FileSyncElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FileSyncElapsedMicroseconds_total"),
		"Number of FileSyncElapsedMicroseconds total processed",
		nil,
		nil,
	)
	ReadCompressedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReadCompressedBytes_total"),
		"Number of ReadCompressedBytes total processed",
		nil,
		nil,
	)
	CompressedReadBufferBlocks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompressedReadBufferBlocks_total"),
		"Number of CompressedReadBufferBlocks total processed",
		nil,
		nil,
	)
	CompressedReadBufferBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompressedReadBufferBytes_total"),
		"Number of CompressedReadBufferBytes total processed",
		nil,
		nil,
	)
	OpenedFileCacheHits := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OpenedFileCacheHits_total"),
		"Number of OpenedFileCacheHits total processed",
		nil,
		nil,
	)
	OpenedFileCacheMisses := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OpenedFileCacheMisses_total"),
		"Number of OpenedFileCacheMisses total processed",
		nil,
		nil,
	)
	OpenedFileCacheMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OpenedFileCacheMicroseconds_total"),
		"Number of OpenedFileCacheMicroseconds total processed",
		nil,
		nil,
	)
	IOBufferAllocs := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOBufferAllocs_total"),
		"Number of IOBufferAllocs total processed",
		nil,
		nil,
	)
	IOBufferAllocBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOBufferAllocBytes_total"),
		"Number of IOBufferAllocBytes total processed",
		nil,
		nil,
	)
	ArenaAllocChunks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ArenaAllocChunks_total"),
		"Number of ArenaAllocChunks total processed",
		nil,
		nil,
	)
	ArenaAllocBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ArenaAllocBytes_total"),
		"Number of ArenaAllocBytes total processed",
		nil,
		nil,
	)
	FunctionExecute := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FunctionExecute_total"),
		"Number of FunctionExecute total processed",
		nil,
		nil,
	)
	TableFunctionExecute := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TableFunctionExecute_total"),
		"Number of TableFunctionExecute total processed",
		nil,
		nil,
	)
	CreatedReadBufferOrdinary := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CreatedReadBufferOrdinary_total"),
		"Number of CreatedReadBufferOrdinary total processed",
		nil,
		nil,
	)
	DiskReadElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskReadElapsedMicroseconds_total"),
		"Number of DiskReadElapsedMicroseconds total processed",
		nil,
		nil,
	)
	DiskWriteElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskWriteElapsedMicroseconds_total"),
		"Number of DiskWriteElapsedMicroseconds total processed",
		nil,
		nil,
	)
	NetworkReceiveElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkReceiveElapsedMicroseconds_total"),
		"Number of NetworkReceiveElapsedMicroseconds total processed",
		nil,
		nil,
	)
	NetworkSendElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkSendElapsedMicroseconds_total"),
		"Number of NetworkSendElapsedMicroseconds total processed",
		nil,
		nil,
	)
	NetworkReceiveBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkReceiveBytes_total"),
		"Number of NetworkReceiveBytes total processed",
		nil,
		nil,
	)
	NetworkSendBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkSendBytes_total"),
		"Number of NetworkSendBytes total processed",
		nil,
		nil,
	)
	GlobalThreadPoolExpansions := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadPoolExpansions_total"),
		"Number of GlobalThreadPoolExpansions total processed",
		nil,
		nil,
	)
	GlobalThreadPoolThreadCreationMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadPoolThreadCreationMicroseconds_total"),
		"Number of GlobalThreadPoolThreadCreationMicroseconds total processed",
		nil,
		nil,
	)
	GlobalThreadPoolLockWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadPoolLockWaitMicroseconds_total"),
		"Number of GlobalThreadPoolLockWaitMicroseconds total processed",
		nil,
		nil,
	)
	GlobalThreadPoolJobs := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadPoolJobs_total"),
		"Number of GlobalThreadPoolJobs total processed",
		nil,
		nil,
	)
	GlobalThreadPoolJobWaitTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadPoolJobWaitTimeMicroseconds_total"),
		"Number of GlobalThreadPoolJobWaitTimeMicroseconds total processed",
		nil,
		nil,
	)
	LocalThreadPoolExpansions := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolExpansions_total"),
		"Number of LocalThreadPoolExpansions total processed",
		nil,
		nil,
	)
	LocalThreadPoolShrinks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolShrinks_total"),
		"Number of LocalThreadPoolShrinks total processed",
		nil,
		nil,
	)
	LocalThreadPoolThreadCreationMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolThreadCreationMicroseconds_total"),
		"Number of LocalThreadPoolThreadCreationMicroseconds total processed",
		nil,
		nil,
	)
	LocalThreadPoolLockWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolLockWaitMicroseconds_total"),
		"Number of LocalThreadPoolLockWaitMicroseconds total processed",
		nil,
		nil,
	)
	LocalThreadPoolJobs := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolJobs_total"),
		"Number of LocalThreadPoolJobs total processed",
		nil,
		nil,
	)
	LocalThreadPoolBusyMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolBusyMicroseconds_total"),
		"Number of LocalThreadPoolBusyMicroseconds total processed",
		nil,
		nil,
	)
	LocalThreadPoolJobWaitTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadPoolJobWaitTimeMicroseconds_total"),
		"Number of LocalThreadPoolJobWaitTimeMicroseconds total processed",
		nil,
		nil,
	)
	InsertedRows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InsertedRows_total"),
		"Number of InsertedRows total processed",
		nil,
		nil,
	)
	InsertedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InsertedBytes_total"),
		"Number of InsertedBytes total processed",
		nil,
		nil,
	)
	CompileFunction := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompileFunction_total"),
		"Number of CompileFunction total processed",
		nil,
		nil,
	)
	CompileExpressionsMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompileExpressionsMicroseconds_total"),
		"Number of CompileExpressionsMicroseconds total processed",
		nil,
		nil,
	)
	CompileExpressionsBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompileExpressionsBytes_total"),
		"Number of CompileExpressionsBytes total processed",
		nil,
		nil,
	)
	ExternalProcessingFilesTotal := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ExternalProcessingFilesTotal_total"),
		"Number of ExternalProcessingFilesTotal total processed",
		nil,
		nil,
	)
	JoinBuildTableRowCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "JoinBuildTableRowCount_total"),
		"Number of JoinBuildTableRowCount total processed",
		nil,
		nil,
	)
	JoinProbeTableRowCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "JoinProbeTableRowCount_total"),
		"Number of JoinProbeTableRowCount total processed",
		nil,
		nil,
	)
	JoinResultRowCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "JoinResultRowCount_total"),
		"Number of JoinResultRowCount total processed",
		nil,
		nil,
	)
	SelectedRows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SelectedRows_total"),
		"Number of SelectedRows total processed",
		nil,
		nil,
	)
	SelectedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SelectedBytes_total"),
		"Number of SelectedBytes total processed",
		nil,
		nil,
	)
	RowsReadByMainReader := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RowsReadByMainReader_total"),
		"Number of RowsReadByMainReader total processed",
		nil,
		nil,
	)
	LoadedDataParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoadedDataParts_total"),
		"Number of LoadedDataParts total processed",
		nil,
		nil,
	)
	LoadedDataPartsMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoadedDataPartsMicroseconds_total"),
		"Number of LoadedDataPartsMicroseconds total processed",
		nil,
		nil,
	)
	WaitMarksLoadMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "WaitMarksLoadMicroseconds_total"),
		"Number of WaitMarksLoadMicroseconds total processed",
		nil,
		nil,
	)
	LoadedMarksFiles := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoadedMarksFiles_total"),
		"Number of LoadedMarksFiles total processed",
		nil,
		nil,
	)
	LoadedMarksCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoadedMarksCount_total"),
		"Number of LoadedMarksCount total processed",
		nil,
		nil,
	)
	LoadedMarksMemoryBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoadedMarksMemoryBytes_total"),
		"Number of LoadedMarksMemoryBytes total processed",
		nil,
		nil,
	)
	Merge := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Merge_total"),
		"Number of Merge total processed",
		nil,
		nil,
	)
	MergeSourceParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeSourceParts_total"),
		"Number of MergeSourceParts total processed",
		nil,
		nil,
	)
	MergedRows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergedRows_total"),
		"Number of MergedRows total processed",
		nil,
		nil,
	)
	MergedColumns := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergedColumns_total"),
		"Number of MergedColumns total processed",
		nil,
		nil,
	)
	GatheredColumns := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GatheredColumns_total"),
		"Number of GatheredColumns total processed",
		nil,
		nil,
	)
	MergedUncompressedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergedUncompressedBytes_total"),
		"Number of MergedUncompressedBytes total processed",
		nil,
		nil,
	)
	MergeTotalMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTotalMilliseconds_total"),
		"Number of MergeTotalMilliseconds total processed",
		nil,
		nil,
	)
	MergeExecuteMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeExecuteMilliseconds_total"),
		"Number of MergeExecuteMilliseconds total processed",
		nil,
		nil,
	)
	MergeHorizontalStageTotalMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeHorizontalStageTotalMilliseconds_total"),
		"Number of MergeHorizontalStageTotalMilliseconds total processed",
		nil,
		nil,
	)
	MergeHorizontalStageExecuteMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeHorizontalStageExecuteMilliseconds_total"),
		"Number of MergeHorizontalStageExecuteMilliseconds total processed",
		nil,
		nil,
	)
	MergeVerticalStageTotalMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeVerticalStageTotalMilliseconds_total"),
		"Number of MergeVerticalStageTotalMilliseconds total processed",
		nil,
		nil,
	)
	MergeVerticalStageExecuteMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeVerticalStageExecuteMilliseconds_total"),
		"Number of MergeVerticalStageExecuteMilliseconds total processed",
		nil,
		nil,
	)
	MergeProjectionStageTotalMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeProjectionStageTotalMilliseconds_total"),
		"Number of MergeProjectionStageTotalMilliseconds total processed",
		nil,
		nil,
	)
	MergeProjectionStageExecuteMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeProjectionStageExecuteMilliseconds_total"),
		"Number of MergeProjectionStageExecuteMilliseconds total processed",
		nil,
		nil,
	)
	MergingSortedMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergingSortedMilliseconds_total"),
		"Number of MergingSortedMilliseconds total processed",
		nil,
		nil,
	)
	GatheringColumnMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GatheringColumnMilliseconds_total"),
		"Number of GatheringColumnMilliseconds total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterRows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterRows_total"),
		"Number of MergeTreeDataWriterRows total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterUncompressedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterUncompressedBytes_total"),
		"Number of MergeTreeDataWriterUncompressedBytes total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterCompressedBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterCompressedBytes_total"),
		"Number of MergeTreeDataWriterCompressedBytes total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterBlocks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterBlocks_total"),
		"Number of MergeTreeDataWriterBlocks total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterBlocksAlreadySorted := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterBlocksAlreadySorted_total"),
		"Number of MergeTreeDataWriterBlocksAlreadySorted total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterSortingBlocksMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterSortingBlocksMicroseconds_total"),
		"Number of MergeTreeDataWriterSortingBlocksMicroseconds total processed",
		nil,
		nil,
	)
	MergeTreeDataWriterMergingBlocksMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataWriterMergingBlocksMicroseconds_total"),
		"Number of MergeTreeDataWriterMergingBlocksMicroseconds total processed",
		nil,
		nil,
	)
	InsertedWideParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InsertedWideParts_total"),
		"Number of InsertedWideParts total processed",
		nil,
		nil,
	)
	InsertedCompactParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InsertedCompactParts_total"),
		"Number of InsertedCompactParts total processed",
		nil,
		nil,
	)
	MergedIntoWideParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergedIntoWideParts_total"),
		"Number of MergedIntoWideParts total processed",
		nil,
		nil,
	)
	MergedIntoCompactParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergedIntoCompactParts_total"),
		"Number of MergedIntoCompactParts total processed",
		nil,
		nil,
	)
	ContextLock := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ContextLock_total"),
		"Number of ContextLock total processed",
		nil,
		nil,
	)
	ContextLockWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ContextLockWaitMicroseconds_total"),
		"Number of ContextLockWaitMicroseconds total processed",
		nil,
		nil,
	)
	SystemLogErrorOnFlush := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SystemLogErrorOnFlush_total"),
		"Number of SystemLogErrorOnFlush total processed",
		nil,
		nil,
	)
	RWLockAcquiredReadLocks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockAcquiredReadLocks_total"),
		"Number of RWLockAcquiredReadLocks total processed",
		nil,
		nil,
	)
	RWLockReadersWaitMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockReadersWaitMilliseconds_total"),
		"Number of RWLockReadersWaitMilliseconds total processed",
		nil,
		nil,
	)
	PartsLockHoldMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsLockHoldMicroseconds_total"),
		"Number of PartsLockHoldMicroseconds total processed",
		nil,
		nil,
	)
	PartsLockWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsLockWaitMicroseconds_total"),
		"Number of PartsLockWaitMicroseconds total processed",
		nil,
		nil,
	)
	RealTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RealTimeMicroseconds_total"),
		"Number of RealTimeMicroseconds total processed",
		nil,
		nil,
	)
	UserTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "UserTimeMicroseconds_total"),
		"Number of UserTimeMicroseconds total processed",
		nil,
		nil,
	)
	SystemTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SystemTimeMicroseconds_total"),
		"Number of SystemTimeMicroseconds total processed",
		nil,
		nil,
	)
	SoftPageFaults := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SoftPageFaults_total"),
		"Number of SoftPageFaults total processed",
		nil,
		nil,
	)
	HardPageFaults := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HardPageFaults_total"),
		"Number of HardPageFaults total processed",
		nil,
		nil,
	)
	OSCPUWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSCPUWaitMicroseconds_total"),
		"Number of OSCPUWaitMicroseconds total processed",
		nil,
		nil,
	)
	OSCPUVirtualTimeMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSCPUVirtualTimeMicroseconds_total"),
		"Number of OSCPUVirtualTimeMicroseconds total processed",
		nil,
		nil,
	)
	OSReadBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSReadBytes_total"),
		"Number of OSReadBytes total processed",
		nil,
		nil,
	)
	OSWriteBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSWriteBytes_total"),
		"Number of OSWriteBytes total processed",
		nil,
		nil,
	)
	OSReadChars := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSReadChars_total"),
		"Number of OSReadChars total processed",
		nil,
		nil,
	)
	OSWriteChars := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OSWriteChars_total"),
		"Number of OSWriteChars total processed",
		nil,
		nil,
	)
	QueryProfilerSignalOverruns := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryProfilerSignalOverruns_total"),
		"Number of QueryProfilerSignalOverruns total processed",
		nil,
		nil,
	)
	QueryProfilerRuns := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryProfilerRuns_total"),
		"Number of QueryProfilerRuns total processed",
		nil,
		nil,
	)
	QueryMemoryLimitExceeded := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryMemoryLimitExceeded_total"),
		"Number of QueryMemoryLimitExceeded total processed",
		nil,
		nil,
	)
	ThreadPoolReaderPageCacheHitElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolReaderPageCacheHitElapsedMicroseconds_total"),
		"Number of ThreadPoolReaderPageCacheHitElapsedMicroseconds total processed",
		nil,
		nil,
	)
	ThreadPoolReaderPageCacheMiss := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolReaderPageCacheMiss_total"),
		"Number of ThreadPoolReaderPageCacheMiss total processed",
		nil,
		nil,
	)
	ThreadPoolReaderPageCacheMissBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolReaderPageCacheMissBytes_total"),
		"Number of ThreadPoolReaderPageCacheMissBytes total processed",
		nil,
		nil,
	)
	ThreadPoolReaderPageCacheMissElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolReaderPageCacheMissElapsedMicroseconds_total"),
		"Number of ThreadPoolReaderPageCacheMissElapsedMicroseconds total processed",
		nil,
		nil,
	)
	SynchronousReadWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SynchronousReadWaitMicroseconds_total"),
		"Number of SynchronousReadWaitMicroseconds total processed",
		nil,
		nil,
	)
	MainConfigLoads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MainConfigLoads_total"),
		"Number of MainConfigLoads total processed",
		nil,
		nil,
	)
	ServerStartupMilliseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ServerStartupMilliseconds_total"),
		"Number of ServerStartupMilliseconds total processed",
		nil,
		nil,
	)
	MergerMutatorsGetPartsForMergeElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorsGetPartsForMergeElapsedMicroseconds_total"),
		"Number of MergerMutatorsGetPartsForMergeElapsedMicroseconds total processed",
		nil,
		nil,
	)
	MergerMutatorPrepareRangesForMergeElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorPrepareRangesForMergeElapsedMicroseconds_total"),
		"Number of MergerMutatorPrepareRangesForMergeElapsedMicroseconds total processed",
		nil,
		nil,
	)
	MergerMutatorSelectPartsForMergeElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorSelectPartsForMergeElapsedMicroseconds_total"),
		"Number of MergerMutatorSelectPartsForMergeElapsedMicroseconds total processed",
		nil,
		nil,
	)
	MergerMutatorRangesForMergeCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorRangesForMergeCount_total"),
		"Number of MergerMutatorRangesForMergeCount total processed",
		nil,
		nil,
	)
	MergerMutatorPartsInRangesForMergeCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorPartsInRangesForMergeCount_total"),
		"Number of MergerMutatorPartsInRangesForMergeCount total processed",
		nil,
		nil,
	)
	MergerMutatorSelectRangePartsCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergerMutatorSelectRangePartsCount_total"),
		"Number of MergerMutatorSelectRangePartsCount total processed",
		nil,
		nil,
	)
	AsyncLoaderWaitMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsyncLoaderWaitMicroseconds_total"),
		"Number of AsyncLoaderWaitMicroseconds total processed",
		nil,
		nil,
	)
	LogTrace := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LogTrace_total"),
		"Number of LogTrace total processed",
		nil,
		nil,
	)
	LogDebug := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LogDebug_total"),
		"Number of LogDebug total processed",
		nil,
		nil,
	)
	LogInfo := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LogInfo_total"),
		"Number of LogInfo total processed",
		nil,
		nil,
	)
	LogWarning := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LogWarning_total"),
		"Number of LogWarning total processed",
		nil,
		nil,
	)
	LogError := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LogError_total"),
		"Number of LogError total processed",
		nil,
		nil,
	)
	LoggerElapsedNanoseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LoggerElapsedNanoseconds_total"),
		"Number of LoggerElapsedNanoseconds total processed",
		nil,
		nil,
	)
	InterfaceHTTPSendBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InterfaceHTTPSendBytes_total"),
		"Number of InterfaceHTTPSendBytes total processed",
		nil,
		nil,
	)
	InterfaceHTTPReceiveBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InterfaceHTTPReceiveBytes_total"),
		"Number of InterfaceHTTPReceiveBytes total processed",
		nil,
		nil,
	)
	InterfaceNativeSendBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InterfaceNativeSendBytes_total"),
		"Number of InterfaceNativeSendBytes total processed",
		nil,
		nil,
	)
	InterfaceNativeReceiveBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InterfaceNativeReceiveBytes_total"),
		"Number of InterfaceNativeReceiveBytes total processed",
		nil,
		nil,
	)
	ConcurrencyControlSlotsGranted := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrencyControlSlotsGranted_total"),
		"Number of ConcurrencyControlSlotsGranted total processed",
		nil,
		nil,
	)
	ConcurrencyControlSlotsAcquired := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrencyControlSlotsAcquired_total"),
		"Number of ConcurrencyControlSlotsAcquired total processed",
		nil,
		nil,
	)
	GWPAsanAllocateFailed := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GWPAsanAllocateFailed_total"),
		"Number of GWPAsanAllocateFailed total processed",
		nil,
		nil,
	)
	MemoryWorkerRun := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MemoryWorkerRun_total"),
		"Number of MemoryWorkerRun total processed",
		nil,
		nil,
	)
	MemoryWorkerRunElapsedMicroseconds := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MemoryWorkerRunElapsedMicroseconds_total"),
		"Number of MemoryWorkerRunElapsedMicroseconds total processed",
		nil,
		nil,
	)
	return &EventsURIExporter{
		Query:                                                 Query,
		SelectQuery:                                           SelectQuery,
		InitialQuery:                                          InitialQuery,
		QueriesWithSubqueries:                                 QueriesWithSubqueries,
		SelectQueriesWithSubqueries:                           SelectQueriesWithSubqueries,
		FailedQuery:                                           FailedQuery,
		QueryTimeMicroseconds:                                 QueryTimeMicroseconds,
		SelectQueryTimeMicroseconds:                           SelectQueryTimeMicroseconds,
		OtherQueryTimeMicroseconds:                            OtherQueryTimeMicroseconds,
		FileOpen:                                              FileOpen,
		Seek:                                                  Seek,
		ReadBufferFromFileDescriptorRead:                      ReadBufferFromFileDescriptorRead,
		ReadBufferFromFileDescriptorReadBytes:                 ReadBufferFromFileDescriptorReadBytes,
		WriteBufferFromFileDescriptorWrite:                    WriteBufferFromFileDescriptorWrite,
		WriteBufferFromFileDescriptorWriteBytes:               WriteBufferFromFileDescriptorWriteBytes,
		FileSync:                                              FileSync,
		FileSyncElapsedMicroseconds:                           FileSyncElapsedMicroseconds,
		ReadCompressedBytes:                                   ReadCompressedBytes,
		CompressedReadBufferBlocks:                            CompressedReadBufferBlocks,
		CompressedReadBufferBytes:                             CompressedReadBufferBytes,
		OpenedFileCacheHits:                                   OpenedFileCacheHits,
		OpenedFileCacheMisses:                                 OpenedFileCacheMisses,
		OpenedFileCacheMicroseconds:                           OpenedFileCacheMicroseconds,
		IOBufferAllocs:                                        IOBufferAllocs,
		IOBufferAllocBytes:                                    IOBufferAllocBytes,
		ArenaAllocChunks:                                      ArenaAllocChunks,
		ArenaAllocBytes:                                       ArenaAllocBytes,
		FunctionExecute:                                       FunctionExecute,
		TableFunctionExecute:                                  TableFunctionExecute,
		CreatedReadBufferOrdinary:                             CreatedReadBufferOrdinary,
		DiskReadElapsedMicroseconds:                           DiskReadElapsedMicroseconds,
		DiskWriteElapsedMicroseconds:                          DiskWriteElapsedMicroseconds,
		NetworkReceiveElapsedMicroseconds:                     NetworkReceiveElapsedMicroseconds,
		NetworkSendElapsedMicroseconds:                        NetworkSendElapsedMicroseconds,
		NetworkReceiveBytes:                                   NetworkReceiveBytes,
		NetworkSendBytes:                                      NetworkSendBytes,
		GlobalThreadPoolExpansions:                            GlobalThreadPoolExpansions,
		GlobalThreadPoolThreadCreationMicroseconds:            GlobalThreadPoolThreadCreationMicroseconds,
		GlobalThreadPoolLockWaitMicroseconds:                  GlobalThreadPoolLockWaitMicroseconds,
		GlobalThreadPoolJobs:                                  GlobalThreadPoolJobs,
		GlobalThreadPoolJobWaitTimeMicroseconds:               GlobalThreadPoolJobWaitTimeMicroseconds,
		LocalThreadPoolExpansions:                             LocalThreadPoolExpansions,
		LocalThreadPoolShrinks:                                LocalThreadPoolShrinks,
		LocalThreadPoolThreadCreationMicroseconds:             LocalThreadPoolThreadCreationMicroseconds,
		LocalThreadPoolLockWaitMicroseconds:                   LocalThreadPoolLockWaitMicroseconds,
		LocalThreadPoolJobs:                                   LocalThreadPoolJobs,
		LocalThreadPoolBusyMicroseconds:                       LocalThreadPoolBusyMicroseconds,
		LocalThreadPoolJobWaitTimeMicroseconds:                LocalThreadPoolJobWaitTimeMicroseconds,
		InsertedRows:                                          InsertedRows,
		InsertedBytes:                                         InsertedBytes,
		CompileFunction:                                       CompileFunction,
		CompileExpressionsMicroseconds:                        CompileExpressionsMicroseconds,
		CompileExpressionsBytes:                               CompileExpressionsBytes,
		ExternalProcessingFilesTotal:                          ExternalProcessingFilesTotal,
		JoinBuildTableRowCount:                                JoinBuildTableRowCount,
		JoinProbeTableRowCount:                                JoinProbeTableRowCount,
		JoinResultRowCount:                                    JoinResultRowCount,
		SelectedRows:                                          SelectedRows,
		SelectedBytes:                                         SelectedBytes,
		RowsReadByMainReader:                                  RowsReadByMainReader,
		LoadedDataParts:                                       LoadedDataParts,
		LoadedDataPartsMicroseconds:                           LoadedDataPartsMicroseconds,
		WaitMarksLoadMicroseconds:                             WaitMarksLoadMicroseconds,
		LoadedMarksFiles:                                      LoadedMarksFiles,
		LoadedMarksCount:                                      LoadedMarksCount,
		LoadedMarksMemoryBytes:                                LoadedMarksMemoryBytes,
		Merge:                                                 Merge,
		MergeSourceParts:                                      MergeSourceParts,
		MergedRows:                                            MergedRows,
		MergedColumns:                                         MergedColumns,
		GatheredColumns:                                       GatheredColumns,
		MergedUncompressedBytes:                               MergedUncompressedBytes,
		MergeTotalMilliseconds:                                MergeTotalMilliseconds,
		MergeExecuteMilliseconds:                              MergeExecuteMilliseconds,
		MergeHorizontalStageTotalMilliseconds:                 MergeHorizontalStageTotalMilliseconds,
		MergeHorizontalStageExecuteMilliseconds:               MergeHorizontalStageExecuteMilliseconds,
		MergeVerticalStageTotalMilliseconds:                   MergeVerticalStageTotalMilliseconds,
		MergeVerticalStageExecuteMilliseconds:                 MergeVerticalStageExecuteMilliseconds,
		MergeProjectionStageTotalMilliseconds:                 MergeProjectionStageTotalMilliseconds,
		MergeProjectionStageExecuteMilliseconds:               MergeProjectionStageExecuteMilliseconds,
		MergingSortedMilliseconds:                             MergingSortedMilliseconds,
		GatheringColumnMilliseconds:                           GatheringColumnMilliseconds,
		MergeTreeDataWriterRows:                               MergeTreeDataWriterRows,
		MergeTreeDataWriterUncompressedBytes:                  MergeTreeDataWriterUncompressedBytes,
		MergeTreeDataWriterCompressedBytes:                    MergeTreeDataWriterCompressedBytes,
		MergeTreeDataWriterBlocks:                             MergeTreeDataWriterBlocks,
		MergeTreeDataWriterBlocksAlreadySorted:                MergeTreeDataWriterBlocksAlreadySorted,
		MergeTreeDataWriterSortingBlocksMicroseconds:          MergeTreeDataWriterSortingBlocksMicroseconds,
		MergeTreeDataWriterMergingBlocksMicroseconds:          MergeTreeDataWriterMergingBlocksMicroseconds,
		InsertedWideParts:                                     InsertedWideParts,
		InsertedCompactParts:                                  InsertedCompactParts,
		MergedIntoWideParts:                                   MergedIntoWideParts,
		MergedIntoCompactParts:                                MergedIntoCompactParts,
		ContextLock:                                           ContextLock,
		ContextLockWaitMicroseconds:                           ContextLockWaitMicroseconds,
		SystemLogErrorOnFlush:                                 SystemLogErrorOnFlush,
		RWLockAcquiredReadLocks:                               RWLockAcquiredReadLocks,
		RWLockReadersWaitMilliseconds:                         RWLockReadersWaitMilliseconds,
		PartsLockHoldMicroseconds:                             PartsLockHoldMicroseconds,
		PartsLockWaitMicroseconds:                             PartsLockWaitMicroseconds,
		RealTimeMicroseconds:                                  RealTimeMicroseconds,
		UserTimeMicroseconds:                                  UserTimeMicroseconds,
		SystemTimeMicroseconds:                                SystemTimeMicroseconds,
		SoftPageFaults:                                        SoftPageFaults,
		HardPageFaults:                                        HardPageFaults,
		OSCPUWaitMicroseconds:                                 OSCPUWaitMicroseconds,
		OSCPUVirtualTimeMicroseconds:                          OSCPUVirtualTimeMicroseconds,
		OSReadBytes:                                           OSReadBytes,
		OSWriteBytes:                                          OSWriteBytes,
		OSReadChars:                                           OSReadChars,
		OSWriteChars:                                          OSWriteChars,
		QueryProfilerSignalOverruns:                           QueryProfilerSignalOverruns,
		QueryProfilerRuns:                                     QueryProfilerRuns,
		QueryMemoryLimitExceeded:                              QueryMemoryLimitExceeded,
		ThreadPoolReaderPageCacheHitElapsedMicroseconds:       ThreadPoolReaderPageCacheHitElapsedMicroseconds,
		ThreadPoolReaderPageCacheMiss:                         ThreadPoolReaderPageCacheMiss,
		ThreadPoolReaderPageCacheMissBytes:                    ThreadPoolReaderPageCacheMissBytes,
		ThreadPoolReaderPageCacheMissElapsedMicroseconds:      ThreadPoolReaderPageCacheMissElapsedMicroseconds,
		SynchronousReadWaitMicroseconds:                       SynchronousReadWaitMicroseconds,
		MainConfigLoads:                                       MainConfigLoads,
		ServerStartupMilliseconds:                             ServerStartupMilliseconds,
		MergerMutatorsGetPartsForMergeElapsedMicroseconds:     MergerMutatorsGetPartsForMergeElapsedMicroseconds,
		MergerMutatorPrepareRangesForMergeElapsedMicroseconds: MergerMutatorPrepareRangesForMergeElapsedMicroseconds,
		MergerMutatorSelectPartsForMergeElapsedMicroseconds:   MergerMutatorSelectPartsForMergeElapsedMicroseconds,
		MergerMutatorRangesForMergeCount:                      MergerMutatorRangesForMergeCount,
		MergerMutatorPartsInRangesForMergeCount:               MergerMutatorPartsInRangesForMergeCount,
		MergerMutatorSelectRangePartsCount:                    MergerMutatorSelectRangePartsCount,
		AsyncLoaderWaitMicroseconds:                           AsyncLoaderWaitMicroseconds,
		LogTrace:                                              LogTrace,
		LogDebug:                                              LogDebug,
		LogInfo:                                               LogInfo,
		LogWarning:                                            LogWarning,
		LogError:                                              LogError,
		LoggerElapsedNanoseconds:                              LoggerElapsedNanoseconds,
		InterfaceHTTPSendBytes:                                InterfaceHTTPSendBytes,
		InterfaceHTTPReceiveBytes:                             InterfaceHTTPReceiveBytes,
		InterfaceNativeSendBytes:                              InterfaceNativeSendBytes,
		InterfaceNativeReceiveBytes:                           InterfaceNativeReceiveBytes,
		ConcurrencyControlSlotsGranted:                        ConcurrencyControlSlotsGranted,
		ConcurrencyControlSlotsAcquired:                       ConcurrencyControlSlotsAcquired,
		GWPAsanAllocateFailed:                                 GWPAsanAllocateFailed,
		MemoryWorkerRun:                                       MemoryWorkerRun,
		MemoryWorkerRunElapsedMicroseconds:                    MemoryWorkerRunElapsedMicroseconds,
	}
}

func (e *EventsURIExporter) Describe(ch chan<- *prometheus.Desc) {

}

func (e *EventsURIExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	if err := e.collect(ch); err != nil {
		log.Info().Msgf("Error scraping clickhouse: %s", err)
	}

}

func (e *EventsURIExporter) collect(ch chan<- prometheus.Metric) error {
	mu, err := url.Parse(URI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	q := mu.Query()
	eventsURI := mu
	q.Set("query", "select event, value from system.events")
	eventsURI.RawQuery = q.Encode()

	metrics, err := e.parseEventsURIResponse(eventsURI.String())
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", eventsURI.String(), err)
	}
	ch <- prometheus.MustNewConstMetric(
		e.Query,
		prometheus.GaugeValue,
		float64(metrics.Query),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SelectQuery,
		prometheus.GaugeValue,
		float64(metrics.SelectQuery),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InitialQuery,
		prometheus.GaugeValue,
		float64(metrics.InitialQuery),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueriesWithSubqueries,
		prometheus.GaugeValue,
		float64(metrics.QueriesWithSubqueries),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SelectQueriesWithSubqueries,
		prometheus.GaugeValue,
		float64(metrics.SelectQueriesWithSubqueries),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FailedQuery,
		prometheus.GaugeValue,
		float64(metrics.FailedQuery),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.QueryTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SelectQueryTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.SelectQueryTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OtherQueryTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.OtherQueryTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FileOpen,
		prometheus.GaugeValue,
		float64(metrics.FileOpen),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Seek,
		prometheus.GaugeValue,
		float64(metrics.Seek),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReadBufferFromFileDescriptorRead,
		prometheus.GaugeValue,
		float64(metrics.ReadBufferFromFileDescriptorRead),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReadBufferFromFileDescriptorReadBytes,
		prometheus.GaugeValue,
		float64(metrics.ReadBufferFromFileDescriptorReadBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.WriteBufferFromFileDescriptorWrite,
		prometheus.GaugeValue,
		float64(metrics.WriteBufferFromFileDescriptorWrite),
	)
	ch <- prometheus.MustNewConstMetric(
		e.WriteBufferFromFileDescriptorWriteBytes,
		prometheus.GaugeValue,
		float64(metrics.WriteBufferFromFileDescriptorWriteBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FileSync,
		prometheus.GaugeValue,
		float64(metrics.FileSync),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FileSyncElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.FileSyncElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReadCompressedBytes,
		prometheus.GaugeValue,
		float64(metrics.ReadCompressedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompressedReadBufferBlocks,
		prometheus.GaugeValue,
		float64(metrics.CompressedReadBufferBlocks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompressedReadBufferBytes,
		prometheus.GaugeValue,
		float64(metrics.CompressedReadBufferBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OpenedFileCacheHits,
		prometheus.GaugeValue,
		float64(metrics.OpenedFileCacheHits),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OpenedFileCacheMisses,
		prometheus.GaugeValue,
		float64(metrics.OpenedFileCacheMisses),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OpenedFileCacheMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.OpenedFileCacheMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOBufferAllocs,
		prometheus.GaugeValue,
		float64(metrics.IOBufferAllocs),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOBufferAllocBytes,
		prometheus.GaugeValue,
		float64(metrics.IOBufferAllocBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ArenaAllocChunks,
		prometheus.GaugeValue,
		float64(metrics.ArenaAllocChunks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ArenaAllocBytes,
		prometheus.GaugeValue,
		float64(metrics.ArenaAllocBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FunctionExecute,
		prometheus.GaugeValue,
		float64(metrics.FunctionExecute),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TableFunctionExecute,
		prometheus.GaugeValue,
		float64(metrics.TableFunctionExecute),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CreatedReadBufferOrdinary,
		prometheus.GaugeValue,
		float64(metrics.CreatedReadBufferOrdinary),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskReadElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.DiskReadElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskWriteElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.DiskWriteElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkReceiveElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.NetworkReceiveElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkSendElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.NetworkSendElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkReceiveBytes,
		prometheus.GaugeValue,
		float64(metrics.NetworkReceiveBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkSendBytes,
		prometheus.GaugeValue,
		float64(metrics.NetworkSendBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadPoolExpansions,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadPoolExpansions),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadPoolThreadCreationMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadPoolThreadCreationMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadPoolLockWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadPoolLockWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadPoolJobs,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadPoolJobs),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadPoolJobWaitTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadPoolJobWaitTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolExpansions,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolExpansions),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolShrinks,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolShrinks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolThreadCreationMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolThreadCreationMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolLockWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolLockWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolJobs,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolJobs),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolBusyMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolBusyMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadPoolJobWaitTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadPoolJobWaitTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InsertedRows,
		prometheus.GaugeValue,
		float64(metrics.InsertedRows),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InsertedBytes,
		prometheus.GaugeValue,
		float64(metrics.InsertedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompileFunction,
		prometheus.GaugeValue,
		float64(metrics.CompileFunction),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompileExpressionsMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.CompileExpressionsMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompileExpressionsBytes,
		prometheus.GaugeValue,
		float64(metrics.CompileExpressionsBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ExternalProcessingFilesTotal,
		prometheus.GaugeValue,
		float64(metrics.ExternalProcessingFilesTotal),
	)
	ch <- prometheus.MustNewConstMetric(
		e.JoinBuildTableRowCount,
		prometheus.GaugeValue,
		float64(metrics.JoinBuildTableRowCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.JoinProbeTableRowCount,
		prometheus.GaugeValue,
		float64(metrics.JoinProbeTableRowCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.JoinResultRowCount,
		prometheus.GaugeValue,
		float64(metrics.JoinResultRowCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SelectedRows,
		prometheus.GaugeValue,
		float64(metrics.SelectedRows),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SelectedBytes,
		prometheus.GaugeValue,
		float64(metrics.SelectedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RowsReadByMainReader,
		prometheus.GaugeValue,
		float64(metrics.RowsReadByMainReader),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoadedDataParts,
		prometheus.GaugeValue,
		float64(metrics.LoadedDataParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoadedDataPartsMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.LoadedDataPartsMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.WaitMarksLoadMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.WaitMarksLoadMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoadedMarksFiles,
		prometheus.GaugeValue,
		float64(metrics.LoadedMarksFiles),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoadedMarksCount,
		prometheus.GaugeValue,
		float64(metrics.LoadedMarksCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoadedMarksMemoryBytes,
		prometheus.GaugeValue,
		float64(metrics.LoadedMarksMemoryBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Merge,
		prometheus.GaugeValue,
		float64(metrics.Merge),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeSourceParts,
		prometheus.GaugeValue,
		float64(metrics.MergeSourceParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergedRows,
		prometheus.GaugeValue,
		float64(metrics.MergedRows),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergedColumns,
		prometheus.GaugeValue,
		float64(metrics.MergedColumns),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GatheredColumns,
		prometheus.GaugeValue,
		float64(metrics.GatheredColumns),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergedUncompressedBytes,
		prometheus.GaugeValue,
		float64(metrics.MergedUncompressedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTotalMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeTotalMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeExecuteMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeExecuteMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeHorizontalStageTotalMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeHorizontalStageTotalMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeHorizontalStageExecuteMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeHorizontalStageExecuteMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeVerticalStageTotalMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeVerticalStageTotalMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeVerticalStageExecuteMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeVerticalStageExecuteMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeProjectionStageTotalMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeProjectionStageTotalMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeProjectionStageExecuteMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeProjectionStageExecuteMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergingSortedMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.MergingSortedMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GatheringColumnMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.GatheringColumnMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterRows,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterRows),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterUncompressedBytes,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterUncompressedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterCompressedBytes,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterCompressedBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterBlocks,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterBlocks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterBlocksAlreadySorted,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterBlocksAlreadySorted),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterSortingBlocksMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterSortingBlocksMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataWriterMergingBlocksMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataWriterMergingBlocksMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InsertedWideParts,
		prometheus.GaugeValue,
		float64(metrics.InsertedWideParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InsertedCompactParts,
		prometheus.GaugeValue,
		float64(metrics.InsertedCompactParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergedIntoWideParts,
		prometheus.GaugeValue,
		float64(metrics.MergedIntoWideParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergedIntoCompactParts,
		prometheus.GaugeValue,
		float64(metrics.MergedIntoCompactParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ContextLock,
		prometheus.GaugeValue,
		float64(metrics.ContextLock),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ContextLockWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.ContextLockWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SystemLogErrorOnFlush,
		prometheus.GaugeValue,
		float64(metrics.SystemLogErrorOnFlush),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockAcquiredReadLocks,
		prometheus.GaugeValue,
		float64(metrics.RWLockAcquiredReadLocks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockReadersWaitMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.RWLockReadersWaitMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsLockHoldMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.PartsLockHoldMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsLockWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.PartsLockWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RealTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.RealTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.UserTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.UserTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SystemTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.SystemTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SoftPageFaults,
		prometheus.GaugeValue,
		float64(metrics.SoftPageFaults),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HardPageFaults,
		prometheus.GaugeValue,
		float64(metrics.HardPageFaults),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSCPUWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.OSCPUWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSCPUVirtualTimeMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.OSCPUVirtualTimeMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSReadBytes,
		prometheus.GaugeValue,
		float64(metrics.OSReadBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSWriteBytes,
		prometheus.GaugeValue,
		float64(metrics.OSWriteBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSReadChars,
		prometheus.GaugeValue,
		float64(metrics.OSReadChars),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OSWriteChars,
		prometheus.GaugeValue,
		float64(metrics.OSWriteChars),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryProfilerSignalOverruns,
		prometheus.GaugeValue,
		float64(metrics.QueryProfilerSignalOverruns),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryProfilerRuns,
		prometheus.GaugeValue,
		float64(metrics.QueryProfilerRuns),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryMemoryLimitExceeded,
		prometheus.GaugeValue,
		float64(metrics.QueryMemoryLimitExceeded),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolReaderPageCacheHitElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolReaderPageCacheHitElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolReaderPageCacheMiss,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolReaderPageCacheMiss),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolReaderPageCacheMissBytes,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolReaderPageCacheMissBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolReaderPageCacheMissElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolReaderPageCacheMissElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SynchronousReadWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.SynchronousReadWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MainConfigLoads,
		prometheus.GaugeValue,
		float64(metrics.MainConfigLoads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ServerStartupMilliseconds,
		prometheus.GaugeValue,
		float64(metrics.ServerStartupMilliseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorsGetPartsForMergeElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorsGetPartsForMergeElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorPrepareRangesForMergeElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorPrepareRangesForMergeElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorSelectPartsForMergeElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorSelectPartsForMergeElapsedMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorRangesForMergeCount,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorRangesForMergeCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorPartsInRangesForMergeCount,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorPartsInRangesForMergeCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergerMutatorSelectRangePartsCount,
		prometheus.GaugeValue,
		float64(metrics.MergerMutatorSelectRangePartsCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsyncLoaderWaitMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.AsyncLoaderWaitMicroseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LogTrace,
		prometheus.GaugeValue,
		float64(metrics.LogTrace),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LogDebug,
		prometheus.GaugeValue,
		float64(metrics.LogDebug),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LogInfo,
		prometheus.GaugeValue,
		float64(metrics.LogInfo),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LogWarning,
		prometheus.GaugeValue,
		float64(metrics.LogWarning),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LogError,
		prometheus.GaugeValue,
		float64(metrics.LogError),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LoggerElapsedNanoseconds,
		prometheus.GaugeValue,
		float64(metrics.LoggerElapsedNanoseconds),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InterfaceHTTPSendBytes,
		prometheus.GaugeValue,
		float64(metrics.InterfaceHTTPSendBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InterfaceHTTPReceiveBytes,
		prometheus.GaugeValue,
		float64(metrics.InterfaceHTTPReceiveBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InterfaceNativeSendBytes,
		prometheus.GaugeValue,
		float64(metrics.InterfaceNativeSendBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InterfaceNativeReceiveBytes,
		prometheus.GaugeValue,
		float64(metrics.InterfaceNativeReceiveBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrencyControlSlotsGranted,
		prometheus.GaugeValue,
		float64(metrics.ConcurrencyControlSlotsGranted),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrencyControlSlotsAcquired,
		prometheus.GaugeValue,
		float64(metrics.ConcurrencyControlSlotsAcquired),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GWPAsanAllocateFailed,
		prometheus.GaugeValue,
		float64(metrics.GWPAsanAllocateFailed),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MemoryWorkerRun,
		prometheus.GaugeValue,
		float64(metrics.MemoryWorkerRun),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MemoryWorkerRunElapsedMicroseconds,
		prometheus.GaugeValue,
		float64(metrics.MemoryWorkerRunElapsedMicroseconds),
	)
	return nil
}

func (e *EventsURIExporter) parseEventsURIResponse(uri string) (*EventsUri, error) {
	data, err := handleResponse(uri)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)

	// 按行拆分字符串
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 2 {
			return nil, fmt.Errorf("parseKeyValueResponse: unexpected %d line: %s", i, line)
		}
		k := strings.TrimSpace(parts[0])
		k = metricName(k)
		v, err := parseNumber(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		result[k] = v // 存储到映射中
	}

	// 将映射编码为 JSON 格式
	jsonData, err := json.Marshal(result)
	if err != nil {
		fmt.Println("Error encoding to JSON:", err)
		return nil, err
	}

	var event EventsUri
	err = json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	return &event, nil
}
