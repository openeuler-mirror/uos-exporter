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
type MetricsURIExporter struct {
	Query                                             *prometheus.Desc
	Merge                                             *prometheus.Desc
	MergeParts                                        *prometheus.Desc
	Move                                              *prometheus.Desc
	PartMutation                                      *prometheus.Desc
	ReplicatedFetch                                   *prometheus.Desc
	ReplicatedSend                                    *prometheus.Desc
	ReplicatedChecks                                  *prometheus.Desc
	BackgroundMergesAndMutationsPoolTask              *prometheus.Desc
	BackgroundMergesAndMutationsPoolSize              *prometheus.Desc
	BackgroundFetchesPoolTask                         *prometheus.Desc
	BackgroundFetchesPoolSize                         *prometheus.Desc
	BackgroundCommonPoolTask                          *prometheus.Desc
	BackgroundCommonPoolSize                          *prometheus.Desc
	BackgroundMovePoolTask                            *prometheus.Desc
	BackgroundMovePoolSize                            *prometheus.Desc
	BackgroundSchedulePoolTask                        *prometheus.Desc
	BackgroundSchedulePoolSize                        *prometheus.Desc
	BackgroundBufferFlushSchedulePoolTask             *prometheus.Desc
	BackgroundBufferFlushSchedulePoolSize             *prometheus.Desc
	BackgroundDistributedSchedulePoolTask             *prometheus.Desc
	BackgroundDistributedSchedulePoolSize             *prometheus.Desc
	BackgroundMessageBrokerSchedulePoolTask           *prometheus.Desc
	BackgroundMessageBrokerSchedulePoolSize           *prometheus.Desc
	CacheDictionaryUpdateQueueBatches                 *prometheus.Desc
	CacheDictionaryUpdateQueueKeys                    *prometheus.Desc
	DiskSpaceReservedForMerge                         *prometheus.Desc
	DistributedSend                                   *prometheus.Desc
	QueryPreempted                                    *prometheus.Desc
	TCPConnection                                     *prometheus.Desc
	MySQLConnection                                   *prometheus.Desc
	HTTPConnection                                    *prometheus.Desc
	InterserverConnection                             *prometheus.Desc
	PostgreSQLConnection                              *prometheus.Desc
	OpenFileForRead                                   *prometheus.Desc
	OpenFileForWrite                                  *prometheus.Desc
	Compressing                                       *prometheus.Desc
	Decompressing                                     *prometheus.Desc
	ParallelCompressedWriteBufferThreads              *prometheus.Desc
	ParallelCompressedWriteBufferWait                 *prometheus.Desc
	TotalTemporaryFiles                               *prometheus.Desc
	TemporaryFilesForSort                             *prometheus.Desc
	TemporaryFilesForAggregation                      *prometheus.Desc
	TemporaryFilesForJoin                             *prometheus.Desc
	TemporaryFilesForMerge                            *prometheus.Desc
	TemporaryFilesUnknown                             *prometheus.Desc
	Read                                              *prometheus.Desc
	RemoteRead                                        *prometheus.Desc
	Write                                             *prometheus.Desc
	NetworkReceive                                    *prometheus.Desc
	NetworkSend                                       *prometheus.Desc
	SendScalars                                       *prometheus.Desc
	SendExternalTables                                *prometheus.Desc
	QueryThread                                       *prometheus.Desc
	ReadonlyReplica                                   *prometheus.Desc
	MemoryTracking                                    *prometheus.Desc
	MemoryTrackingUncorrected                         *prometheus.Desc
	MergesMutationsMemoryTracking                     *prometheus.Desc
	EphemeralNode                                     *prometheus.Desc
	ZooKeeperSession                                  *prometheus.Desc
	ZooKeeperWatch                                    *prometheus.Desc
	ZooKeeperRequest                                  *prometheus.Desc
	DelayedInserts                                    *prometheus.Desc
	ContextLockWait                                   *prometheus.Desc
	StorageBufferRows                                 *prometheus.Desc
	StorageBufferBytes                                *prometheus.Desc
	DictCacheRequests                                 *prometheus.Desc
	Revision                                          *prometheus.Desc
	VersionInteger                                    *prometheus.Desc
	RWLockWaitingReaders                              *prometheus.Desc
	RWLockWaitingWriters                              *prometheus.Desc
	RWLockActiveReaders                               *prometheus.Desc
	RWLockActiveWriters                               *prometheus.Desc
	GlobalThread                                      *prometheus.Desc
	GlobalThreadActive                                *prometheus.Desc
	GlobalThreadScheduled                             *prometheus.Desc
	LocalThread                                       *prometheus.Desc
	LocalThreadActive                                 *prometheus.Desc
	LocalThreadScheduled                              *prometheus.Desc
	MergeTreeDataSelectExecutorThreads                *prometheus.Desc
	MergeTreeDataSelectExecutorThreadsActive          *prometheus.Desc
	MergeTreeDataSelectExecutorThreadsScheduled       *prometheus.Desc
	BackupsThreads                                    *prometheus.Desc
	BackupsThreadsActive                              *prometheus.Desc
	BackupsThreadsScheduled                           *prometheus.Desc
	RestoreThreads                                    *prometheus.Desc
	RestoreThreadsActive                              *prometheus.Desc
	RestoreThreadsScheduled                           *prometheus.Desc
	MarksLoaderThreads                                *prometheus.Desc
	MarksLoaderThreadsActive                          *prometheus.Desc
	MarksLoaderThreadsScheduled                       *prometheus.Desc
	IOPrefetchThreads                                 *prometheus.Desc
	IOPrefetchThreadsActive                           *prometheus.Desc
	IOPrefetchThreadsScheduled                        *prometheus.Desc
	IOWriterThreads                                   *prometheus.Desc
	IOWriterThreadsActive                             *prometheus.Desc
	IOWriterThreadsScheduled                          *prometheus.Desc
	IOThreads                                         *prometheus.Desc
	IOThreadsActive                                   *prometheus.Desc
	IOThreadsScheduled                                *prometheus.Desc
	CompressionThread                                 *prometheus.Desc
	CompressionThreadActive                           *prometheus.Desc
	CompressionThreadScheduled                        *prometheus.Desc
	ThreadPoolRemoteFSReaderThreads                   *prometheus.Desc
	ThreadPoolRemoteFSReaderThreadsActive             *prometheus.Desc
	ThreadPoolRemoteFSReaderThreadsScheduled          *prometheus.Desc
	ThreadPoolFSReaderThreads                         *prometheus.Desc
	ThreadPoolFSReaderThreadsActive                   *prometheus.Desc
	ThreadPoolFSReaderThreadsScheduled                *prometheus.Desc
	BackupsIOThreads                                  *prometheus.Desc
	BackupsIOThreadsActive                            *prometheus.Desc
	BackupsIOThreadsScheduled                         *prometheus.Desc
	DiskObjectStorageAsyncThreads                     *prometheus.Desc
	DiskObjectStorageAsyncThreadsActive               *prometheus.Desc
	StorageHiveThreads                                *prometheus.Desc
	StorageHiveThreadsActive                          *prometheus.Desc
	StorageHiveThreadsScheduled                       *prometheus.Desc
	TablesLoaderBackgroundThreads                     *prometheus.Desc
	TablesLoaderBackgroundThreadsActive               *prometheus.Desc
	TablesLoaderBackgroundThreadsScheduled            *prometheus.Desc
	TablesLoaderForegroundThreads                     *prometheus.Desc
	TablesLoaderForegroundThreadsActive               *prometheus.Desc
	TablesLoaderForegroundThreadsScheduled            *prometheus.Desc
	DatabaseOnDiskThreads                             *prometheus.Desc
	DatabaseOnDiskThreadsActive                       *prometheus.Desc
	DatabaseOnDiskThreadsScheduled                    *prometheus.Desc
	DatabaseBackupThreads                             *prometheus.Desc
	DatabaseBackupThreadsActive                       *prometheus.Desc
	DatabaseBackupThreadsScheduled                    *prometheus.Desc
	DatabaseCatalogThreads                            *prometheus.Desc
	DatabaseCatalogThreadsActive                      *prometheus.Desc
	DatabaseCatalogThreadsScheduled                   *prometheus.Desc
	DestroyAggregatesThreads                          *prometheus.Desc
	DestroyAggregatesThreadsActive                    *prometheus.Desc
	DestroyAggregatesThreadsScheduled                 *prometheus.Desc
	ConcurrentHashJoinPoolThreads                     *prometheus.Desc
	ConcurrentHashJoinPoolThreadsActive               *prometheus.Desc
	ConcurrentHashJoinPoolThreadsScheduled            *prometheus.Desc
	HashedDictionaryThreads                           *prometheus.Desc
	HashedDictionaryThreadsActive                     *prometheus.Desc
	HashedDictionaryThreadsScheduled                  *prometheus.Desc
	CacheDictionaryThreads                            *prometheus.Desc
	CacheDictionaryThreadsActive                      *prometheus.Desc
	CacheDictionaryThreadsScheduled                   *prometheus.Desc
	ParallelFormattingOutputFormatThreads             *prometheus.Desc
	ParallelFormattingOutputFormatThreadsActive       *prometheus.Desc
	ParallelFormattingOutputFormatThreadsScheduled    *prometheus.Desc
	ParallelParsingInputFormatThreads                 *prometheus.Desc
	ParallelParsingInputFormatThreadsActive           *prometheus.Desc
	ParallelParsingInputFormatThreadsScheduled        *prometheus.Desc
	MergeTreeBackgroundExecutorThreads                *prometheus.Desc
	MergeTreeBackgroundExecutorThreadsActive          *prometheus.Desc
	MergeTreeBackgroundExecutorThreadsScheduled       *prometheus.Desc
	AsynchronousInsertThreads                         *prometheus.Desc
	AsynchronousInsertThreadsActive                   *prometheus.Desc
	AsynchronousInsertThreadsScheduled                *prometheus.Desc
	AsynchronousInsertQueueSize                       *prometheus.Desc
	AsynchronousInsertQueueBytes                      *prometheus.Desc
	StartupSystemTablesThreads                        *prometheus.Desc
	StartupSystemTablesThreadsActive                  *prometheus.Desc
	StartupSystemTablesThreadsScheduled               *prometheus.Desc
	AggregatorThreads                                 *prometheus.Desc
	AggregatorThreadsActive                           *prometheus.Desc
	AggregatorThreadsScheduled                        *prometheus.Desc
	DDLWorkerThreads                                  *prometheus.Desc
	DDLWorkerThreadsActive                            *prometheus.Desc
	DDLWorkerThreadsScheduled                         *prometheus.Desc
	StorageDistributedThreads                         *prometheus.Desc
	StorageDistributedThreadsActive                   *prometheus.Desc
	StorageDistributedThreadsScheduled                *prometheus.Desc
	DistributedInsertThreads                          *prometheus.Desc
	DistributedInsertThreadsActive                    *prometheus.Desc
	DistributedInsertThreadsScheduled                 *prometheus.Desc
	StorageS3Threads                                  *prometheus.Desc
	StorageS3ThreadsActive                            *prometheus.Desc
	StorageS3ThreadsScheduled                         *prometheus.Desc
	ObjectStorageS3Threads                            *prometheus.Desc
	ObjectStorageS3ThreadsActive                      *prometheus.Desc
	ObjectStorageS3ThreadsScheduled                   *prometheus.Desc
	StorageObjectStorageThreads                       *prometheus.Desc
	StorageObjectStorageThreadsActive                 *prometheus.Desc
	StorageObjectStorageThreadsScheduled              *prometheus.Desc
	ObjectStorageAzureThreads                         *prometheus.Desc
	ObjectStorageAzureThreadsActive                   *prometheus.Desc
	ObjectStorageAzureThreadsScheduled                *prometheus.Desc
	BuildVectorSimilarityIndexThreads                 *prometheus.Desc
	BuildVectorSimilarityIndexThreadsActive           *prometheus.Desc
	BuildVectorSimilarityIndexThreadsScheduled        *prometheus.Desc
	ObjectStorageQueueRegisteredServers               *prometheus.Desc
	IcebergCatalogThreads                             *prometheus.Desc
	IcebergCatalogThreadsActive                       *prometheus.Desc
	IcebergCatalogThreadsScheduled                    *prometheus.Desc
	ParallelWithQueryThreads                          *prometheus.Desc
	ParallelWithQueryActiveThreads                    *prometheus.Desc
	ParallelWithQueryScheduledThreads                 *prometheus.Desc
	DiskPlainRewritableAzureDirectoryMapSize          *prometheus.Desc
	DiskPlainRewritableAzureFileCount                 *prometheus.Desc
	DiskPlainRewritableAzureUniqueFileNamesCount      *prometheus.Desc
	DiskPlainRewritableLocalDirectoryMapSize          *prometheus.Desc
	DiskPlainRewritableLocalFileCount                 *prometheus.Desc
	DiskPlainRewritableLocalUniqueFileNamesCount      *prometheus.Desc
	DiskPlainRewritableS3DirectoryMapSize             *prometheus.Desc
	DiskPlainRewritableS3FileCount                    *prometheus.Desc
	DiskPlainRewritableS3UniqueFileNamesCount         *prometheus.Desc
	MergeTreeFetchPartitionThreads                    *prometheus.Desc
	MergeTreeFetchPartitionThreadsActive              *prometheus.Desc
	MergeTreeFetchPartitionThreadsScheduled           *prometheus.Desc
	MergeTreePartsLoaderThreads                       *prometheus.Desc
	MergeTreePartsLoaderThreadsActive                 *prometheus.Desc
	MergeTreePartsLoaderThreadsScheduled              *prometheus.Desc
	MergeTreeOutdatedPartsLoaderThreads               *prometheus.Desc
	MergeTreeOutdatedPartsLoaderThreadsActive         *prometheus.Desc
	MergeTreeOutdatedPartsLoaderThreadsScheduled      *prometheus.Desc
	MergeTreeUnexpectedPartsLoaderThreads             *prometheus.Desc
	MergeTreeUnexpectedPartsLoaderThreadsActive       *prometheus.Desc
	MergeTreeUnexpectedPartsLoaderThreadsScheduled    *prometheus.Desc
	MergeTreePartsCleanerThreads                      *prometheus.Desc
	MergeTreePartsCleanerThreadsActive                *prometheus.Desc
	MergeTreePartsCleanerThreadsScheduled             *prometheus.Desc
	DatabaseReplicatedCreateTablesThreads             *prometheus.Desc
	DatabaseReplicatedCreateTablesThreadsActive       *prometheus.Desc
	DatabaseReplicatedCreateTablesThreadsScheduled    *prometheus.Desc
	IDiskCopierThreads                                *prometheus.Desc
	IDiskCopierThreadsActive                          *prometheus.Desc
	IDiskCopierThreadsScheduled                       *prometheus.Desc
	SystemReplicasThreads                             *prometheus.Desc
	SystemReplicasThreadsActive                       *prometheus.Desc
	SystemReplicasThreadsScheduled                    *prometheus.Desc
	RestartReplicaThreads                             *prometheus.Desc
	RestartReplicaThreadsActive                       *prometheus.Desc
	RestartReplicaThreadsScheduled                    *prometheus.Desc
	QueryPipelineExecutorThreads                      *prometheus.Desc
	QueryPipelineExecutorThreadsActive                *prometheus.Desc
	QueryPipelineExecutorThreadsScheduled             *prometheus.Desc
	ParquetDecoderThreads                             *prometheus.Desc
	ParquetDecoderThreadsActive                       *prometheus.Desc
	ParquetDecoderThreadsScheduled                    *prometheus.Desc
	ParquetDecoderIOThreads                           *prometheus.Desc
	ParquetDecoderIOThreadsActive                     *prometheus.Desc
	ParquetDecoderIOThreadsScheduled                  *prometheus.Desc
	ParquetEncoderThreads                             *prometheus.Desc
	ParquetEncoderThreadsActive                       *prometheus.Desc
	ParquetEncoderThreadsScheduled                    *prometheus.Desc
	MergeTreeSubcolumnsReaderThreads                  *prometheus.Desc
	MergeTreeSubcolumnsReaderThreadsActive            *prometheus.Desc
	MergeTreeSubcolumnsReaderThreadsScheduled         *prometheus.Desc
	DWARFReaderThreads                                *prometheus.Desc
	DWARFReaderThreadsActive                          *prometheus.Desc
	DWARFReaderThreadsScheduled                       *prometheus.Desc
	OutdatedPartsLoadingThreads                       *prometheus.Desc
	OutdatedPartsLoadingThreadsActive                 *prometheus.Desc
	OutdatedPartsLoadingThreadsScheduled              *prometheus.Desc
	PolygonDictionaryThreads                          *prometheus.Desc
	PolygonDictionaryThreadsActive                    *prometheus.Desc
	PolygonDictionaryThreadsScheduled                 *prometheus.Desc
	DistributedBytesToInsert                          *prometheus.Desc
	BrokenDistributedBytesToInsert                    *prometheus.Desc
	DistributedFilesToInsert                          *prometheus.Desc
	BrokenDistributedFilesToInsert                    *prometheus.Desc
	TablesToDropQueueSize                             *prometheus.Desc
	MaxDDLEntryID                                     *prometheus.Desc
	MaxPushedDDLEntryID                               *prometheus.Desc
	PartsTemporary                                    *prometheus.Desc
	PartsPreCommitted                                 *prometheus.Desc
	PartsCommitted                                    *prometheus.Desc
	PartsPreActive                                    *prometheus.Desc
	PartsActive                                       *prometheus.Desc
	AttachedDatabase                                  *prometheus.Desc
	AttachedTable                                     *prometheus.Desc
	AttachedReplicatedTable                           *prometheus.Desc
	AttachedView                                      *prometheus.Desc
	AttachedDictionary                                *prometheus.Desc
	PartsOutdated                                     *prometheus.Desc
	PartsDeleting                                     *prometheus.Desc
	PartsDeleteOnDestroy                              *prometheus.Desc
	PartsWide                                         *prometheus.Desc
	PartsCompact                                      *prometheus.Desc
	MMappedFiles                                      *prometheus.Desc
	MMappedFileBytes                                  *prometheus.Desc
	AsynchronousReadWait                              *prometheus.Desc
	PendingAsyncInsert                                *prometheus.Desc
	KafkaConsumers                                    *prometheus.Desc
	KafkaConsumersWithAssignment                      *prometheus.Desc
	KafkaProducers                                    *prometheus.Desc
	KafkaLibrdkafkaThreads                            *prometheus.Desc
	KafkaBackgroundReads                              *prometheus.Desc
	KafkaConsumersInUse                               *prometheus.Desc
	KafkaWrites                                       *prometheus.Desc
	KafkaAssignedPartitions                           *prometheus.Desc
	FilesystemCacheReadBuffers                        *prometheus.Desc
	CacheFileSegments                                 *prometheus.Desc
	CacheDetachedFileSegments                         *prometheus.Desc
	FilesystemCacheSize                               *prometheus.Desc
	FilesystemCacheSizeLimit                          *prometheus.Desc
	FilesystemCacheElements                           *prometheus.Desc
	FilesystemCacheDownloadQueueElements              *prometheus.Desc
	FilesystemCacheDelayedCleanupElements             *prometheus.Desc
	FilesystemCacheHoldFileSegments                   *prometheus.Desc
	AsyncInsertCacheSize                              *prometheus.Desc
	SkippingIndexCacheSize                            *prometheus.Desc
	S3Requests                                        *prometheus.Desc
	KeeperAliveConnections                            *prometheus.Desc
	KeeperOutstandingRequests                         *prometheus.Desc
	ThreadsInOvercommitTracker                        *prometheus.Desc
	IOUringPendingEvents                              *prometheus.Desc
	IOUringInFlightEvents                             *prometheus.Desc
	ReadTaskRequestsSent                              *prometheus.Desc
	MergeTreeReadTaskRequestsSent                     *prometheus.Desc
	MergeTreeAllRangesAnnouncementsSent               *prometheus.Desc
	CreatedTimersInQueryProfiler                      *prometheus.Desc
	ActiveTimersInQueryProfiler                       *prometheus.Desc
	RefreshableViews                                  *prometheus.Desc
	RefreshingViews                                   *prometheus.Desc
	StorageBufferFlushThreads                         *prometheus.Desc
	StorageBufferFlushThreadsActive                   *prometheus.Desc
	StorageBufferFlushThreadsScheduled                *prometheus.Desc
	SharedMergeTreeThreads                            *prometheus.Desc
	SharedMergeTreeThreadsActive                      *prometheus.Desc
	SharedMergeTreeThreadsScheduled                   *prometheus.Desc
	SharedMergeTreeFetch                              *prometheus.Desc
	CacheWarmerBytesInProgress                        *prometheus.Desc
	DistrCacheOpenedConnections                       *prometheus.Desc
	DistrCacheUsedConnections                         *prometheus.Desc
	DistrCacheAllocatedConnections                    *prometheus.Desc
	DistrCacheBorrowedConnections                     *prometheus.Desc
	DistrCacheReadRequests                            *prometheus.Desc
	DistrCacheWriteRequests                           *prometheus.Desc
	DistrCacheServerConnections                       *prometheus.Desc
	DistrCacheRegisteredServers                       *prometheus.Desc
	DistrCacheRegisteredServersCurrentAZ              *prometheus.Desc
	DistrCacheServerS3CachedClients                   *prometheus.Desc
	SchedulerIOReadScheduled                          *prometheus.Desc
	SchedulerIOWriteScheduled                         *prometheus.Desc
	StorageConnectionsStored                          *prometheus.Desc
	StorageConnectionsTotal                           *prometheus.Desc
	DiskConnectionsStored                             *prometheus.Desc
	DiskConnectionsTotal                              *prometheus.Desc
	HTTPConnectionsStored                             *prometheus.Desc
	HTTPConnectionsTotal                              *prometheus.Desc
	AddressesActive                                   *prometheus.Desc
	AddressesBanned                                   *prometheus.Desc
	FilteringMarksWithPrimaryKey                      *prometheus.Desc
	FilteringMarksWithSecondaryKeys                   *prometheus.Desc
	ConcurrencyControlAcquired                        *prometheus.Desc
	ConcurrencyControlSoftLimit                       *prometheus.Desc
	DiskS3NoSuchKeyErrors                             *prometheus.Desc
	SharedCatalogStateApplicationThreads              *prometheus.Desc
	SharedCatalogStateApplicationThreadsActive        *prometheus.Desc
	SharedCatalogStateApplicationThreadsScheduled     *prometheus.Desc
	SharedCatalogDropLocalThreads                     *prometheus.Desc
	SharedCatalogDropLocalThreadsActive               *prometheus.Desc
	SharedCatalogDropLocalThreadsScheduled            *prometheus.Desc
	SharedCatalogDropZooKeeperThreads                 *prometheus.Desc
	SharedCatalogDropZooKeeperThreadsActive           *prometheus.Desc
	SharedCatalogDropZooKeeperThreadsScheduled        *prometheus.Desc
	SharedDatabaseCatalogTablesInLocalDropDetachQueue *prometheus.Desc
	StartupScriptsExecutionState                      *prometheus.Desc
	IsServerShuttingDown                              *prometheus.Desc
}

func NewMetricsURIExporter() *MetricsURIExporter {
	Query := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "query"),
		"Number of query currently processed",
		nil,
		nil,
	)

	Merge := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Merge"),
		"Number of Merge currently processed",
		nil,
		nil,
	)
	MergeParts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeParts"),
		"Number of MergeParts currently processed",
		nil,
		nil,
	)
	Move := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Move"),
		"Number of Move currently processed",
		nil,
		nil,
	)
	PartMutation := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartMutation"),
		"Number of PartMutation currently processed",
		nil,
		nil,
	)
	ReplicatedFetch := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReplicatedFetch"),
		"Number of ReplicatedFetch currently processed",
		nil,
		nil,
	)
	ReplicatedSend := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReplicatedSend"),
		"Number of ReplicatedSend currently processed",
		nil,
		nil,
	)
	ReplicatedChecks := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReplicatedChecks"),
		"Number of ReplicatedChecks currently processed",
		nil,
		nil,
	)
	BackgroundMergesAndMutationsPoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMergesAndMutationsPoolTask"),
		"Number of BackgroundMergesAndMutationsPoolTask currently processed",
		nil,
		nil,
	)
	BackgroundMergesAndMutationsPoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMergesAndMutationsPoolSize"),
		"Number of BackgroundMergesAndMutationsPoolSize currently processed",
		nil,
		nil,
	)
	BackgroundFetchesPoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundFetchesPoolTask"),
		"Number of BackgroundFetchesPoolTask currently processed",
		nil,
		nil,
	)
	BackgroundFetchesPoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundFetchesPoolSize"),
		"Number of BackgroundFetchesPoolSize currently processed",
		nil,
		nil,
	)
	BackgroundCommonPoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundCommonPoolTask"),
		"Number of BackgroundCommonPoolTask currently processed",
		nil,
		nil,
	)
	BackgroundCommonPoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundCommonPoolSize"),
		"Number of BackgroundCommonPoolSize currently processed",
		nil,
		nil,
	)
	BackgroundMovePoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMovePoolTask"),
		"Number of BackgroundMovePoolTask currently processed",
		nil,
		nil,
	)
	BackgroundMovePoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMovePoolSize"),
		"Number of BackgroundMovePoolSize currently processed",
		nil,
		nil,
	)
	BackgroundSchedulePoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundSchedulePoolTask"),
		"Number of BackgroundSchedulePoolTask currently processed",
		nil,
		nil,
	)
	BackgroundSchedulePoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundSchedulePoolSize"),
		"Number of BackgroundSchedulePoolSize currently processed",
		nil,
		nil,
	)
	BackgroundBufferFlushSchedulePoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundBufferFlushSchedulePoolTask"),
		"Number of BackgroundBufferFlushSchedulePoolTask currently processed",
		nil,
		nil,
	)
	BackgroundBufferFlushSchedulePoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundBufferFlushSchedulePoolSize"),
		"Number of BackgroundBufferFlushSchedulePoolSize currently processed",
		nil,
		nil,
	)
	BackgroundDistributedSchedulePoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundDistributedSchedulePoolTask"),
		"Number of BackgroundDistributedSchedulePoolTask currently processed",
		nil,
		nil,
	)
	BackgroundDistributedSchedulePoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundDistributedSchedulePoolSize"),
		"Number of BackgroundDistributedSchedulePoolSize currently processed",
		nil,
		nil,
	)
	BackgroundMessageBrokerSchedulePoolTask := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMessageBrokerSchedulePoolTask"),
		"Number of BackgroundMessageBrokerSchedulePoolTask currently processed",
		nil,
		nil,
	)
	BackgroundMessageBrokerSchedulePoolSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackgroundMessageBrokerSchedulePoolSize"),
		"Number of BackgroundMessageBrokerSchedulePoolSize currently processed",
		nil,
		nil,
	)
	CacheDictionaryUpdateQueueBatches := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDictionaryUpdateQueueBatches"),
		"Number of CacheDictionaryUpdateQueueBatches currently processed",
		nil,
		nil,
	)
	CacheDictionaryUpdateQueueKeys := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDictionaryUpdateQueueKeys"),
		"Number of CacheDictionaryUpdateQueueKeys currently processed",
		nil,
		nil,
	)
	DiskSpaceReservedForMerge := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskSpaceReservedForMerge"),
		"Number of DiskSpaceReservedForMerge currently processed",
		nil,
		nil,
	)
	DistributedSend := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedSend"),
		"Number of DistributedSend currently processed",
		nil,
		nil,
	)
	QueryPreempted := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryPreempted"),
		"Number of QueryPreempted currently processed",
		nil,
		nil,
	)
	TCPConnection := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TCPConnection"),
		"Number of TCPConnection currently processed",
		nil,
		nil,
	)
	MySQLConnection := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MySQLConnection"),
		"Number of MySQLConnection currently processed",
		nil,
		nil,
	)
	HTTPConnection := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HTTPConnection"),
		"Number of HTTPConnection currently processed",
		nil,
		nil,
	)
	InterserverConnection := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "InterserverConnection"),
		"Number of InterserverConnection currently processed",
		nil,
		nil,
	)
	PostgreSQLConnection := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PostgreSQLConnection"),
		"Number of PostgreSQLConnection currently processed",
		nil,
		nil,
	)
	OpenFileForRead := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OpenFileForRead"),
		"Number of OpenFileForRead currently processed",
		nil,
		nil,
	)
	OpenFileForWrite := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OpenFileForWrite"),
		"Number of OpenFileForWrite currently processed",
		nil,
		nil,
	)
	Compressing := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Compressing"),
		"Number of Compressing currently processed",
		nil,
		nil,
	)
	Decompressing := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Decompressing"),
		"Number of Decompressing currently processed",
		nil,
		nil,
	)
	ParallelCompressedWriteBufferThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelCompressedWriteBufferThreads"),
		"Number of ParallelCompressedWriteBufferThreads currently processed",
		nil,
		nil,
	)
	ParallelCompressedWriteBufferWait := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelCompressedWriteBufferWait"),
		"Number of ParallelCompressedWriteBufferWait currently processed",
		nil,
		nil,
	)
	TotalTemporaryFiles := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TotalTemporaryFiles"),
		"Number of TotalTemporaryFiles currently processed",
		nil,
		nil,
	)
	TemporaryFilesForSort := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TemporaryFilesForSort"),
		"Number of TemporaryFilesForSort currently processed",
		nil,
		nil,
	)
	TemporaryFilesForAggregation := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TemporaryFilesForAggregation"),
		"Number of TemporaryFilesForAggregation currently processed",
		nil,
		nil,
	)
	TemporaryFilesForJoin := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TemporaryFilesForJoin"),
		"Number of TemporaryFilesForJoin currently processed",
		nil,
		nil,
	)
	TemporaryFilesForMerge := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TemporaryFilesForMerge"),
		"Number of TemporaryFilesForMerge currently processed",
		nil,
		nil,
	)
	TemporaryFilesUnknown := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TemporaryFilesUnknown"),
		"Number of TemporaryFilesUnknown currently processed",
		nil,
		nil,
	)
	Read := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Read"),
		"Number of Read currently processed",
		nil,
		nil,
	)
	RemoteRead := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RemoteRead"),
		"Number of RemoteRead currently processed",
		nil,
		nil,
	)
	Write := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Write"),
		"Number of Write currently processed",
		nil,
		nil,
	)
	NetworkReceive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkReceive"),
		"Number of NetworkReceive currently processed",
		nil,
		nil,
	)
	NetworkSend := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "NetworkSend"),
		"Number of NetworkSend currently processed",
		nil,
		nil,
	)
	SendScalars := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SendScalars"),
		"Number of SendScalars currently processed",
		nil,
		nil,
	)
	SendExternalTables := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SendExternalTables"),
		"Number of SendExternalTables currently processed",
		nil,
		nil,
	)
	QueryThread := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryThread"),
		"Number of QueryThread currently processed",
		nil,
		nil,
	)
	ReadonlyReplica := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReadonlyReplica"),
		"Number of ReadonlyReplica currently processed",
		nil,
		nil,
	)
	MemoryTracking := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MemoryTracking"),
		"Number of MemoryTracking currently processed",
		nil,
		nil,
	)
	MemoryTrackingUncorrected := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MemoryTrackingUncorrected"),
		"Number of MemoryTrackingUncorrected currently processed",
		nil,
		nil,
	)
	MergesMutationsMemoryTracking := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergesMutationsMemoryTracking"),
		"Number of MergesMutationsMemoryTracking currently processed",
		nil,
		nil,
	)
	EphemeralNode := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "EphemeralNode"),
		"Number of EphemeralNode currently processed",
		nil,
		nil,
	)
	ZooKeeperSession := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ZooKeeperSession"),
		"Number of ZooKeeperSession currently processed",
		nil,
		nil,
	)
	ZooKeeperWatch := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ZooKeeperWatch"),
		"Number of ZooKeeperWatch currently processed",
		nil,
		nil,
	)
	ZooKeeperRequest := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ZooKeeperRequest"),
		"Number of ZooKeeperRequest currently processed",
		nil,
		nil,
	)
	DelayedInserts := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DelayedInserts"),
		"Number of DelayedInserts currently processed",
		nil,
		nil,
	)
	ContextLockWait := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ContextLockWait"),
		"Number of ContextLockWait currently processed",
		nil,
		nil,
	)
	StorageBufferRows := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageBufferRows"),
		"Number of StorageBufferRows currently processed",
		nil,
		nil,
	)
	StorageBufferBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageBufferBytes"),
		"Number of StorageBufferBytes currently processed",
		nil,
		nil,
	)
	DictCacheRequests := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DictCacheRequests"),
		"Number of DictCacheRequests currently processed",
		nil,
		nil,
	)
	Revision := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "Revision"),
		"Number of Revision currently processed",
		nil,
		nil,
	)
	VersionInteger := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "VersionInteger"),
		"Number of VersionInteger currently processed",
		nil,
		nil,
	)
	RWLockWaitingReaders := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockWaitingReaders"),
		"Number of RWLockWaitingReaders currently processed",
		nil,
		nil,
	)
	RWLockWaitingWriters := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockWaitingWriters"),
		"Number of RWLockWaitingWriters currently processed",
		nil,
		nil,
	)
	RWLockActiveReaders := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockActiveReaders"),
		"Number of RWLockActiveReaders currently processed",
		nil,
		nil,
	)
	RWLockActiveWriters := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RWLockActiveWriters"),
		"Number of RWLockActiveWriters currently processed",
		nil,
		nil,
	)
	GlobalThread := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThread"),
		"Number of GlobalThread currently processed",
		nil,
		nil,
	)
	GlobalThreadActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadActive"),
		"Number of GlobalThreadActive currently processed",
		nil,
		nil,
	)
	GlobalThreadScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "GlobalThreadScheduled"),
		"Number of GlobalThreadScheduled currently processed",
		nil,
		nil,
	)
	LocalThread := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThread"),
		"Number of LocalThread currently processed",
		nil,
		nil,
	)
	LocalThreadActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadActive"),
		"Number of LocalThreadActive currently processed",
		nil,
		nil,
	)
	LocalThreadScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "LocalThreadScheduled"),
		"Number of LocalThreadScheduled currently processed",
		nil,
		nil,
	)
	MergeTreeDataSelectExecutorThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataSelectExecutorThreads"),
		"Number of MergeTreeDataSelectExecutorThreads currently processed",
		nil,
		nil,
	)
	MergeTreeDataSelectExecutorThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataSelectExecutorThreadsActive"),
		"Number of MergeTreeDataSelectExecutorThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeDataSelectExecutorThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeDataSelectExecutorThreadsScheduled"),
		"Number of MergeTreeDataSelectExecutorThreadsScheduled currently processed",
		nil,
		nil,
	)
	BackupsThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsThreads"),
		"Number of BackupsThreads currently processed",
		nil,
		nil,
	)
	BackupsThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsThreadsActive"),
		"Number of BackupsThreadsActive currently processed",
		nil,
		nil,
	)
	BackupsThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsThreadsScheduled"),
		"Number of BackupsThreadsScheduled currently processed",
		nil,
		nil,
	)
	RestoreThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestoreThreads"),
		"Number of RestoreThreads currently processed",
		nil,
		nil,
	)
	RestoreThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestoreThreadsActive"),
		"Number of RestoreThreadsActive currently processed",
		nil,
		nil,
	)
	RestoreThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestoreThreadsScheduled"),
		"Number of RestoreThreadsScheduled currently processed",
		nil,
		nil,
	)
	MarksLoaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MarksLoaderThreads"),
		"Number of MarksLoaderThreads currently processed",
		nil,
		nil,
	)
	MarksLoaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MarksLoaderThreadsActive"),
		"Number of MarksLoaderThreadsActive currently processed",
		nil,
		nil,
	)
	MarksLoaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MarksLoaderThreadsScheduled"),
		"Number of MarksLoaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	IOPrefetchThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOPrefetchThreads"),
		"Number of IOPrefetchThreads currently processed",
		nil,
		nil,
	)
	IOPrefetchThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOPrefetchThreadsActive"),
		"Number of IOPrefetchThreadsActive currently processed",
		nil,
		nil,
	)
	IOPrefetchThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOPrefetchThreadsScheduled"),
		"Number of IOPrefetchThreadsScheduled currently processed",
		nil,
		nil,
	)
	IOWriterThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOWriterThreads"),
		"Number of IOWriterThreads currently processed",
		nil,
		nil,
	)
	IOWriterThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOWriterThreadsActive"),
		"Number of IOWriterThreadsActive currently processed",
		nil,
		nil,
	)
	IOWriterThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOWriterThreadsScheduled"),
		"Number of IOWriterThreadsScheduled currently processed",
		nil,
		nil,
	)
	IOThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOThreads"),
		"Number of IOThreads currently processed",
		nil,
		nil,
	)
	IOThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOThreadsActive"),
		"Number of IOThreadsActive currently processed",
		nil,
		nil,
	)
	IOThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOThreadsScheduled"),
		"Number of IOThreadsScheduled currently processed",
		nil,
		nil,
	)
	CompressionThread := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompressionThread"),
		"Number of CompressionThread currently processed",
		nil,
		nil,
	)
	CompressionThreadActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompressionThreadActive"),
		"Number of CompressionThreadActive currently processed",
		nil,
		nil,
	)
	CompressionThreadScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CompressionThreadScheduled"),
		"Number of CompressionThreadScheduled currently processed",
		nil,
		nil,
	)
	ThreadPoolRemoteFSReaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolRemoteFSReaderThreads"),
		"Number of ThreadPoolRemoteFSReaderThreads currently processed",
		nil,
		nil,
	)
	ThreadPoolRemoteFSReaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolRemoteFSReaderThreadsActive"),
		"Number of ThreadPoolRemoteFSReaderThreadsActive currently processed",
		nil,
		nil,
	)
	ThreadPoolRemoteFSReaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolRemoteFSReaderThreadsScheduled"),
		"Number of ThreadPoolRemoteFSReaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	ThreadPoolFSReaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolFSReaderThreads"),
		"Number of ThreadPoolFSReaderThreads currently processed",
		nil,
		nil,
	)
	ThreadPoolFSReaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolFSReaderThreadsActive"),
		"Number of ThreadPoolFSReaderThreadsActive currently processed",
		nil,
		nil,
	)
	ThreadPoolFSReaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadPoolFSReaderThreadsScheduled"),
		"Number of ThreadPoolFSReaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	BackupsIOThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsIOThreads"),
		"Number of BackupsIOThreads currently processed",
		nil,
		nil,
	)
	BackupsIOThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsIOThreadsActive"),
		"Number of BackupsIOThreadsActive currently processed",
		nil,
		nil,
	)
	BackupsIOThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BackupsIOThreadsScheduled"),
		"Number of BackupsIOThreadsScheduled currently processed",
		nil,
		nil,
	)
	DiskObjectStorageAsyncThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskObjectStorageAsyncThreads"),
		"Number of DiskObjectStorageAsyncThreads currently processed",
		nil,
		nil,
	)
	DiskObjectStorageAsyncThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskObjectStorageAsyncThreadsActive"),
		"Number of DiskObjectStorageAsyncThreadsActive currently processed",
		nil,
		nil,
	)
	StorageHiveThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageHiveThreads"),
		"Number of StorageHiveThreads currently processed",
		nil,
		nil,
	)
	StorageHiveThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageHiveThreadsActive"),
		"Number of StorageHiveThreadsActive currently processed",
		nil,
		nil,
	)
	StorageHiveThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageHiveThreadsScheduled"),
		"Number of StorageHiveThreadsScheduled currently processed",
		nil,
		nil,
	)
	TablesLoaderBackgroundThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderBackgroundThreads"),
		"Number of TablesLoaderBackgroundThreads currently processed",
		nil,
		nil,
	)
	TablesLoaderBackgroundThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderBackgroundThreadsActive"),
		"Number of TablesLoaderBackgroundThreadsActive currently processed",
		nil,
		nil,
	)
	TablesLoaderBackgroundThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderBackgroundThreadsScheduled"),
		"Number of TablesLoaderBackgroundThreadsScheduled currently processed",
		nil,
		nil,
	)
	TablesLoaderForegroundThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderForegroundThreads"),
		"Number of TablesLoaderForegroundThreads currently processed",
		nil,
		nil,
	)
	TablesLoaderForegroundThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderForegroundThreadsActive"),
		"Number of TablesLoaderForegroundThreadsActive currently processed",
		nil,
		nil,
	)
	TablesLoaderForegroundThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesLoaderForegroundThreadsScheduled"),
		"Number of TablesLoaderForegroundThreadsScheduled currently processed",
		nil,
		nil,
	)
	DatabaseOnDiskThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseOnDiskThreads"),
		"Number of DatabaseOnDiskThreads currently processed",
		nil,
		nil,
	)
	DatabaseOnDiskThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseOnDiskThreadsActive"),
		"Number of DatabaseOnDiskThreadsActive currently processed",
		nil,
		nil,
	)
	DatabaseOnDiskThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseOnDiskThreadsScheduled"),
		"Number of DatabaseOnDiskThreadsScheduled currently processed",
		nil,
		nil,
	)
	DatabaseBackupThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseBackupThreads"),
		"Number of DatabaseBackupThreads currently processed",
		nil,
		nil,
	)
	DatabaseBackupThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseBackupThreadsActive"),
		"Number of DatabaseBackupThreadsActive currently processed",
		nil,
		nil,
	)
	DatabaseBackupThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseBackupThreadsScheduled"),
		"Number of DatabaseBackupThreadsScheduled currently processed",
		nil,
		nil,
	)
	DatabaseCatalogThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseCatalogThreads"),
		"Number of DatabaseCatalogThreads currently processed",
		nil,
		nil,
	)
	DatabaseCatalogThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseCatalogThreadsActive"),
		"Number of DatabaseCatalogThreadsActive currently processed",
		nil,
		nil,
	)
	DatabaseCatalogThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseCatalogThreadsScheduled"),
		"Number of DatabaseCatalogThreadsScheduled currently processed",
		nil,
		nil,
	)
	DestroyAggregatesThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DestroyAggregatesThreads"),
		"Number of DestroyAggregatesThreads currently processed",
		nil,
		nil,
	)
	DestroyAggregatesThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DestroyAggregatesThreadsActive"),
		"Number of DestroyAggregatesThreadsActive currently processed",
		nil,
		nil,
	)
	DestroyAggregatesThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DestroyAggregatesThreadsScheduled"),
		"Number of DestroyAggregatesThreadsScheduled currently processed",
		nil,
		nil,
	)
	ConcurrentHashJoinPoolThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrentHashJoinPoolThreads"),
		"Number of ConcurrentHashJoinPoolThreads currently processed",
		nil,
		nil,
	)
	ConcurrentHashJoinPoolThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrentHashJoinPoolThreadsActive"),
		"Number of ConcurrentHashJoinPoolThreadsActive currently processed",
		nil,
		nil,
	)
	ConcurrentHashJoinPoolThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrentHashJoinPoolThreadsScheduled"),
		"Number of ConcurrentHashJoinPoolThreadsScheduled currently processed",
		nil,
		nil,
	)
	HashedDictionaryThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HashedDictionaryThreads"),
		"Number of HashedDictionaryThreads currently processed",
		nil,
		nil,
	)
	HashedDictionaryThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HashedDictionaryThreadsActive"),
		"Number of HashedDictionaryThreadsActive currently processed",
		nil,
		nil,
	)
	HashedDictionaryThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HashedDictionaryThreadsScheduled"),
		"Number of HashedDictionaryThreadsScheduled currently processed",
		nil,
		nil,
	)
	CacheDictionaryThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDictionaryThreads"),
		"Number of CacheDictionaryThreads currently processed",
		nil,
		nil,
	)
	CacheDictionaryThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDictionaryThreadsActive"),
		"Number of CacheDictionaryThreadsActive currently processed",
		nil,
		nil,
	)
	CacheDictionaryThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDictionaryThreadsScheduled"),
		"Number of CacheDictionaryThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParallelFormattingOutputFormatThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelFormattingOutputFormatThreads"),
		"Number of ParallelFormattingOutputFormatThreads currently processed",
		nil,
		nil,
	)
	ParallelFormattingOutputFormatThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelFormattingOutputFormatThreadsActive"),
		"Number of ParallelFormattingOutputFormatThreadsActive currently processed",
		nil,
		nil,
	)
	ParallelFormattingOutputFormatThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelFormattingOutputFormatThreadsScheduled"),
		"Number of ParallelFormattingOutputFormatThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParallelParsingInputFormatThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelParsingInputFormatThreads"),
		"Number of ParallelParsingInputFormatThreads currently processed",
		nil,
		nil,
	)
	ParallelParsingInputFormatThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelParsingInputFormatThreadsActive"),
		"Number of ParallelParsingInputFormatThreadsActive currently processed",
		nil,
		nil,
	)
	ParallelParsingInputFormatThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelParsingInputFormatThreadsScheduled"),
		"Number of ParallelParsingInputFormatThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreeBackgroundExecutorThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeBackgroundExecutorThreads"),
		"Number of MergeTreeBackgroundExecutorThreads currently processed",
		nil,
		nil,
	)
	MergeTreeBackgroundExecutorThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeBackgroundExecutorThreadsActive"),
		"Number of MergeTreeBackgroundExecutorThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeBackgroundExecutorThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeBackgroundExecutorThreadsScheduled"),
		"Number of MergeTreeBackgroundExecutorThreadsScheduled currently processed",
		nil,
		nil,
	)
	AsynchronousInsertThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousInsertThreads"),
		"Number of AsynchronousInsertThreads currently processed",
		nil,
		nil,
	)
	AsynchronousInsertThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousInsertThreadsActive"),
		"Number of AsynchronousInsertThreadsActive currently processed",
		nil,
		nil,
	)
	AsynchronousInsertThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousInsertThreadsScheduled"),
		"Number of AsynchronousInsertThreadsScheduled currently processed",
		nil,
		nil,
	)
	AsynchronousInsertQueueSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousInsertQueueSize"),
		"Number of AsynchronousInsertQueueSize currently processed",
		nil,
		nil,
	)
	AsynchronousInsertQueueBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousInsertQueueBytes"),
		"Number of AsynchronousInsertQueueBytes currently processed",
		nil,
		nil,
	)
	StartupSystemTablesThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StartupSystemTablesThreads"),
		"Number of StartupSystemTablesThreads currently processed",
		nil,
		nil,
	)
	StartupSystemTablesThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StartupSystemTablesThreadsActive"),
		"Number of StartupSystemTablesThreadsActive currently processed",
		nil,
		nil,
	)
	StartupSystemTablesThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StartupSystemTablesThreadsScheduled"),
		"Number of StartupSystemTablesThreadsScheduled currently processed",
		nil,
		nil,
	)
	AggregatorThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AggregatorThreads"),
		"Number of AggregatorThreads currently processed",
		nil,
		nil,
	)
	AggregatorThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AggregatorThreadsActive"),
		"Number of AggregatorThreadsActive currently processed",
		nil,
		nil,
	)
	AggregatorThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AggregatorThreadsScheduled"),
		"Number of AggregatorThreadsScheduled currently processed",
		nil,
		nil,
	)
	DDLWorkerThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DDLWorkerThreads"),
		"Number of DDLWorkerThreads currently processed",
		nil,
		nil,
	)
	DDLWorkerThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DDLWorkerThreadsActive"),
		"Number of DDLWorkerThreadsActive currently processed",
		nil,
		nil,
	)
	DDLWorkerThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DDLWorkerThreadsScheduled"),
		"Number of DDLWorkerThreadsScheduled currently processed",
		nil,
		nil,
	)
	StorageDistributedThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageDistributedThreads"),
		"Number of StorageDistributedThreads currently processed",
		nil,
		nil,
	)
	StorageDistributedThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageDistributedThreadsActive"),
		"Number of StorageDistributedThreadsActive currently processed",
		nil,
		nil,
	)
	StorageDistributedThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageDistributedThreadsScheduled"),
		"Number of StorageDistributedThreadsScheduled currently processed",
		nil,
		nil,
	)
	DistributedInsertThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedInsertThreads"),
		"Number of DistributedInsertThreads currently processed",
		nil,
		nil,
	)
	DistributedInsertThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedInsertThreadsActive"),
		"Number of DistributedInsertThreadsActive currently processed",
		nil,
		nil,
	)
	DistributedInsertThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedInsertThreadsScheduled"),
		"Number of DistributedInsertThreadsScheduled currently processed",
		nil,
		nil,
	)
	StorageS3Threads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageS3Threads"),
		"Number of StorageS3Threads currently processed",
		nil,
		nil,
	)
	StorageS3ThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageS3ThreadsActive"),
		"Number of StorageS3ThreadsActive currently processed",
		nil,
		nil,
	)
	StorageS3ThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageS3ThreadsScheduled"),
		"Number of StorageS3ThreadsScheduled currently processed",
		nil,
		nil,
	)
	ObjectStorageS3Threads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageS3Threads"),
		"Number of ObjectStorageS3Threads currently processed",
		nil,
		nil,
	)
	ObjectStorageS3ThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageS3ThreadsActive"),
		"Number of ObjectStorageS3ThreadsActive currently processed",
		nil,
		nil,
	)
	ObjectStorageS3ThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageS3ThreadsScheduled"),
		"Number of ObjectStorageS3ThreadsScheduled currently processed",
		nil,
		nil,
	)
	StorageObjectStorageThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageObjectStorageThreads"),
		"Number of StorageObjectStorageThreads currently processed",
		nil,
		nil,
	)
	StorageObjectStorageThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageObjectStorageThreadsActive"),
		"Number of StorageObjectStorageThreadsActive currently processed",
		nil,
		nil,
	)
	StorageObjectStorageThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageObjectStorageThreadsScheduled"),
		"Number of StorageObjectStorageThreadsScheduled currently processed",
		nil,
		nil,
	)
	ObjectStorageAzureThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageAzureThreads"),
		"Number of ObjectStorageAzureThreads currently processed",
		nil,
		nil,
	)
	ObjectStorageAzureThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageAzureThreadsActive"),
		"Number of ObjectStorageAzureThreadsActive currently processed",
		nil,
		nil,
	)
	ObjectStorageAzureThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageAzureThreadsScheduled"),
		"Number of ObjectStorageAzureThreadsScheduled currently processed",
		nil,
		nil,
	)
	BuildVectorSimilarityIndexThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BuildVectorSimilarityIndexThreads"),
		"Number of BuildVectorSimilarityIndexThreads currently processed",
		nil,
		nil,
	)
	BuildVectorSimilarityIndexThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BuildVectorSimilarityIndexThreadsActive"),
		"Number of BuildVectorSimilarityIndexThreadsActive currently processed",
		nil,
		nil,
	)
	BuildVectorSimilarityIndexThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BuildVectorSimilarityIndexThreadsScheduled"),
		"Number of BuildVectorSimilarityIndexThreadsScheduled currently processed",
		nil,
		nil,
	)
	ObjectStorageQueueRegisteredServers := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ObjectStorageQueueRegisteredServers"),
		"Number of ObjectStorageQueueRegisteredServers currently processed",
		nil,
		nil,
	)
	IcebergCatalogThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IcebergCatalogThreads"),
		"Number of IcebergCatalogThreads currently processed",
		nil,
		nil,
	)
	IcebergCatalogThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IcebergCatalogThreadsActive"),
		"Number of IcebergCatalogThreadsActive currently processed",
		nil,
		nil,
	)
	IcebergCatalogThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IcebergCatalogThreadsScheduled"),
		"Number of IcebergCatalogThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParallelWithQueryThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelWithQueryThreads"),
		"Number of ParallelWithQueryThreads currently processed",
		nil,
		nil,
	)
	ParallelWithQueryActiveThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelWithQueryActiveThreads"),
		"Number of ParallelWithQueryActiveThreads currently processed",
		nil,
		nil,
	)
	ParallelWithQueryScheduledThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParallelWithQueryScheduledThreads"),
		"Number of ParallelWithQueryScheduledThreads currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableAzureDirectoryMapSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableAzureDirectoryMapSize"),
		"Number of DiskPlainRewritableAzureDirectoryMapSize currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableAzureFileCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableAzureFileCount"),
		"Number of DiskPlainRewritableAzureFileCount currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableAzureUniqueFileNamesCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableAzureUniqueFileNamesCount"),
		"Number of DiskPlainRewritableAzureUniqueFileNamesCount currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableLocalDirectoryMapSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableLocalDirectoryMapSize"),
		"Number of DiskPlainRewritableLocalDirectoryMapSize currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableLocalFileCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableLocalFileCount"),
		"Number of DiskPlainRewritableLocalFileCount currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableLocalUniqueFileNamesCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableLocalUniqueFileNamesCount"),
		"Number of DiskPlainRewritableLocalUniqueFileNamesCount currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableS3DirectoryMapSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableS3DirectoryMapSize"),
		"Number of DiskPlainRewritableS3DirectoryMapSize currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableS3FileCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableS3FileCount"),
		"Number of DiskPlainRewritableS3FileCount currently processed",
		nil,
		nil,
	)
	DiskPlainRewritableS3UniqueFileNamesCount := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskPlainRewritableS3UniqueFileNamesCount"),
		"Number of DiskPlainRewritableS3UniqueFileNamesCount currently processed",
		nil,
		nil,
	)
	MergeTreeFetchPartitionThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeFetchPartitionThreads"),
		"Number of MergeTreeFetchPartitionThreads currently processed",
		nil,
		nil,
	)
	MergeTreeFetchPartitionThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeFetchPartitionThreadsActive"),
		"Number of MergeTreeFetchPartitionThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeFetchPartitionThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeFetchPartitionThreadsScheduled"),
		"Number of MergeTreeFetchPartitionThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreePartsLoaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsLoaderThreads"),
		"Number of MergeTreePartsLoaderThreads currently processed",
		nil,
		nil,
	)
	MergeTreePartsLoaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsLoaderThreadsActive"),
		"Number of MergeTreePartsLoaderThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreePartsLoaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsLoaderThreadsScheduled"),
		"Number of MergeTreePartsLoaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreeOutdatedPartsLoaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeOutdatedPartsLoaderThreads"),
		"Number of MergeTreeOutdatedPartsLoaderThreads currently processed",
		nil,
		nil,
	)
	MergeTreeOutdatedPartsLoaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeOutdatedPartsLoaderThreadsActive"),
		"Number of MergeTreeOutdatedPartsLoaderThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeOutdatedPartsLoaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeOutdatedPartsLoaderThreadsScheduled"),
		"Number of MergeTreeOutdatedPartsLoaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreeUnexpectedPartsLoaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeUnexpectedPartsLoaderThreads"),
		"Number of MergeTreeUnexpectedPartsLoaderThreads currently processed",
		nil,
		nil,
	)
	MergeTreeUnexpectedPartsLoaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeUnexpectedPartsLoaderThreadsActive"),
		"Number of MergeTreeUnexpectedPartsLoaderThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeUnexpectedPartsLoaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeUnexpectedPartsLoaderThreadsScheduled"),
		"Number of MergeTreeUnexpectedPartsLoaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreePartsCleanerThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsCleanerThreads"),
		"Number of MergeTreePartsCleanerThreads currently processed",
		nil,
		nil,
	)
	MergeTreePartsCleanerThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsCleanerThreadsActive"),
		"Number of MergeTreePartsCleanerThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreePartsCleanerThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreePartsCleanerThreadsScheduled"),
		"Number of MergeTreePartsCleanerThreadsScheduled currently processed",
		nil,
		nil,
	)
	DatabaseReplicatedCreateTablesThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseReplicatedCreateTablesThreads"),
		"Number of DatabaseReplicatedCreateTablesThreads currently processed",
		nil,
		nil,
	)
	DatabaseReplicatedCreateTablesThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseReplicatedCreateTablesThreadsActive"),
		"Number of DatabaseReplicatedCreateTablesThreadsActive currently processed",
		nil,
		nil,
	)
	DatabaseReplicatedCreateTablesThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DatabaseReplicatedCreateTablesThreadsScheduled"),
		"Number of DatabaseReplicatedCreateTablesThreadsScheduled currently processed",
		nil,
		nil,
	)
	IDiskCopierThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IDiskCopierThreads"),
		"Number of IDiskCopierThreads currently processed",
		nil,
		nil,
	)
	IDiskCopierThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IDiskCopierThreadsActive"),
		"Number of IDiskCopierThreadsActive currently processed",
		nil,
		nil,
	)
	IDiskCopierThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IDiskCopierThreadsScheduled"),
		"Number of IDiskCopierThreadsScheduled currently processed",
		nil,
		nil,
	)
	SystemReplicasThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SystemReplicasThreads"),
		"Number of SystemReplicasThreads currently processed",
		nil,
		nil,
	)
	SystemReplicasThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SystemReplicasThreadsActive"),
		"Number of SystemReplicasThreadsActive currently processed",
		nil,
		nil,
	)
	SystemReplicasThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SystemReplicasThreadsScheduled"),
		"Number of SystemReplicasThreadsScheduled currently processed",
		nil,
		nil,
	)
	RestartReplicaThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestartReplicaThreads"),
		"Number of RestartReplicaThreads currently processed",
		nil,
		nil,
	)
	RestartReplicaThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestartReplicaThreadsActive"),
		"Number of RestartReplicaThreadsActive currently processed",
		nil,
		nil,
	)
	RestartReplicaThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RestartReplicaThreadsScheduled"),
		"Number of RestartReplicaThreadsScheduled currently processed",
		nil,
		nil,
	)
	QueryPipelineExecutorThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryPipelineExecutorThreads"),
		"Number of QueryPipelineExecutorThreads currently processed",
		nil,
		nil,
	)
	QueryPipelineExecutorThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryPipelineExecutorThreadsActive"),
		"Number of QueryPipelineExecutorThreadsActive currently processed",
		nil,
		nil,
	)
	QueryPipelineExecutorThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "QueryPipelineExecutorThreadsScheduled"),
		"Number of QueryPipelineExecutorThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParquetDecoderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderThreads"),
		"Number of ParquetDecoderThreads currently processed",
		nil,
		nil,
	)
	ParquetDecoderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderThreadsActive"),
		"Number of ParquetDecoderThreadsActive currently processed",
		nil,
		nil,
	)
	ParquetDecoderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderThreadsScheduled"),
		"Number of ParquetDecoderThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParquetDecoderIOThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderIOThreads"),
		"Number of ParquetDecoderIOThreads currently processed",
		nil,
		nil,
	)
	ParquetDecoderIOThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderIOThreadsActive"),
		"Number of ParquetDecoderIOThreadsActive currently processed",
		nil,
		nil,
	)
	ParquetDecoderIOThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetDecoderIOThreadsScheduled"),
		"Number of ParquetDecoderIOThreadsScheduled currently processed",
		nil,
		nil,
	)
	ParquetEncoderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetEncoderThreads"),
		"Number of ParquetEncoderThreads currently processed",
		nil,
		nil,
	)
	ParquetEncoderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetEncoderThreadsActive"),
		"Number of ParquetEncoderThreadsActive currently processed",
		nil,
		nil,
	)
	ParquetEncoderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ParquetEncoderThreadsScheduled"),
		"Number of ParquetEncoderThreadsScheduled currently processed",
		nil,
		nil,
	)
	MergeTreeSubcolumnsReaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeSubcolumnsReaderThreads"),
		"Number of MergeTreeSubcolumnsReaderThreads currently processed",
		nil,
		nil,
	)
	MergeTreeSubcolumnsReaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeSubcolumnsReaderThreadsActive"),
		"Number of MergeTreeSubcolumnsReaderThreadsActive currently processed",
		nil,
		nil,
	)
	MergeTreeSubcolumnsReaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeSubcolumnsReaderThreadsScheduled"),
		"Number of MergeTreeSubcolumnsReaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	DWARFReaderThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DWARFReaderThreads"),
		"Number of DWARFReaderThreads currently processed",
		nil,
		nil,
	)
	DWARFReaderThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DWARFReaderThreadsActive"),
		"Number of DWARFReaderThreadsActive currently processed",
		nil,
		nil,
	)
	DWARFReaderThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DWARFReaderThreadsScheduled"),
		"Number of DWARFReaderThreadsScheduled currently processed",
		nil,
		nil,
	)
	OutdatedPartsLoadingThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OutdatedPartsLoadingThreads"),
		"Number of OutdatedPartsLoadingThreads currently processed",
		nil,
		nil,
	)
	OutdatedPartsLoadingThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OutdatedPartsLoadingThreadsActive"),
		"Number of OutdatedPartsLoadingThreadsActive currently processed",
		nil,
		nil,
	)
	OutdatedPartsLoadingThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "OutdatedPartsLoadingThreadsScheduled"),
		"Number of OutdatedPartsLoadingThreadsScheduled currently processed",
		nil,
		nil,
	)
	PolygonDictionaryThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PolygonDictionaryThreads"),
		"Number of PolygonDictionaryThreads currently processed",
		nil,
		nil,
	)
	PolygonDictionaryThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PolygonDictionaryThreadsActive"),
		"Number of PolygonDictionaryThreadsActive currently processed",
		nil,
		nil,
	)
	PolygonDictionaryThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PolygonDictionaryThreadsScheduled"),
		"Number of PolygonDictionaryThreadsScheduled currently processed",
		nil,
		nil,
	)
	DistributedBytesToInsert := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedBytesToInsert"),
		"Number of DistributedBytesToInsert currently processed",
		nil,
		nil,
	)
	BrokenDistributedBytesToInsert := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BrokenDistributedBytesToInsert"),
		"Number of BrokenDistributedBytesToInsert currently processed",
		nil,
		nil,
	)
	DistributedFilesToInsert := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistributedFilesToInsert"),
		"Number of DistributedFilesToInsert currently processed",
		nil,
		nil,
	)
	BrokenDistributedFilesToInsert := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "BrokenDistributedFilesToInsert"),
		"Number of BrokenDistributedFilesToInsert currently processed",
		nil,
		nil,
	)
	TablesToDropQueueSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "TablesToDropQueueSize"),
		"Number of TablesToDropQueueSize currently processed",
		nil,
		nil,
	)
	MaxDDLEntryID := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MaxDDLEntryID"),
		"Number of MaxDDLEntryID currently processed",
		nil,
		nil,
	)
	MaxPushedDDLEntryID := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MaxPushedDDLEntryID"),
		"Number of MaxPushedDDLEntryID currently processed",
		nil,
		nil,
	)
	PartsTemporary := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsTemporary"),
		"Number of PartsTemporary currently processed",
		nil,
		nil,
	)
	PartsPreCommitted := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsPreCommitted"),
		"Number of PartsPreCommitted currently processed",
		nil,
		nil,
	)
	PartsCommitted := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsCommitted"),
		"Number of PartsCommitted currently processed",
		nil,
		nil,
	)
	PartsPreActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsPreActive"),
		"Number of PartsPreActive currently processed",
		nil,
		nil,
	)
	PartsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsActive"),
		"Number of PartsActive currently processed",
		nil,
		nil,
	)
	AttachedDatabase := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AttachedDatabase"),
		"Number of AttachedDatabase currently processed",
		nil,
		nil,
	)
	AttachedTable := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AttachedTable"),
		"Number of AttachedTable currently processed",
		nil,
		nil,
	)
	AttachedReplicatedTable := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AttachedReplicatedTable"),
		"Number of AttachedReplicatedTable currently processed",
		nil,
		nil,
	)
	AttachedView := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AttachedView"),
		"Number of AttachedView currently processed",
		nil,
		nil,
	)
	AttachedDictionary := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AttachedDictionary"),
		"Number of AttachedDictionary currently processed",
		nil,
		nil,
	)
	PartsOutdated := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsOutdated"),
		"Number of PartsOutdated currently processed",
		nil,
		nil,
	)
	PartsDeleting := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsDeleting"),
		"Number of PartsDeleting currently processed",
		nil,
		nil,
	)
	PartsDeleteOnDestroy := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsDeleteOnDestroy"),
		"Number of PartsDeleteOnDestroy currently processed",
		nil,
		nil,
	)
	PartsWide := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsWide"),
		"Number of PartsWide currently processed",
		nil,
		nil,
	)
	PartsCompact := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PartsCompact"),
		"Number of PartsCompact currently processed",
		nil,
		nil,
	)
	MMappedFiles := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MMappedFiles"),
		"Number of MMappedFiles currently processed",
		nil,
		nil,
	)
	MMappedFileBytes := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MMappedFileBytes"),
		"Number of MMappedFileBytes currently processed",
		nil,
		nil,
	)
	AsynchronousReadWait := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsynchronousReadWait"),
		"Number of AsynchronousReadWait currently processed",
		nil,
		nil,
	)
	PendingAsyncInsert := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "PendingAsyncInsert"),
		"Number of PendingAsyncInsert currently processed",
		nil,
		nil,
	)
	KafkaConsumers := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaConsumers"),
		"Number of KafkaConsumers currently processed",
		nil,
		nil,
	)
	KafkaConsumersWithAssignment := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaConsumersWithAssignment"),
		"Number of KafkaConsumersWithAssignment currently processed",
		nil,
		nil,
	)
	KafkaProducers := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaProducers"),
		"Number of KafkaProducers currently processed",
		nil,
		nil,
	)
	KafkaLibrdkafkaThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaLibrdkafkaThreads"),
		"Number of KafkaLibrdkafkaThreads currently processed",
		nil,
		nil,
	)
	KafkaBackgroundReads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaBackgroundReads"),
		"Number of KafkaBackgroundReads currently processed",
		nil,
		nil,
	)
	KafkaConsumersInUse := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaConsumersInUse"),
		"Number of KafkaConsumersInUse currently processed",
		nil,
		nil,
	)
	KafkaWrites := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaWrites"),
		"Number of KafkaWrites currently processed",
		nil,
		nil,
	)
	KafkaAssignedPartitions := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KafkaAssignedPartitions"),
		"Number of KafkaAssignedPartitions currently processed",
		nil,
		nil,
	)
	FilesystemCacheReadBuffers := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheReadBuffers"),
		"Number of FilesystemCacheReadBuffers currently processed",
		nil,
		nil,
	)
	CacheFileSegments := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheFileSegments"),
		"Number of CacheFileSegments currently processed",
		nil,
		nil,
	)
	CacheDetachedFileSegments := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheDetachedFileSegments"),
		"Number of CacheDetachedFileSegments currently processed",
		nil,
		nil,
	)
	FilesystemCacheSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheSize"),
		"Number of FilesystemCacheSize currently processed",
		nil,
		nil,
	)
	FilesystemCacheSizeLimit := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheSizeLimit"),
		"Number of FilesystemCacheSizeLimit currently processed",
		nil,
		nil,
	)
	FilesystemCacheElements := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheElements"),
		"Number of FilesystemCacheElements currently processed",
		nil,
		nil,
	)
	FilesystemCacheDownloadQueueElements := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheDownloadQueueElements"),
		"Number of FilesystemCacheDownloadQueueElements currently processed",
		nil,
		nil,
	)
	FilesystemCacheDelayedCleanupElements := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheDelayedCleanupElements"),
		"Number of FilesystemCacheDelayedCleanupElements currently processed",
		nil,
		nil,
	)
	FilesystemCacheHoldFileSegments := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilesystemCacheHoldFileSegments"),
		"Number of FilesystemCacheHoldFileSegments currently processed",
		nil,
		nil,
	)
	AsyncInsertCacheSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AsyncInsertCacheSize"),
		"Number of AsyncInsertCacheSize currently processed",
		nil,
		nil,
	)
	SkippingIndexCacheSize := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SkippingIndexCacheSize"),
		"Number of SkippingIndexCacheSize currently processed",
		nil,
		nil,
	)
	S3Requests := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "S3Requests"),
		"Number of S3Requests currently processed",
		nil,
		nil,
	)
	KeeperAliveConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KeeperAliveConnections"),
		"Number of KeeperAliveConnections currently processed",
		nil,
		nil,
	)
	KeeperOutstandingRequests := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "KeeperOutstandingRequests"),
		"Number of KeeperOutstandingRequests currently processed",
		nil,
		nil,
	)
	ThreadsInOvercommitTracker := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ThreadsInOvercommitTracker"),
		"Number of ThreadsInOvercommitTracker currently processed",
		nil,
		nil,
	)
	IOUringPendingEvents := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOUringPendingEvents"),
		"Number of IOUringPendingEvents currently processed",
		nil,
		nil,
	)
	IOUringInFlightEvents := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IOUringInFlightEvents"),
		"Number of IOUringInFlightEvents currently processed",
		nil,
		nil,
	)
	ReadTaskRequestsSent := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ReadTaskRequestsSent"),
		"Number of ReadTaskRequestsSent currently processed",
		nil,
		nil,
	)
	MergeTreeReadTaskRequestsSent := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeReadTaskRequestsSent"),
		"Number of MergeTreeReadTaskRequestsSent currently processed",
		nil,
		nil,
	)
	MergeTreeAllRangesAnnouncementsSent := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "MergeTreeAllRangesAnnouncementsSent"),
		"Number of MergeTreeAllRangesAnnouncementsSent currently processed",
		nil,
		nil,
	)
	CreatedTimersInQueryProfiler := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CreatedTimersInQueryProfiler"),
		"Number of CreatedTimersInQueryProfiler currently processed",
		nil,
		nil,
	)
	ActiveTimersInQueryProfiler := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ActiveTimersInQueryProfiler"),
		"Number of ActiveTimersInQueryProfiler currently processed",
		nil,
		nil,
	)
	RefreshableViews := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RefreshableViews"),
		"Number of RefreshableViews currently processed",
		nil,
		nil,
	)
	RefreshingViews := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "RefreshingViews"),
		"Number of RefreshingViews currently processed",
		nil,
		nil,
	)
	StorageBufferFlushThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageBufferFlushThreads"),
		"Number of StorageBufferFlushThreads currently processed",
		nil,
		nil,
	)
	StorageBufferFlushThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageBufferFlushThreadsActive"),
		"Number of StorageBufferFlushThreadsActive currently processed",
		nil,
		nil,
	)
	StorageBufferFlushThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageBufferFlushThreadsScheduled"),
		"Number of StorageBufferFlushThreadsScheduled currently processed",
		nil,
		nil,
	)
	SharedMergeTreeThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedMergeTreeThreads"),
		"Number of SharedMergeTreeThreads currently processed",
		nil,
		nil,
	)
	SharedMergeTreeThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedMergeTreeThreadsActive"),
		"Number of SharedMergeTreeThreadsActive currently processed",
		nil,
		nil,
	)
	SharedMergeTreeThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedMergeTreeThreadsScheduled"),
		"Number of SharedMergeTreeThreadsScheduled currently processed",
		nil,
		nil,
	)
	SharedMergeTreeFetch := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedMergeTreeFetch"),
		"Number of SharedMergeTreeFetch currently processed",
		nil,
		nil,
	)
	CacheWarmerBytesInProgress := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "CacheWarmerBytesInProgress"),
		"Number of CacheWarmerBytesInProgress currently processed",
		nil,
		nil,
	)
	DistrCacheOpenedConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheOpenedConnections"),
		"Number of DistrCacheOpenedConnections currently processed",
		nil,
		nil,
	)
	DistrCacheUsedConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheUsedConnections"),
		"Number of DistrCacheUsedConnections currently processed",
		nil,
		nil,
	)
	DistrCacheAllocatedConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheAllocatedConnections"),
		"Number of DistrCacheAllocatedConnections currently processed",
		nil,
		nil,
	)
	DistrCacheBorrowedConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheBorrowedConnections"),
		"Number of DistrCacheBorrowedConnections currently processed",
		nil,
		nil,
	)
	DistrCacheReadRequests := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheReadRequests"),
		"Number of DistrCacheReadRequests currently processed",
		nil,
		nil,
	)
	DistrCacheWriteRequests := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheWriteRequests"),
		"Number of DistrCacheWriteRequests currently processed",
		nil,
		nil,
	)
	DistrCacheServerConnections := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheServerConnections"),
		"Number of DistrCacheServerConnections currently processed",
		nil,
		nil,
	)
	DistrCacheRegisteredServers := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheRegisteredServers"),
		"Number of DistrCacheRegisteredServers currently processed",
		nil,
		nil,
	)
	DistrCacheRegisteredServersCurrentAZ := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheRegisteredServersCurrentAZ"),
		"Number of DistrCacheRegisteredServersCurrentAZ currently processed",
		nil,
		nil,
	)
	DistrCacheServerS3CachedClients := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DistrCacheServerS3CachedClients"),
		"Number of DistrCacheServerS3CachedClients currently processed",
		nil,
		nil,
	)
	SchedulerIOReadScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SchedulerIOReadScheduled"),
		"Number of SchedulerIOReadScheduled currently processed",
		nil,
		nil,
	)
	SchedulerIOWriteScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SchedulerIOWriteScheduled"),
		"Number of SchedulerIOWriteScheduled currently processed",
		nil,
		nil,
	)
	StorageConnectionsStored := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageConnectionsStored"),
		"Number of StorageConnectionsStored currently processed",
		nil,
		nil,
	)
	StorageConnectionsTotal := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StorageConnectionsTotal"),
		"Number of StorageConnectionsTotal currently processed",
		nil,
		nil,
	)
	DiskConnectionsStored := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskConnectionsStored"),
		"Number of DiskConnectionsStored currently processed",
		nil,
		nil,
	)
	DiskConnectionsTotal := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskConnectionsTotal"),
		"Number of DiskConnectionsTotal currently processed",
		nil,
		nil,
	)
	HTTPConnectionsStored := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HTTPConnectionsStored"),
		"Number of HTTPConnectionsStored currently processed",
		nil,
		nil,
	)
	HTTPConnectionsTotal := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "HTTPConnectionsTotal"),
		"Number of HTTPConnectionsTotal currently processed",
		nil,
		nil,
	)
	AddressesActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AddressesActive"),
		"Number of AddressesActive currently processed",
		nil,
		nil,
	)
	AddressesBanned := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "AddressesBanned"),
		"Number of AddressesBanned currently processed",
		nil,
		nil,
	)
	FilteringMarksWithPrimaryKey := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilteringMarksWithPrimaryKey"),
		"Number of FilteringMarksWithPrimaryKey currently processed",
		nil,
		nil,
	)
	FilteringMarksWithSecondaryKeys := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "FilteringMarksWithSecondaryKeys"),
		"Number of FilteringMarksWithSecondaryKeys currently processed",
		nil,
		nil,
	)
	ConcurrencyControlAcquired := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrencyControlAcquired"),
		"Number of ConcurrencyControlAcquired currently processed",
		nil,
		nil,
	)
	ConcurrencyControlSoftLimit := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "ConcurrencyControlSoftLimit"),
		"Number of ConcurrencyControlSoftLimit currently processed",
		nil,
		nil,
	)
	DiskS3NoSuchKeyErrors := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "DiskS3NoSuchKeyErrors"),
		"Number of DiskS3NoSuchKeyErrors currently processed",
		nil,
		nil,
	)
	SharedCatalogStateApplicationThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogStateApplicationThreads"),
		"Number of SharedCatalogStateApplicationThreads currently processed",
		nil,
		nil,
	)
	SharedCatalogStateApplicationThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogStateApplicationThreadsActive"),
		"Number of SharedCatalogStateApplicationThreadsActive currently processed",
		nil,
		nil,
	)
	SharedCatalogStateApplicationThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogStateApplicationThreadsScheduled"),
		"Number of SharedCatalogStateApplicationThreadsScheduled currently processed",
		nil,
		nil,
	)
	SharedCatalogDropLocalThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropLocalThreads"),
		"Number of SharedCatalogDropLocalThreads currently processed",
		nil,
		nil,
	)
	SharedCatalogDropLocalThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropLocalThreadsActive"),
		"Number of SharedCatalogDropLocalThreadsActive currently processed",
		nil,
		nil,
	)
	SharedCatalogDropLocalThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropLocalThreadsScheduled"),
		"Number of SharedCatalogDropLocalThreadsScheduled currently processed",
		nil,
		nil,
	)
	SharedCatalogDropZooKeeperThreads := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropZooKeeperThreads"),
		"Number of SharedCatalogDropZooKeeperThreads currently processed",
		nil,
		nil,
	)
	SharedCatalogDropZooKeeperThreadsActive := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropZooKeeperThreadsActive"),
		"Number of SharedCatalogDropZooKeeperThreadsActive currently processed",
		nil,
		nil,
	)
	SharedCatalogDropZooKeeperThreadsScheduled := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedCatalogDropZooKeeperThreadsScheduled"),
		"Number of SharedCatalogDropZooKeeperThreadsScheduled currently processed",
		nil,
		nil,
	)
	SharedDatabaseCatalogTablesInLocalDropDetachQueue := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "SharedDatabaseCatalogTablesInLocalDropDetachQueue"),
		"Number of SharedDatabaseCatalogTablesInLocalDropDetachQueue currently processed",
		nil,
		nil,
	)
	StartupScriptsExecutionState := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "StartupScriptsExecutionState"),
		"Number of StartupScriptsExecutionState currently processed",
		nil,
		nil,
	)
	IsServerShuttingDown := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "IsServerShuttingDown"),
		"Number of IsServerShuttingDown currently processed",
		nil,
		nil,
	)

	return &MetricsURIExporter{
		Query:                                             Query,
		Merge:                                             Merge,
		MergeParts:                                        MergeParts,
		Move:                                              Move,
		PartMutation:                                      PartMutation,
		ReplicatedFetch:                                   ReplicatedFetch,
		ReplicatedSend:                                    ReplicatedSend,
		ReplicatedChecks:                                  ReplicatedChecks,
		BackgroundMergesAndMutationsPoolTask:              BackgroundMergesAndMutationsPoolTask,
		BackgroundMergesAndMutationsPoolSize:              BackgroundMergesAndMutationsPoolSize,
		BackgroundFetchesPoolTask:                         BackgroundFetchesPoolTask,
		BackgroundFetchesPoolSize:                         BackgroundFetchesPoolSize,
		BackgroundCommonPoolTask:                          BackgroundCommonPoolTask,
		BackgroundCommonPoolSize:                          BackgroundCommonPoolSize,
		BackgroundMovePoolTask:                            BackgroundMovePoolTask,
		BackgroundMovePoolSize:                            BackgroundMovePoolSize,
		BackgroundSchedulePoolTask:                        BackgroundSchedulePoolTask,
		BackgroundSchedulePoolSize:                        BackgroundSchedulePoolSize,
		BackgroundBufferFlushSchedulePoolTask:             BackgroundBufferFlushSchedulePoolTask,
		BackgroundBufferFlushSchedulePoolSize:             BackgroundBufferFlushSchedulePoolSize,
		BackgroundDistributedSchedulePoolTask:             BackgroundDistributedSchedulePoolTask,
		BackgroundDistributedSchedulePoolSize:             BackgroundDistributedSchedulePoolSize,
		BackgroundMessageBrokerSchedulePoolTask:           BackgroundMessageBrokerSchedulePoolTask,
		BackgroundMessageBrokerSchedulePoolSize:           BackgroundMessageBrokerSchedulePoolSize,
		CacheDictionaryUpdateQueueBatches:                 CacheDictionaryUpdateQueueBatches,
		CacheDictionaryUpdateQueueKeys:                    CacheDictionaryUpdateQueueKeys,
		DiskSpaceReservedForMerge:                         DiskSpaceReservedForMerge,
		DistributedSend:                                   DistributedSend,
		QueryPreempted:                                    QueryPreempted,
		TCPConnection:                                     TCPConnection,
		MySQLConnection:                                   MySQLConnection,
		HTTPConnection:                                    HTTPConnection,
		InterserverConnection:                             InterserverConnection,
		PostgreSQLConnection:                              PostgreSQLConnection,
		OpenFileForRead:                                   OpenFileForRead,
		OpenFileForWrite:                                  OpenFileForWrite,
		Compressing:                                       Compressing,
		Decompressing:                                     Decompressing,
		ParallelCompressedWriteBufferThreads:              ParallelCompressedWriteBufferThreads,
		ParallelCompressedWriteBufferWait:                 ParallelCompressedWriteBufferWait,
		TotalTemporaryFiles:                               TotalTemporaryFiles,
		TemporaryFilesForSort:                             TemporaryFilesForSort,
		TemporaryFilesForAggregation:                      TemporaryFilesForAggregation,
		TemporaryFilesForJoin:                             TemporaryFilesForJoin,
		TemporaryFilesForMerge:                            TemporaryFilesForMerge,
		TemporaryFilesUnknown:                             TemporaryFilesUnknown,
		Read:                                              Read,
		RemoteRead:                                        RemoteRead,
		Write:                                             Write,
		NetworkReceive:                                    NetworkReceive,
		NetworkSend:                                       NetworkSend,
		SendScalars:                                       SendScalars,
		SendExternalTables:                                SendExternalTables,
		QueryThread:                                       QueryThread,
		ReadonlyReplica:                                   ReadonlyReplica,
		MemoryTracking:                                    MemoryTracking,
		MemoryTrackingUncorrected:                         MemoryTrackingUncorrected,
		MergesMutationsMemoryTracking:                     MergesMutationsMemoryTracking,
		EphemeralNode:                                     EphemeralNode,
		ZooKeeperSession:                                  ZooKeeperSession,
		ZooKeeperWatch:                                    ZooKeeperWatch,
		ZooKeeperRequest:                                  ZooKeeperRequest,
		DelayedInserts:                                    DelayedInserts,
		ContextLockWait:                                   ContextLockWait,
		StorageBufferRows:                                 StorageBufferRows,
		StorageBufferBytes:                                StorageBufferBytes,
		DictCacheRequests:                                 DictCacheRequests,
		Revision:                                          Revision,
		VersionInteger:                                    VersionInteger,
		RWLockWaitingReaders:                              RWLockWaitingReaders,
		RWLockWaitingWriters:                              RWLockWaitingWriters,
		RWLockActiveReaders:                               RWLockActiveReaders,
		RWLockActiveWriters:                               RWLockActiveWriters,
		GlobalThread:                                      GlobalThread,
		GlobalThreadActive:                                GlobalThreadActive,
		GlobalThreadScheduled:                             GlobalThreadScheduled,
		LocalThread:                                       LocalThread,
		LocalThreadActive:                                 LocalThreadActive,
		LocalThreadScheduled:                              LocalThreadScheduled,
		MergeTreeDataSelectExecutorThreads:                MergeTreeDataSelectExecutorThreads,
		MergeTreeDataSelectExecutorThreadsActive:          MergeTreeDataSelectExecutorThreadsActive,
		MergeTreeDataSelectExecutorThreadsScheduled:       MergeTreeDataSelectExecutorThreadsScheduled,
		BackupsThreads:                                    BackupsThreads,
		BackupsThreadsActive:                              BackupsThreadsActive,
		BackupsThreadsScheduled:                           BackupsThreadsScheduled,
		RestoreThreads:                                    RestoreThreads,
		RestoreThreadsActive:                              RestoreThreadsActive,
		RestoreThreadsScheduled:                           RestoreThreadsScheduled,
		MarksLoaderThreads:                                MarksLoaderThreads,
		MarksLoaderThreadsActive:                          MarksLoaderThreadsActive,
		MarksLoaderThreadsScheduled:                       MarksLoaderThreadsScheduled,
		IOPrefetchThreads:                                 IOPrefetchThreads,
		IOPrefetchThreadsActive:                           IOPrefetchThreadsActive,
		IOPrefetchThreadsScheduled:                        IOPrefetchThreadsScheduled,
		IOWriterThreads:                                   IOWriterThreads,
		IOWriterThreadsActive:                             IOWriterThreadsActive,
		IOWriterThreadsScheduled:                          IOWriterThreadsScheduled,
		IOThreads:                                         IOThreads,
		IOThreadsActive:                                   IOThreadsActive,
		IOThreadsScheduled:                                IOThreadsScheduled,
		CompressionThread:                                 CompressionThread,
		CompressionThreadActive:                           CompressionThreadActive,
		CompressionThreadScheduled:                        CompressionThreadScheduled,
		ThreadPoolRemoteFSReaderThreads:                   ThreadPoolRemoteFSReaderThreads,
		ThreadPoolRemoteFSReaderThreadsActive:             ThreadPoolRemoteFSReaderThreadsActive,
		ThreadPoolRemoteFSReaderThreadsScheduled:          ThreadPoolRemoteFSReaderThreadsScheduled,
		ThreadPoolFSReaderThreads:                         ThreadPoolFSReaderThreads,
		ThreadPoolFSReaderThreadsActive:                   ThreadPoolFSReaderThreadsActive,
		ThreadPoolFSReaderThreadsScheduled:                ThreadPoolFSReaderThreadsScheduled,
		BackupsIOThreads:                                  BackupsIOThreads,
		BackupsIOThreadsActive:                            BackupsIOThreadsActive,
		BackupsIOThreadsScheduled:                         BackupsIOThreadsScheduled,
		DiskObjectStorageAsyncThreads:                     DiskObjectStorageAsyncThreads,
		DiskObjectStorageAsyncThreadsActive:               DiskObjectStorageAsyncThreadsActive,
		StorageHiveThreads:                                StorageHiveThreads,
		StorageHiveThreadsActive:                          StorageHiveThreadsActive,
		StorageHiveThreadsScheduled:                       StorageHiveThreadsScheduled,
		TablesLoaderBackgroundThreads:                     TablesLoaderBackgroundThreads,
		TablesLoaderBackgroundThreadsActive:               TablesLoaderBackgroundThreadsActive,
		TablesLoaderBackgroundThreadsScheduled:            TablesLoaderBackgroundThreadsScheduled,
		TablesLoaderForegroundThreads:                     TablesLoaderForegroundThreads,
		TablesLoaderForegroundThreadsActive:               TablesLoaderForegroundThreadsActive,
		TablesLoaderForegroundThreadsScheduled:            TablesLoaderForegroundThreadsScheduled,
		DatabaseOnDiskThreads:                             DatabaseOnDiskThreads,
		DatabaseOnDiskThreadsActive:                       DatabaseOnDiskThreadsActive,
		DatabaseOnDiskThreadsScheduled:                    DatabaseOnDiskThreadsScheduled,
		DatabaseBackupThreads:                             DatabaseBackupThreads,
		DatabaseBackupThreadsActive:                       DatabaseBackupThreadsActive,
		DatabaseBackupThreadsScheduled:                    DatabaseBackupThreadsScheduled,
		DatabaseCatalogThreads:                            DatabaseCatalogThreads,
		DatabaseCatalogThreadsActive:                      DatabaseCatalogThreadsActive,
		DatabaseCatalogThreadsScheduled:                   DatabaseCatalogThreadsScheduled,
		DestroyAggregatesThreads:                          DestroyAggregatesThreads,
		DestroyAggregatesThreadsActive:                    DestroyAggregatesThreadsActive,
		DestroyAggregatesThreadsScheduled:                 DestroyAggregatesThreadsScheduled,
		ConcurrentHashJoinPoolThreads:                     ConcurrentHashJoinPoolThreads,
		ConcurrentHashJoinPoolThreadsActive:               ConcurrentHashJoinPoolThreadsActive,
		ConcurrentHashJoinPoolThreadsScheduled:            ConcurrentHashJoinPoolThreadsScheduled,
		HashedDictionaryThreads:                           HashedDictionaryThreads,
		HashedDictionaryThreadsActive:                     HashedDictionaryThreadsActive,
		HashedDictionaryThreadsScheduled:                  HashedDictionaryThreadsScheduled,
		CacheDictionaryThreads:                            CacheDictionaryThreads,
		CacheDictionaryThreadsActive:                      CacheDictionaryThreadsActive,
		CacheDictionaryThreadsScheduled:                   CacheDictionaryThreadsScheduled,
		ParallelFormattingOutputFormatThreads:             ParallelFormattingOutputFormatThreads,
		ParallelFormattingOutputFormatThreadsActive:       ParallelFormattingOutputFormatThreadsActive,
		ParallelFormattingOutputFormatThreadsScheduled:    ParallelFormattingOutputFormatThreadsScheduled,
		ParallelParsingInputFormatThreads:                 ParallelParsingInputFormatThreads,
		ParallelParsingInputFormatThreadsActive:           ParallelParsingInputFormatThreadsActive,
		ParallelParsingInputFormatThreadsScheduled:        ParallelParsingInputFormatThreadsScheduled,
		MergeTreeBackgroundExecutorThreads:                MergeTreeBackgroundExecutorThreads,
		MergeTreeBackgroundExecutorThreadsActive:          MergeTreeBackgroundExecutorThreadsActive,
		MergeTreeBackgroundExecutorThreadsScheduled:       MergeTreeBackgroundExecutorThreadsScheduled,
		AsynchronousInsertThreads:                         AsynchronousInsertThreads,
		AsynchronousInsertThreadsActive:                   AsynchronousInsertThreadsActive,
		AsynchronousInsertThreadsScheduled:                AsynchronousInsertThreadsScheduled,
		AsynchronousInsertQueueSize:                       AsynchronousInsertQueueSize,
		AsynchronousInsertQueueBytes:                      AsynchronousInsertQueueBytes,
		StartupSystemTablesThreads:                        StartupSystemTablesThreads,
		StartupSystemTablesThreadsActive:                  StartupSystemTablesThreadsActive,
		StartupSystemTablesThreadsScheduled:               StartupSystemTablesThreadsScheduled,
		AggregatorThreads:                                 AggregatorThreads,
		AggregatorThreadsActive:                           AggregatorThreadsActive,
		AggregatorThreadsScheduled:                        AggregatorThreadsScheduled,
		DDLWorkerThreads:                                  DDLWorkerThreads,
		DDLWorkerThreadsActive:                            DDLWorkerThreadsActive,
		DDLWorkerThreadsScheduled:                         DDLWorkerThreadsScheduled,
		StorageDistributedThreads:                         StorageDistributedThreads,
		StorageDistributedThreadsActive:                   StorageDistributedThreadsActive,
		StorageDistributedThreadsScheduled:                StorageDistributedThreadsScheduled,
		DistributedInsertThreads:                          DistributedInsertThreads,
		DistributedInsertThreadsActive:                    DistributedInsertThreadsActive,
		DistributedInsertThreadsScheduled:                 DistributedInsertThreadsScheduled,
		StorageS3Threads:                                  StorageS3Threads,
		StorageS3ThreadsActive:                            StorageS3ThreadsActive,
		StorageS3ThreadsScheduled:                         StorageS3ThreadsScheduled,
		ObjectStorageS3Threads:                            ObjectStorageS3Threads,
		ObjectStorageS3ThreadsActive:                      ObjectStorageS3ThreadsActive,
		ObjectStorageS3ThreadsScheduled:                   ObjectStorageS3ThreadsScheduled,
		StorageObjectStorageThreads:                       StorageObjectStorageThreads,
		StorageObjectStorageThreadsActive:                 StorageObjectStorageThreadsActive,
		StorageObjectStorageThreadsScheduled:              StorageObjectStorageThreadsScheduled,
		ObjectStorageAzureThreads:                         ObjectStorageAzureThreads,
		ObjectStorageAzureThreadsActive:                   ObjectStorageAzureThreadsActive,
		ObjectStorageAzureThreadsScheduled:                ObjectStorageAzureThreadsScheduled,
		BuildVectorSimilarityIndexThreads:                 BuildVectorSimilarityIndexThreads,
		BuildVectorSimilarityIndexThreadsActive:           BuildVectorSimilarityIndexThreadsActive,
		BuildVectorSimilarityIndexThreadsScheduled:        BuildVectorSimilarityIndexThreadsScheduled,
		ObjectStorageQueueRegisteredServers:               ObjectStorageQueueRegisteredServers,
		IcebergCatalogThreads:                             IcebergCatalogThreads,
		IcebergCatalogThreadsActive:                       IcebergCatalogThreadsActive,
		IcebergCatalogThreadsScheduled:                    IcebergCatalogThreadsScheduled,
		ParallelWithQueryThreads:                          ParallelWithQueryThreads,
		ParallelWithQueryActiveThreads:                    ParallelWithQueryActiveThreads,
		ParallelWithQueryScheduledThreads:                 ParallelWithQueryScheduledThreads,
		DiskPlainRewritableAzureDirectoryMapSize:          DiskPlainRewritableAzureDirectoryMapSize,
		DiskPlainRewritableAzureFileCount:                 DiskPlainRewritableAzureFileCount,
		DiskPlainRewritableAzureUniqueFileNamesCount:      DiskPlainRewritableAzureUniqueFileNamesCount,
		DiskPlainRewritableLocalDirectoryMapSize:          DiskPlainRewritableLocalDirectoryMapSize,
		DiskPlainRewritableLocalFileCount:                 DiskPlainRewritableLocalFileCount,
		DiskPlainRewritableLocalUniqueFileNamesCount:      DiskPlainRewritableLocalUniqueFileNamesCount,
		DiskPlainRewritableS3DirectoryMapSize:             DiskPlainRewritableS3DirectoryMapSize,
		DiskPlainRewritableS3FileCount:                    DiskPlainRewritableS3FileCount,
		DiskPlainRewritableS3UniqueFileNamesCount:         DiskPlainRewritableS3UniqueFileNamesCount,
		MergeTreeFetchPartitionThreads:                    MergeTreeFetchPartitionThreads,
		MergeTreeFetchPartitionThreadsActive:              MergeTreeFetchPartitionThreadsActive,
		MergeTreeFetchPartitionThreadsScheduled:           MergeTreeFetchPartitionThreadsScheduled,
		MergeTreePartsLoaderThreads:                       MergeTreePartsLoaderThreads,
		MergeTreePartsLoaderThreadsActive:                 MergeTreePartsLoaderThreadsActive,
		MergeTreePartsLoaderThreadsScheduled:              MergeTreePartsLoaderThreadsScheduled,
		MergeTreeOutdatedPartsLoaderThreads:               MergeTreeOutdatedPartsLoaderThreads,
		MergeTreeOutdatedPartsLoaderThreadsActive:         MergeTreeOutdatedPartsLoaderThreadsActive,
		MergeTreeOutdatedPartsLoaderThreadsScheduled:      MergeTreeOutdatedPartsLoaderThreadsScheduled,
		MergeTreeUnexpectedPartsLoaderThreads:             MergeTreeUnexpectedPartsLoaderThreads,
		MergeTreeUnexpectedPartsLoaderThreadsActive:       MergeTreeUnexpectedPartsLoaderThreadsActive,
		MergeTreeUnexpectedPartsLoaderThreadsScheduled:    MergeTreeUnexpectedPartsLoaderThreadsScheduled,
		MergeTreePartsCleanerThreads:                      MergeTreePartsCleanerThreads,
		MergeTreePartsCleanerThreadsActive:                MergeTreePartsCleanerThreadsActive,
		MergeTreePartsCleanerThreadsScheduled:             MergeTreePartsCleanerThreadsScheduled,
		DatabaseReplicatedCreateTablesThreads:             DatabaseReplicatedCreateTablesThreads,
		DatabaseReplicatedCreateTablesThreadsActive:       DatabaseReplicatedCreateTablesThreadsActive,
		DatabaseReplicatedCreateTablesThreadsScheduled:    DatabaseReplicatedCreateTablesThreadsScheduled,
		IDiskCopierThreads:                                IDiskCopierThreads,
		IDiskCopierThreadsActive:                          IDiskCopierThreadsActive,
		IDiskCopierThreadsScheduled:                       IDiskCopierThreadsScheduled,
		SystemReplicasThreads:                             SystemReplicasThreads,
		SystemReplicasThreadsActive:                       SystemReplicasThreadsActive,
		SystemReplicasThreadsScheduled:                    SystemReplicasThreadsScheduled,
		RestartReplicaThreads:                             RestartReplicaThreads,
		RestartReplicaThreadsActive:                       RestartReplicaThreadsActive,
		RestartReplicaThreadsScheduled:                    RestartReplicaThreadsScheduled,
		QueryPipelineExecutorThreads:                      QueryPipelineExecutorThreads,
		QueryPipelineExecutorThreadsActive:                QueryPipelineExecutorThreadsActive,
		QueryPipelineExecutorThreadsScheduled:             QueryPipelineExecutorThreadsScheduled,
		ParquetDecoderThreads:                             ParquetDecoderThreads,
		ParquetDecoderThreadsActive:                       ParquetDecoderThreadsActive,
		ParquetDecoderThreadsScheduled:                    ParquetDecoderThreadsScheduled,
		ParquetDecoderIOThreads:                           ParquetDecoderIOThreads,
		ParquetDecoderIOThreadsActive:                     ParquetDecoderIOThreadsActive,
		ParquetDecoderIOThreadsScheduled:                  ParquetDecoderIOThreadsScheduled,
		ParquetEncoderThreads:                             ParquetEncoderThreads,
		ParquetEncoderThreadsActive:                       ParquetEncoderThreadsActive,
		ParquetEncoderThreadsScheduled:                    ParquetEncoderThreadsScheduled,
		MergeTreeSubcolumnsReaderThreads:                  MergeTreeSubcolumnsReaderThreads,
		MergeTreeSubcolumnsReaderThreadsActive:            MergeTreeSubcolumnsReaderThreadsActive,
		MergeTreeSubcolumnsReaderThreadsScheduled:         MergeTreeSubcolumnsReaderThreadsScheduled,
		DWARFReaderThreads:                                DWARFReaderThreads,
		DWARFReaderThreadsActive:                          DWARFReaderThreadsActive,
		DWARFReaderThreadsScheduled:                       DWARFReaderThreadsScheduled,
		OutdatedPartsLoadingThreads:                       OutdatedPartsLoadingThreads,
		OutdatedPartsLoadingThreadsActive:                 OutdatedPartsLoadingThreadsActive,
		OutdatedPartsLoadingThreadsScheduled:              OutdatedPartsLoadingThreadsScheduled,
		PolygonDictionaryThreads:                          PolygonDictionaryThreads,
		PolygonDictionaryThreadsActive:                    PolygonDictionaryThreadsActive,
		PolygonDictionaryThreadsScheduled:                 PolygonDictionaryThreadsScheduled,
		DistributedBytesToInsert:                          DistributedBytesToInsert,
		BrokenDistributedBytesToInsert:                    BrokenDistributedBytesToInsert,
		DistributedFilesToInsert:                          DistributedFilesToInsert,
		BrokenDistributedFilesToInsert:                    BrokenDistributedFilesToInsert,
		TablesToDropQueueSize:                             TablesToDropQueueSize,
		MaxDDLEntryID:                                     MaxDDLEntryID,
		MaxPushedDDLEntryID:                               MaxPushedDDLEntryID,
		PartsTemporary:                                    PartsTemporary,
		PartsPreCommitted:                                 PartsPreCommitted,
		PartsCommitted:                                    PartsCommitted,
		PartsPreActive:                                    PartsPreActive,
		PartsActive:                                       PartsActive,
		AttachedDatabase:                                  AttachedDatabase,
		AttachedTable:                                     AttachedTable,
		AttachedReplicatedTable:                           AttachedReplicatedTable,
		AttachedView:                                      AttachedView,
		AttachedDictionary:                                AttachedDictionary,
		PartsOutdated:                                     PartsOutdated,
		PartsDeleting:                                     PartsDeleting,
		PartsDeleteOnDestroy:                              PartsDeleteOnDestroy,
		PartsWide:                                         PartsWide,
		PartsCompact:                                      PartsCompact,
		MMappedFiles:                                      MMappedFiles,
		MMappedFileBytes:                                  MMappedFileBytes,
		AsynchronousReadWait:                              AsynchronousReadWait,
		PendingAsyncInsert:                                PendingAsyncInsert,
		KafkaConsumers:                                    KafkaConsumers,
		KafkaConsumersWithAssignment:                      KafkaConsumersWithAssignment,
		KafkaProducers:                                    KafkaProducers,
		KafkaLibrdkafkaThreads:                            KafkaLibrdkafkaThreads,
		KafkaBackgroundReads:                              KafkaBackgroundReads,
		KafkaConsumersInUse:                               KafkaConsumersInUse,
		KafkaWrites:                                       KafkaWrites,
		KafkaAssignedPartitions:                           KafkaAssignedPartitions,
		FilesystemCacheReadBuffers:                        FilesystemCacheReadBuffers,
		CacheFileSegments:                                 CacheFileSegments,
		CacheDetachedFileSegments:                         CacheDetachedFileSegments,
		FilesystemCacheSize:                               FilesystemCacheSize,
		FilesystemCacheSizeLimit:                          FilesystemCacheSizeLimit,
		FilesystemCacheElements:                           FilesystemCacheElements,
		FilesystemCacheDownloadQueueElements:              FilesystemCacheDownloadQueueElements,
		FilesystemCacheDelayedCleanupElements:             FilesystemCacheDelayedCleanupElements,
		FilesystemCacheHoldFileSegments:                   FilesystemCacheHoldFileSegments,
		AsyncInsertCacheSize:                              AsyncInsertCacheSize,
		SkippingIndexCacheSize:                            SkippingIndexCacheSize,
		S3Requests:                                        S3Requests,
		KeeperAliveConnections:                            KeeperAliveConnections,
		KeeperOutstandingRequests:                         KeeperOutstandingRequests,
		ThreadsInOvercommitTracker:                        ThreadsInOvercommitTracker,
		IOUringPendingEvents:                              IOUringPendingEvents,
		IOUringInFlightEvents:                             IOUringInFlightEvents,
		ReadTaskRequestsSent:                              ReadTaskRequestsSent,
		MergeTreeReadTaskRequestsSent:                     MergeTreeReadTaskRequestsSent,
		MergeTreeAllRangesAnnouncementsSent:               MergeTreeAllRangesAnnouncementsSent,
		CreatedTimersInQueryProfiler:                      CreatedTimersInQueryProfiler,
		ActiveTimersInQueryProfiler:                       ActiveTimersInQueryProfiler,
		RefreshableViews:                                  RefreshableViews,
		RefreshingViews:                                   RefreshingViews,
		StorageBufferFlushThreads:                         StorageBufferFlushThreads,
		StorageBufferFlushThreadsActive:                   StorageBufferFlushThreadsActive,
		StorageBufferFlushThreadsScheduled:                StorageBufferFlushThreadsScheduled,
		SharedMergeTreeThreads:                            SharedMergeTreeThreads,
		SharedMergeTreeThreadsActive:                      SharedMergeTreeThreadsActive,
		SharedMergeTreeThreadsScheduled:                   SharedMergeTreeThreadsScheduled,
		SharedMergeTreeFetch:                              SharedMergeTreeFetch,
		CacheWarmerBytesInProgress:                        CacheWarmerBytesInProgress,
		DistrCacheOpenedConnections:                       DistrCacheOpenedConnections,
		DistrCacheUsedConnections:                         DistrCacheUsedConnections,
		DistrCacheAllocatedConnections:                    DistrCacheAllocatedConnections,
		DistrCacheBorrowedConnections:                     DistrCacheBorrowedConnections,
		DistrCacheReadRequests:                            DistrCacheReadRequests,
		DistrCacheWriteRequests:                           DistrCacheWriteRequests,
		DistrCacheServerConnections:                       DistrCacheServerConnections,
		DistrCacheRegisteredServers:                       DistrCacheRegisteredServers,
		DistrCacheRegisteredServersCurrentAZ:              DistrCacheRegisteredServersCurrentAZ,
		DistrCacheServerS3CachedClients:                   DistrCacheServerS3CachedClients,
		SchedulerIOReadScheduled:                          SchedulerIOReadScheduled,
		SchedulerIOWriteScheduled:                         SchedulerIOWriteScheduled,
		StorageConnectionsStored:                          StorageConnectionsStored,
		StorageConnectionsTotal:                           StorageConnectionsTotal,
		DiskConnectionsStored:                             DiskConnectionsStored,
		DiskConnectionsTotal:                              DiskConnectionsTotal,
		HTTPConnectionsStored:                             HTTPConnectionsStored,
		HTTPConnectionsTotal:                              HTTPConnectionsTotal,
		AddressesActive:                                   AddressesActive,
		AddressesBanned:                                   AddressesBanned,
		FilteringMarksWithPrimaryKey:                      FilteringMarksWithPrimaryKey,
		FilteringMarksWithSecondaryKeys:                   FilteringMarksWithSecondaryKeys,
		ConcurrencyControlAcquired:                        ConcurrencyControlAcquired,
		ConcurrencyControlSoftLimit:                       ConcurrencyControlSoftLimit,
		DiskS3NoSuchKeyErrors:                             DiskS3NoSuchKeyErrors,
		SharedCatalogStateApplicationThreads:              SharedCatalogStateApplicationThreads,
		SharedCatalogStateApplicationThreadsActive:        SharedCatalogStateApplicationThreadsActive,
		SharedCatalogStateApplicationThreadsScheduled:     SharedCatalogStateApplicationThreadsScheduled,
		SharedCatalogDropLocalThreads:                     SharedCatalogDropLocalThreads,
		SharedCatalogDropLocalThreadsActive:               SharedCatalogDropLocalThreadsActive,
		SharedCatalogDropLocalThreadsScheduled:            SharedCatalogDropLocalThreadsScheduled,
		SharedCatalogDropZooKeeperThreads:                 SharedCatalogDropZooKeeperThreads,
		SharedCatalogDropZooKeeperThreadsActive:           SharedCatalogDropZooKeeperThreadsActive,
		SharedCatalogDropZooKeeperThreadsScheduled:        SharedCatalogDropZooKeeperThreadsScheduled,
		SharedDatabaseCatalogTablesInLocalDropDetachQueue: SharedDatabaseCatalogTablesInLocalDropDetachQueue,
		StartupScriptsExecutionState:                      StartupScriptsExecutionState,
		IsServerShuttingDown:                              IsServerShuttingDown,
	}
}

func (e *MetricsURIExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.Query
}

func (e *MetricsURIExporter) Collect(ch chan<- prometheus.Metric) {
	// log.Println("run here Collect")

	upValue := 1

	if err := e.collect(ch); err != nil {
		log.Info().Msgf("Error scraping clickhouse: %s", err)
		upValue = 0
	}

	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Was the last query of ClickHouse successful.",
			nil, nil,
		),
		prometheus.GaugeValue, float64(upValue),
	)
}
func (e *MetricsURIExporter) collect(ch chan<- prometheus.Metric) error {
	mu, err := url.Parse(URI)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	q := mu.Query()
	metricsURI := mu
	q.Set("query", "select metric, value from system.metrics")
	metricsURI.RawQuery = q.Encode()

	metrics, err := e.parseMetricsURIResponse(metricsURI.String())
	if err != nil {
		return fmt.Errorf("error scraping clickhouse url %v: %v", metricsURI.String(), err)
	}

	ch <- prometheus.MustNewConstMetric(
		e.Query,
		prometheus.GaugeValue,
		float64(metrics.Query),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Merge,
		prometheus.GaugeValue,
		float64(metrics.Merge),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeParts,
		prometheus.GaugeValue,
		float64(metrics.MergeParts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Move,
		prometheus.GaugeValue,
		float64(metrics.Move),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartMutation,
		prometheus.GaugeValue,
		float64(metrics.PartMutation),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReplicatedFetch,
		prometheus.GaugeValue,
		float64(metrics.ReplicatedFetch),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReplicatedSend,
		prometheus.GaugeValue,
		float64(metrics.ReplicatedSend),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReplicatedChecks,
		prometheus.GaugeValue,
		float64(metrics.ReplicatedChecks),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMergesAndMutationsPoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMergesAndMutationsPoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMergesAndMutationsPoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMergesAndMutationsPoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundFetchesPoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundFetchesPoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundFetchesPoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundFetchesPoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundCommonPoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundCommonPoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundCommonPoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundCommonPoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMovePoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMovePoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMovePoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMovePoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundSchedulePoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundSchedulePoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundSchedulePoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundSchedulePoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundBufferFlushSchedulePoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundBufferFlushSchedulePoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundBufferFlushSchedulePoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundBufferFlushSchedulePoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundDistributedSchedulePoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundDistributedSchedulePoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundDistributedSchedulePoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundDistributedSchedulePoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMessageBrokerSchedulePoolTask,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMessageBrokerSchedulePoolTask),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackgroundMessageBrokerSchedulePoolSize,
		prometheus.GaugeValue,
		float64(metrics.BackgroundMessageBrokerSchedulePoolSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDictionaryUpdateQueueBatches,
		prometheus.GaugeValue,
		float64(metrics.CacheDictionaryUpdateQueueBatches),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDictionaryUpdateQueueKeys,
		prometheus.GaugeValue,
		float64(metrics.CacheDictionaryUpdateQueueKeys),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskSpaceReservedForMerge,
		prometheus.GaugeValue,
		float64(metrics.DiskSpaceReservedForMerge),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedSend,
		prometheus.GaugeValue,
		float64(metrics.DistributedSend),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryPreempted,
		prometheus.GaugeValue,
		float64(metrics.QueryPreempted),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TCPConnection,
		prometheus.GaugeValue,
		float64(metrics.TCPConnection),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MySQLConnection,
		prometheus.GaugeValue,
		float64(metrics.MySQLConnection),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HTTPConnection,
		prometheus.GaugeValue,
		float64(metrics.HTTPConnection),
	)
	ch <- prometheus.MustNewConstMetric(
		e.InterserverConnection,
		prometheus.GaugeValue,
		float64(metrics.InterserverConnection),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PostgreSQLConnection,
		prometheus.GaugeValue,
		float64(metrics.PostgreSQLConnection),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OpenFileForRead,
		prometheus.GaugeValue,
		float64(metrics.OpenFileForRead),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OpenFileForWrite,
		prometheus.GaugeValue,
		float64(metrics.OpenFileForWrite),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Compressing,
		prometheus.GaugeValue,
		float64(metrics.Compressing),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Decompressing,
		prometheus.GaugeValue,
		float64(metrics.Decompressing),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelCompressedWriteBufferThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelCompressedWriteBufferThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelCompressedWriteBufferWait,
		prometheus.GaugeValue,
		float64(metrics.ParallelCompressedWriteBufferWait),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TotalTemporaryFiles,
		prometheus.GaugeValue,
		float64(metrics.TotalTemporaryFiles),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TemporaryFilesForSort,
		prometheus.GaugeValue,
		float64(metrics.TemporaryFilesForSort),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TemporaryFilesForAggregation,
		prometheus.GaugeValue,
		float64(metrics.TemporaryFilesForAggregation),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TemporaryFilesForJoin,
		prometheus.GaugeValue,
		float64(metrics.TemporaryFilesForJoin),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TemporaryFilesForMerge,
		prometheus.GaugeValue,
		float64(metrics.TemporaryFilesForMerge),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TemporaryFilesUnknown,
		prometheus.GaugeValue,
		float64(metrics.TemporaryFilesUnknown),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Read,
		prometheus.GaugeValue,
		float64(metrics.Read),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RemoteRead,
		prometheus.GaugeValue,
		float64(metrics.RemoteRead),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Write,
		prometheus.GaugeValue,
		float64(metrics.Write),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkReceive,
		prometheus.GaugeValue,
		float64(metrics.NetworkReceive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.NetworkSend,
		prometheus.GaugeValue,
		float64(metrics.NetworkSend),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SendScalars,
		prometheus.GaugeValue,
		float64(metrics.SendScalars),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SendExternalTables,
		prometheus.GaugeValue,
		float64(metrics.SendExternalTables),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryThread,
		prometheus.GaugeValue,
		float64(metrics.QueryThread),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReadonlyReplica,
		prometheus.GaugeValue,
		float64(metrics.ReadonlyReplica),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MemoryTracking,
		prometheus.GaugeValue,
		float64(metrics.MemoryTracking),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MemoryTrackingUncorrected,
		prometheus.GaugeValue,
		float64(metrics.MemoryTrackingUncorrected),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergesMutationsMemoryTracking,
		prometheus.GaugeValue,
		float64(metrics.MergesMutationsMemoryTracking),
	)
	ch <- prometheus.MustNewConstMetric(
		e.EphemeralNode,
		prometheus.GaugeValue,
		float64(metrics.EphemeralNode),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ZooKeeperSession,
		prometheus.GaugeValue,
		float64(metrics.ZooKeeperSession),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ZooKeeperWatch,
		prometheus.GaugeValue,
		float64(metrics.ZooKeeperWatch),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ZooKeeperRequest,
		prometheus.GaugeValue,
		float64(metrics.ZooKeeperRequest),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DelayedInserts,
		prometheus.GaugeValue,
		float64(metrics.DelayedInserts),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ContextLockWait,
		prometheus.GaugeValue,
		float64(metrics.ContextLockWait),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageBufferRows,
		prometheus.GaugeValue,
		float64(metrics.StorageBufferRows),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageBufferBytes,
		prometheus.GaugeValue,
		float64(metrics.StorageBufferBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DictCacheRequests,
		prometheus.GaugeValue,
		float64(metrics.DictCacheRequests),
	)
	ch <- prometheus.MustNewConstMetric(
		e.Revision,
		prometheus.GaugeValue,
		float64(metrics.Revision),
	)
	ch <- prometheus.MustNewConstMetric(
		e.VersionInteger,
		prometheus.GaugeValue,
		float64(metrics.VersionInteger),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockWaitingReaders,
		prometheus.GaugeValue,
		float64(metrics.RWLockWaitingReaders),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockWaitingWriters,
		prometheus.GaugeValue,
		float64(metrics.RWLockWaitingWriters),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockActiveReaders,
		prometheus.GaugeValue,
		float64(metrics.RWLockActiveReaders),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RWLockActiveWriters,
		prometheus.GaugeValue,
		float64(metrics.RWLockActiveWriters),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThread,
		prometheus.GaugeValue,
		float64(metrics.GlobalThread),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadActive,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.GlobalThreadScheduled,
		prometheus.GaugeValue,
		float64(metrics.GlobalThreadScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThread,
		prometheus.GaugeValue,
		float64(metrics.LocalThread),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadActive,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.LocalThreadScheduled,
		prometheus.GaugeValue,
		float64(metrics.LocalThreadScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataSelectExecutorThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataSelectExecutorThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataSelectExecutorThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataSelectExecutorThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeDataSelectExecutorThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeDataSelectExecutorThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsThreads,
		prometheus.GaugeValue,
		float64(metrics.BackupsThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.BackupsThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.BackupsThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestoreThreads,
		prometheus.GaugeValue,
		float64(metrics.RestoreThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestoreThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.RestoreThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestoreThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.RestoreThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MarksLoaderThreads,
		prometheus.GaugeValue,
		float64(metrics.MarksLoaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MarksLoaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MarksLoaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MarksLoaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MarksLoaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOPrefetchThreads,
		prometheus.GaugeValue,
		float64(metrics.IOPrefetchThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOPrefetchThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.IOPrefetchThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOPrefetchThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.IOPrefetchThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOWriterThreads,
		prometheus.GaugeValue,
		float64(metrics.IOWriterThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOWriterThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.IOWriterThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOWriterThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.IOWriterThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOThreads,
		prometheus.GaugeValue,
		float64(metrics.IOThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.IOThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.IOThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompressionThread,
		prometheus.GaugeValue,
		float64(metrics.CompressionThread),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompressionThreadActive,
		prometheus.GaugeValue,
		float64(metrics.CompressionThreadActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CompressionThreadScheduled,
		prometheus.GaugeValue,
		float64(metrics.CompressionThreadScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolRemoteFSReaderThreads,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolRemoteFSReaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolRemoteFSReaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolRemoteFSReaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolRemoteFSReaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolRemoteFSReaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolFSReaderThreads,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolFSReaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolFSReaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolFSReaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadPoolFSReaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ThreadPoolFSReaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsIOThreads,
		prometheus.GaugeValue,
		float64(metrics.BackupsIOThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsIOThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.BackupsIOThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BackupsIOThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.BackupsIOThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskObjectStorageAsyncThreads,
		prometheus.GaugeValue,
		float64(metrics.DiskObjectStorageAsyncThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskObjectStorageAsyncThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DiskObjectStorageAsyncThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageHiveThreads,
		prometheus.GaugeValue,
		float64(metrics.StorageHiveThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageHiveThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StorageHiveThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageHiveThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StorageHiveThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderBackgroundThreads,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderBackgroundThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderBackgroundThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderBackgroundThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderBackgroundThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderBackgroundThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderForegroundThreads,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderForegroundThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderForegroundThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderForegroundThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesLoaderForegroundThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.TablesLoaderForegroundThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseOnDiskThreads,
		prometheus.GaugeValue,
		float64(metrics.DatabaseOnDiskThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseOnDiskThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DatabaseOnDiskThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseOnDiskThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DatabaseOnDiskThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseBackupThreads,
		prometheus.GaugeValue,
		float64(metrics.DatabaseBackupThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseBackupThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DatabaseBackupThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseBackupThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DatabaseBackupThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseCatalogThreads,
		prometheus.GaugeValue,
		float64(metrics.DatabaseCatalogThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseCatalogThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DatabaseCatalogThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseCatalogThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DatabaseCatalogThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DestroyAggregatesThreads,
		prometheus.GaugeValue,
		float64(metrics.DestroyAggregatesThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DestroyAggregatesThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DestroyAggregatesThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DestroyAggregatesThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DestroyAggregatesThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrentHashJoinPoolThreads,
		prometheus.GaugeValue,
		float64(metrics.ConcurrentHashJoinPoolThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrentHashJoinPoolThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ConcurrentHashJoinPoolThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrentHashJoinPoolThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ConcurrentHashJoinPoolThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HashedDictionaryThreads,
		prometheus.GaugeValue,
		float64(metrics.HashedDictionaryThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HashedDictionaryThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.HashedDictionaryThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HashedDictionaryThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.HashedDictionaryThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDictionaryThreads,
		prometheus.GaugeValue,
		float64(metrics.CacheDictionaryThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDictionaryThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.CacheDictionaryThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDictionaryThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.CacheDictionaryThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelFormattingOutputFormatThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelFormattingOutputFormatThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelFormattingOutputFormatThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ParallelFormattingOutputFormatThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelFormattingOutputFormatThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ParallelFormattingOutputFormatThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelParsingInputFormatThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelParsingInputFormatThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelParsingInputFormatThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ParallelParsingInputFormatThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelParsingInputFormatThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ParallelParsingInputFormatThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeBackgroundExecutorThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeBackgroundExecutorThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeBackgroundExecutorThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeBackgroundExecutorThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeBackgroundExecutorThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeBackgroundExecutorThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousInsertThreads,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousInsertThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousInsertThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousInsertThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousInsertThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousInsertThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousInsertQueueSize,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousInsertQueueSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousInsertQueueBytes,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousInsertQueueBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StartupSystemTablesThreads,
		prometheus.GaugeValue,
		float64(metrics.StartupSystemTablesThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StartupSystemTablesThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StartupSystemTablesThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StartupSystemTablesThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StartupSystemTablesThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AggregatorThreads,
		prometheus.GaugeValue,
		float64(metrics.AggregatorThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AggregatorThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.AggregatorThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AggregatorThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.AggregatorThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DDLWorkerThreads,
		prometheus.GaugeValue,
		float64(metrics.DDLWorkerThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DDLWorkerThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DDLWorkerThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DDLWorkerThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DDLWorkerThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageDistributedThreads,
		prometheus.GaugeValue,
		float64(metrics.StorageDistributedThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageDistributedThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StorageDistributedThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageDistributedThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StorageDistributedThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedInsertThreads,
		prometheus.GaugeValue,
		float64(metrics.DistributedInsertThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedInsertThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DistributedInsertThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedInsertThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DistributedInsertThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageS3Threads,
		prometheus.GaugeValue,
		float64(metrics.StorageS3Threads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageS3ThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StorageS3ThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageS3ThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StorageS3ThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageS3Threads,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageS3Threads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageS3ThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageS3ThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageS3ThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageS3ThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageObjectStorageThreads,
		prometheus.GaugeValue,
		float64(metrics.StorageObjectStorageThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageObjectStorageThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StorageObjectStorageThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageObjectStorageThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StorageObjectStorageThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageAzureThreads,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageAzureThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageAzureThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageAzureThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageAzureThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageAzureThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BuildVectorSimilarityIndexThreads,
		prometheus.GaugeValue,
		float64(metrics.BuildVectorSimilarityIndexThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BuildVectorSimilarityIndexThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.BuildVectorSimilarityIndexThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BuildVectorSimilarityIndexThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.BuildVectorSimilarityIndexThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ObjectStorageQueueRegisteredServers,
		prometheus.GaugeValue,
		float64(metrics.ObjectStorageQueueRegisteredServers),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IcebergCatalogThreads,
		prometheus.GaugeValue,
		float64(metrics.IcebergCatalogThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IcebergCatalogThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.IcebergCatalogThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IcebergCatalogThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.IcebergCatalogThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelWithQueryThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelWithQueryThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelWithQueryActiveThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelWithQueryActiveThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParallelWithQueryScheduledThreads,
		prometheus.GaugeValue,
		float64(metrics.ParallelWithQueryScheduledThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableAzureDirectoryMapSize,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableAzureDirectoryMapSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableAzureFileCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableAzureFileCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableAzureUniqueFileNamesCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableAzureUniqueFileNamesCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableLocalDirectoryMapSize,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableLocalDirectoryMapSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableLocalFileCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableLocalFileCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableLocalUniqueFileNamesCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableLocalUniqueFileNamesCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableS3DirectoryMapSize,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableS3DirectoryMapSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableS3FileCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableS3FileCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskPlainRewritableS3UniqueFileNamesCount,
		prometheus.GaugeValue,
		float64(metrics.DiskPlainRewritableS3UniqueFileNamesCount),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeFetchPartitionThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeFetchPartitionThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeFetchPartitionThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeFetchPartitionThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeFetchPartitionThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeFetchPartitionThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsLoaderThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsLoaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsLoaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsLoaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsLoaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsLoaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeOutdatedPartsLoaderThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeOutdatedPartsLoaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeOutdatedPartsLoaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeOutdatedPartsLoaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeOutdatedPartsLoaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeOutdatedPartsLoaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeUnexpectedPartsLoaderThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeUnexpectedPartsLoaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeUnexpectedPartsLoaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeUnexpectedPartsLoaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeUnexpectedPartsLoaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeUnexpectedPartsLoaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsCleanerThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsCleanerThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsCleanerThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsCleanerThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreePartsCleanerThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreePartsCleanerThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseReplicatedCreateTablesThreads,
		prometheus.GaugeValue,
		float64(metrics.DatabaseReplicatedCreateTablesThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseReplicatedCreateTablesThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DatabaseReplicatedCreateTablesThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DatabaseReplicatedCreateTablesThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DatabaseReplicatedCreateTablesThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IDiskCopierThreads,
		prometheus.GaugeValue,
		float64(metrics.IDiskCopierThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IDiskCopierThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.IDiskCopierThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IDiskCopierThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.IDiskCopierThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SystemReplicasThreads,
		prometheus.GaugeValue,
		float64(metrics.SystemReplicasThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SystemReplicasThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.SystemReplicasThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SystemReplicasThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.SystemReplicasThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestartReplicaThreads,
		prometheus.GaugeValue,
		float64(metrics.RestartReplicaThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestartReplicaThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.RestartReplicaThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RestartReplicaThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.RestartReplicaThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryPipelineExecutorThreads,
		prometheus.GaugeValue,
		float64(metrics.QueryPipelineExecutorThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryPipelineExecutorThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.QueryPipelineExecutorThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.QueryPipelineExecutorThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.QueryPipelineExecutorThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderThreads,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderIOThreads,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderIOThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderIOThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderIOThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetDecoderIOThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ParquetDecoderIOThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetEncoderThreads,
		prometheus.GaugeValue,
		float64(metrics.ParquetEncoderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetEncoderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.ParquetEncoderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ParquetEncoderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.ParquetEncoderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeSubcolumnsReaderThreads,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeSubcolumnsReaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeSubcolumnsReaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeSubcolumnsReaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeSubcolumnsReaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeSubcolumnsReaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DWARFReaderThreads,
		prometheus.GaugeValue,
		float64(metrics.DWARFReaderThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DWARFReaderThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.DWARFReaderThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DWARFReaderThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.DWARFReaderThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OutdatedPartsLoadingThreads,
		prometheus.GaugeValue,
		float64(metrics.OutdatedPartsLoadingThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OutdatedPartsLoadingThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.OutdatedPartsLoadingThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.OutdatedPartsLoadingThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.OutdatedPartsLoadingThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PolygonDictionaryThreads,
		prometheus.GaugeValue,
		float64(metrics.PolygonDictionaryThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PolygonDictionaryThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.PolygonDictionaryThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PolygonDictionaryThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.PolygonDictionaryThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedBytesToInsert,
		prometheus.GaugeValue,
		float64(metrics.DistributedBytesToInsert),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BrokenDistributedBytesToInsert,
		prometheus.GaugeValue,
		float64(metrics.BrokenDistributedBytesToInsert),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistributedFilesToInsert,
		prometheus.GaugeValue,
		float64(metrics.DistributedFilesToInsert),
	)
	ch <- prometheus.MustNewConstMetric(
		e.BrokenDistributedFilesToInsert,
		prometheus.GaugeValue,
		float64(metrics.BrokenDistributedFilesToInsert),
	)
	ch <- prometheus.MustNewConstMetric(
		e.TablesToDropQueueSize,
		prometheus.GaugeValue,
		float64(metrics.TablesToDropQueueSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MaxDDLEntryID,
		prometheus.GaugeValue,
		float64(metrics.MaxDDLEntryID),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MaxPushedDDLEntryID,
		prometheus.GaugeValue,
		float64(metrics.MaxPushedDDLEntryID),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsTemporary,
		prometheus.GaugeValue,
		float64(metrics.PartsTemporary),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsPreCommitted,
		prometheus.GaugeValue,
		float64(metrics.PartsPreCommitted),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsCommitted,
		prometheus.GaugeValue,
		float64(metrics.PartsCommitted),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsPreActive,
		prometheus.GaugeValue,
		float64(metrics.PartsPreActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsActive,
		prometheus.GaugeValue,
		float64(metrics.PartsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AttachedDatabase,
		prometheus.GaugeValue,
		float64(metrics.AttachedDatabase),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AttachedTable,
		prometheus.GaugeValue,
		float64(metrics.AttachedTable),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AttachedReplicatedTable,
		prometheus.GaugeValue,
		float64(metrics.AttachedReplicatedTable),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AttachedView,
		prometheus.GaugeValue,
		float64(metrics.AttachedView),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AttachedDictionary,
		prometheus.GaugeValue,
		float64(metrics.AttachedDictionary),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsOutdated,
		prometheus.GaugeValue,
		float64(metrics.PartsOutdated),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsDeleting,
		prometheus.GaugeValue,
		float64(metrics.PartsDeleting),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsDeleteOnDestroy,
		prometheus.GaugeValue,
		float64(metrics.PartsDeleteOnDestroy),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsWide,
		prometheus.GaugeValue,
		float64(metrics.PartsWide),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PartsCompact,
		prometheus.GaugeValue,
		float64(metrics.PartsCompact),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MMappedFiles,
		prometheus.GaugeValue,
		float64(metrics.MMappedFiles),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MMappedFileBytes,
		prometheus.GaugeValue,
		float64(metrics.MMappedFileBytes),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsynchronousReadWait,
		prometheus.GaugeValue,
		float64(metrics.AsynchronousReadWait),
	)
	ch <- prometheus.MustNewConstMetric(
		e.PendingAsyncInsert,
		prometheus.GaugeValue,
		float64(metrics.PendingAsyncInsert),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaConsumers,
		prometheus.GaugeValue,
		float64(metrics.KafkaConsumers),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaConsumersWithAssignment,
		prometheus.GaugeValue,
		float64(metrics.KafkaConsumersWithAssignment),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaProducers,
		prometheus.GaugeValue,
		float64(metrics.KafkaProducers),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaLibrdkafkaThreads,
		prometheus.GaugeValue,
		float64(metrics.KafkaLibrdkafkaThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaBackgroundReads,
		prometheus.GaugeValue,
		float64(metrics.KafkaBackgroundReads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaConsumersInUse,
		prometheus.GaugeValue,
		float64(metrics.KafkaConsumersInUse),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaWrites,
		prometheus.GaugeValue,
		float64(metrics.KafkaWrites),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KafkaAssignedPartitions,
		prometheus.GaugeValue,
		float64(metrics.KafkaAssignedPartitions),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheReadBuffers,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheReadBuffers),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheFileSegments,
		prometheus.GaugeValue,
		float64(metrics.CacheFileSegments),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheDetachedFileSegments,
		prometheus.GaugeValue,
		float64(metrics.CacheDetachedFileSegments),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheSize,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheSizeLimit,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheSizeLimit),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheElements,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheElements),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheDownloadQueueElements,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheDownloadQueueElements),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheDelayedCleanupElements,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheDelayedCleanupElements),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilesystemCacheHoldFileSegments,
		prometheus.GaugeValue,
		float64(metrics.FilesystemCacheHoldFileSegments),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AsyncInsertCacheSize,
		prometheus.GaugeValue,
		float64(metrics.AsyncInsertCacheSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SkippingIndexCacheSize,
		prometheus.GaugeValue,
		float64(metrics.SkippingIndexCacheSize),
	)
	ch <- prometheus.MustNewConstMetric(
		e.S3Requests,
		prometheus.GaugeValue,
		float64(metrics.S3Requests),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KeeperAliveConnections,
		prometheus.GaugeValue,
		float64(metrics.KeeperAliveConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.KeeperOutstandingRequests,
		prometheus.GaugeValue,
		float64(metrics.KeeperOutstandingRequests),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ThreadsInOvercommitTracker,
		prometheus.GaugeValue,
		float64(metrics.ThreadsInOvercommitTracker),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOUringPendingEvents,
		prometheus.GaugeValue,
		float64(metrics.IOUringPendingEvents),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IOUringInFlightEvents,
		prometheus.GaugeValue,
		float64(metrics.IOUringInFlightEvents),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ReadTaskRequestsSent,
		prometheus.GaugeValue,
		float64(metrics.ReadTaskRequestsSent),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeReadTaskRequestsSent,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeReadTaskRequestsSent),
	)
	ch <- prometheus.MustNewConstMetric(
		e.MergeTreeAllRangesAnnouncementsSent,
		prometheus.GaugeValue,
		float64(metrics.MergeTreeAllRangesAnnouncementsSent),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CreatedTimersInQueryProfiler,
		prometheus.GaugeValue,
		float64(metrics.CreatedTimersInQueryProfiler),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ActiveTimersInQueryProfiler,
		prometheus.GaugeValue,
		float64(metrics.ActiveTimersInQueryProfiler),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RefreshableViews,
		prometheus.GaugeValue,
		float64(metrics.RefreshableViews),
	)
	ch <- prometheus.MustNewConstMetric(
		e.RefreshingViews,
		prometheus.GaugeValue,
		float64(metrics.RefreshingViews),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageBufferFlushThreads,
		prometheus.GaugeValue,
		float64(metrics.StorageBufferFlushThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageBufferFlushThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.StorageBufferFlushThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageBufferFlushThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.StorageBufferFlushThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedMergeTreeThreads,
		prometheus.GaugeValue,
		float64(metrics.SharedMergeTreeThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedMergeTreeThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.SharedMergeTreeThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedMergeTreeThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.SharedMergeTreeThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedMergeTreeFetch,
		prometheus.GaugeValue,
		float64(metrics.SharedMergeTreeFetch),
	)
	ch <- prometheus.MustNewConstMetric(
		e.CacheWarmerBytesInProgress,
		prometheus.GaugeValue,
		float64(metrics.CacheWarmerBytesInProgress),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheOpenedConnections,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheOpenedConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheUsedConnections,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheUsedConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheAllocatedConnections,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheAllocatedConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheBorrowedConnections,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheBorrowedConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheReadRequests,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheReadRequests),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheWriteRequests,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheWriteRequests),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheServerConnections,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheServerConnections),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheRegisteredServers,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheRegisteredServers),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheRegisteredServersCurrentAZ,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheRegisteredServersCurrentAZ),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DistrCacheServerS3CachedClients,
		prometheus.GaugeValue,
		float64(metrics.DistrCacheServerS3CachedClients),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SchedulerIOReadScheduled,
		prometheus.GaugeValue,
		float64(metrics.SchedulerIOReadScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SchedulerIOWriteScheduled,
		prometheus.GaugeValue,
		float64(metrics.SchedulerIOWriteScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageConnectionsStored,
		prometheus.GaugeValue,
		float64(metrics.StorageConnectionsStored),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StorageConnectionsTotal,
		prometheus.GaugeValue,
		float64(metrics.StorageConnectionsTotal),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskConnectionsStored,
		prometheus.GaugeValue,
		float64(metrics.DiskConnectionsStored),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskConnectionsTotal,
		prometheus.GaugeValue,
		float64(metrics.DiskConnectionsTotal),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HTTPConnectionsStored,
		prometheus.GaugeValue,
		float64(metrics.HTTPConnectionsStored),
	)
	ch <- prometheus.MustNewConstMetric(
		e.HTTPConnectionsTotal,
		prometheus.GaugeValue,
		float64(metrics.HTTPConnectionsTotal),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AddressesActive,
		prometheus.GaugeValue,
		float64(metrics.AddressesActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.AddressesBanned,
		prometheus.GaugeValue,
		float64(metrics.AddressesBanned),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilteringMarksWithPrimaryKey,
		prometheus.GaugeValue,
		float64(metrics.FilteringMarksWithPrimaryKey),
	)
	ch <- prometheus.MustNewConstMetric(
		e.FilteringMarksWithSecondaryKeys,
		prometheus.GaugeValue,
		float64(metrics.FilteringMarksWithSecondaryKeys),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrencyControlAcquired,
		prometheus.GaugeValue,
		float64(metrics.ConcurrencyControlAcquired),
	)
	ch <- prometheus.MustNewConstMetric(
		e.ConcurrencyControlSoftLimit,
		prometheus.GaugeValue,
		float64(metrics.ConcurrencyControlSoftLimit),
	)
	ch <- prometheus.MustNewConstMetric(
		e.DiskS3NoSuchKeyErrors,
		prometheus.GaugeValue,
		float64(metrics.DiskS3NoSuchKeyErrors),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogStateApplicationThreads,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogStateApplicationThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogStateApplicationThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogStateApplicationThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogStateApplicationThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogStateApplicationThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropLocalThreads,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropLocalThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropLocalThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropLocalThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropLocalThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropLocalThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropZooKeeperThreads,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropZooKeeperThreads),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropZooKeeperThreadsActive,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropZooKeeperThreadsActive),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedCatalogDropZooKeeperThreadsScheduled,
		prometheus.GaugeValue,
		float64(metrics.SharedCatalogDropZooKeeperThreadsScheduled),
	)
	ch <- prometheus.MustNewConstMetric(
		e.SharedDatabaseCatalogTablesInLocalDropDetachQueue,
		prometheus.GaugeValue,
		float64(metrics.SharedDatabaseCatalogTablesInLocalDropDetachQueue),
	)
	ch <- prometheus.MustNewConstMetric(
		e.StartupScriptsExecutionState,
		prometheus.GaugeValue,
		float64(metrics.StartupScriptsExecutionState),
	)
	ch <- prometheus.MustNewConstMetric(
		e.IsServerShuttingDown,
		prometheus.GaugeValue,
		float64(metrics.IsServerShuttingDown),
	)
	return nil
}

func (e *MetricsURIExporter) parseMetricsURIResponse(uri string) (*MetricsUri, error) {
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

	var event MetricsUri
	err = json.Unmarshal([]byte(jsonData), &event)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}
	return &event, nil
}
