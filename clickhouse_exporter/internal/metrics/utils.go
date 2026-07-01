package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type MetricsUri struct {
	Query                                             float64 `json:"Query"`
	Merge                                             float64 `json:"Merge"`
	MergeParts                                        float64 `json:"MergeParts"`
	Move                                              float64 `json:"Move"`
	PartMutation                                      float64 `json:"PartMutation"`
	ReplicatedFetch                                   float64 `json:"ReplicatedFetch"`
	ReplicatedSend                                    float64 `json:"ReplicatedSend"`
	ReplicatedChecks                                  float64 `json:"ReplicatedChecks"`
	BackgroundMergesAndMutationsPoolTask              float64 `json:"BackgroundMergesAndMutationsPoolTask"`
	BackgroundMergesAndMutationsPoolSize              float64 `json:"BackgroundMergesAndMutationsPoolSize"`
	BackgroundFetchesPoolTask                         float64 `json:"BackgroundFetchesPoolTask"`
	BackgroundFetchesPoolSize                         float64 `json:"BackgroundFetchesPoolSize"`
	BackgroundCommonPoolTask                          float64 `json:"BackgroundCommonPoolTask"`
	BackgroundCommonPoolSize                          float64 `json:"BackgroundCommonPoolSize"`
	BackgroundMovePoolTask                            float64 `json:"BackgroundMovePoolTask"`
	BackgroundMovePoolSize                            float64 `json:"BackgroundMovePoolSize"`
	BackgroundSchedulePoolTask                        float64 `json:"BackgroundSchedulePoolTask"`
	BackgroundSchedulePoolSize                        float64 `json:"BackgroundSchedulePoolSize"`
	BackgroundBufferFlushSchedulePoolTask             float64 `json:"BackgroundBufferFlushSchedulePoolTask"`
	BackgroundBufferFlushSchedulePoolSize             float64 `json:"BackgroundBufferFlushSchedulePoolSize"`
	BackgroundDistributedSchedulePoolTask             float64 `json:"BackgroundDistributedSchedulePoolTask"`
	BackgroundDistributedSchedulePoolSize             float64 `json:"BackgroundDistributedSchedulePoolSize"`
	BackgroundMessageBrokerSchedulePoolTask           float64 `json:"BackgroundMessageBrokerSchedulePoolTask"`
	BackgroundMessageBrokerSchedulePoolSize           float64 `json:"BackgroundMessageBrokerSchedulePoolSize"`
	CacheDictionaryUpdateQueueBatches                 float64 `json:"CacheDictionaryUpdateQueueBatches"`
	CacheDictionaryUpdateQueueKeys                    float64 `json:"CacheDictionaryUpdateQueueKeys"`
	DiskSpaceReservedForMerge                         float64 `json:"DiskSpaceReservedForMerge"`
	DistributedSend                                   float64 `json:"DistributedSend"`
	QueryPreempted                                    float64 `json:"QueryPreempted"`
	TCPConnection                                     float64 `json:"TCPConnection"`
	MySQLConnection                                   float64 `json:"MySQLConnection"`
	HTTPConnection                                    float64 `json:"HTTPConnection"`
	InterserverConnection                             float64 `json:"InterserverConnection"`
	PostgreSQLConnection                              float64 `json:"PostgreSQLConnection"`
	OpenFileForRead                                   float64 `json:"OpenFileForRead"`
	OpenFileForWrite                                  float64 `json:"OpenFileForWrite"`
	Compressing                                       float64 `json:"Compressing"`
	Decompressing                                     float64 `json:"Decompressing"`
	ParallelCompressedWriteBufferThreads              float64 `json:"ParallelCompressedWriteBufferThreads"`
	ParallelCompressedWriteBufferWait                 float64 `json:"ParallelCompressedWriteBufferWait"`
	TotalTemporaryFiles                               float64 `json:"TotalTemporaryFiles"`
	TemporaryFilesForSort                             float64 `json:"TemporaryFilesForSort"`
	TemporaryFilesForAggregation                      float64 `json:"TemporaryFilesForAggregation"`
	TemporaryFilesForJoin                             float64 `json:"TemporaryFilesForJoin"`
	TemporaryFilesForMerge                            float64 `json:"TemporaryFilesForMerge"`
	TemporaryFilesUnknown                             float64 `json:"TemporaryFilesUnknown"`
	Read                                              float64 `json:"Read"`
	RemoteRead                                        float64 `json:"RemoteRead"`
	Write                                             float64 `json:"Write"`
	NetworkReceive                                    float64 `json:"NetworkReceive"`
	NetworkSend                                       float64 `json:"NetworkSend"`
	SendScalars                                       float64 `json:"SendScalars"`
	SendExternalTables                                float64 `json:"SendExternalTables"`
	QueryThread                                       float64 `json:"QueryThread"`
	ReadonlyReplica                                   float64 `json:"ReadonlyReplica"`
	MemoryTracking                                    float64 `json:"MemoryTracking"`
	MemoryTrackingUncorrected                         float64 `json:"MemoryTrackingUncorrected"`
	MergesMutationsMemoryTracking                     float64 `json:"MergesMutationsMemoryTracking"`
	EphemeralNode                                     float64 `json:"EphemeralNode"`
	ZooKeeperSession                                  float64 `json:"ZooKeeperSession"`
	ZooKeeperWatch                                    float64 `json:"ZooKeeperWatch"`
	ZooKeeperRequest                                  float64 `json:"ZooKeeperRequest"`
	DelayedInserts                                    float64 `json:"DelayedInserts"`
	ContextLockWait                                   float64 `json:"ContextLockWait"`
	StorageBufferRows                                 float64 `json:"StorageBufferRows"`
	StorageBufferBytes                                float64 `json:"StorageBufferBytes"`
	DictCacheRequests                                 float64 `json:"DictCacheRequests"`
	Revision                                          float64 `json:"Revision"`
	VersionInteger                                    float64 `json:"VersionInteger"`
	RWLockWaitingReaders                              float64 `json:"RWLockWaitingReaders"`
	RWLockWaitingWriters                              float64 `json:"RWLockWaitingWriters"`
	RWLockActiveReaders                               float64 `json:"RWLockActiveReaders"`
	RWLockActiveWriters                               float64 `json:"RWLockActiveWriters"`
	GlobalThread                                      float64 `json:"GlobalThread"`
	GlobalThreadActive                                float64 `json:"GlobalThreadActive"`
	GlobalThreadScheduled                             float64 `json:"GlobalThreadScheduled"`
	LocalThread                                       float64 `json:"LocalThread"`
	LocalThreadActive                                 float64 `json:"LocalThreadActive"`
	LocalThreadScheduled                              float64 `json:"LocalThreadScheduled"`
	MergeTreeDataSelectExecutorThreads                float64 `json:"MergeTreeDataSelectExecutorThreads"`
	MergeTreeDataSelectExecutorThreadsActive          float64 `json:"MergeTreeDataSelectExecutorThreadsActive"`
	MergeTreeDataSelectExecutorThreadsScheduled       float64 `json:"MergeTreeDataSelectExecutorThreadsScheduled"`
	BackupsThreads                                    float64 `json:"BackupsThreads"`
	BackupsThreadsActive                              float64 `json:"BackupsThreadsActive"`
	BackupsThreadsScheduled                           float64 `json:"BackupsThreadsScheduled"`
	RestoreThreads                                    float64 `json:"RestoreThreads"`
	RestoreThreadsActive                              float64 `json:"RestoreThreadsActive"`
	RestoreThreadsScheduled                           float64 `json:"RestoreThreadsScheduled"`
	MarksLoaderThreads                                float64 `json:"MarksLoaderThreads"`
	MarksLoaderThreadsActive                          float64 `json:"MarksLoaderThreadsActive"`
	MarksLoaderThreadsScheduled                       float64 `json:"MarksLoaderThreadsScheduled"`
	IOPrefetchThreads                                 float64 `json:"IOPrefetchThreads"`
	IOPrefetchThreadsActive                           float64 `json:"IOPrefetchThreadsActive"`
	IOPrefetchThreadsScheduled                        float64 `json:"IOPrefetchThreadsScheduled"`
	IOWriterThreads                                   float64 `json:"IOWriterThreads"`
	IOWriterThreadsActive                             float64 `json:"IOWriterThreadsActive"`
	IOWriterThreadsScheduled                          float64 `json:"IOWriterThreadsScheduled"`
	IOThreads                                         float64 `json:"IOThreads"`
	IOThreadsActive                                   float64 `json:"IOThreadsActive"`
	IOThreadsScheduled                                float64 `json:"IOThreadsScheduled"`
	CompressionThread                                 float64 `json:"CompressionThread"`
	CompressionThreadActive                           float64 `json:"CompressionThreadActive"`
	CompressionThreadScheduled                        float64 `json:"CompressionThreadScheduled"`
	ThreadPoolRemoteFSReaderThreads                   float64 `json:"ThreadPoolRemoteFSReaderThreads"`
	ThreadPoolRemoteFSReaderThreadsActive             float64 `json:"ThreadPoolRemoteFSReaderThreadsActive"`
	ThreadPoolRemoteFSReaderThreadsScheduled          float64 `json:"ThreadPoolRemoteFSReaderThreadsScheduled"`
	ThreadPoolFSReaderThreads                         float64 `json:"ThreadPoolFSReaderThreads"`
	ThreadPoolFSReaderThreadsActive                   float64 `json:"ThreadPoolFSReaderThreadsActive"`
	ThreadPoolFSReaderThreadsScheduled                float64 `json:"ThreadPoolFSReaderThreadsScheduled"`
	BackupsIOThreads                                  float64 `json:"BackupsIOThreads"`
	BackupsIOThreadsActive                            float64 `json:"BackupsIOThreadsActive"`
	BackupsIOThreadsScheduled                         float64 `json:"BackupsIOThreadsScheduled"`
	DiskObjectStorageAsyncThreads                     float64 `json:"DiskObjectStorageAsyncThreads"`
	DiskObjectStorageAsyncThreadsActive               float64 `json:"DiskObjectStorageAsyncThreadsActive"`
	StorageHiveThreads                                float64 `json:"StorageHiveThreads"`
	StorageHiveThreadsActive                          float64 `json:"StorageHiveThreadsActive"`
	StorageHiveThreadsScheduled                       float64 `json:"StorageHiveThreadsScheduled"`
	TablesLoaderBackgroundThreads                     float64 `json:"TablesLoaderBackgroundThreads"`
	TablesLoaderBackgroundThreadsActive               float64 `json:"TablesLoaderBackgroundThreadsActive"`
	TablesLoaderBackgroundThreadsScheduled            float64 `json:"TablesLoaderBackgroundThreadsScheduled"`
	TablesLoaderForegroundThreads                     float64 `json:"TablesLoaderForegroundThreads"`
	TablesLoaderForegroundThreadsActive               float64 `json:"TablesLoaderForegroundThreadsActive"`
	TablesLoaderForegroundThreadsScheduled            float64 `json:"TablesLoaderForegroundThreadsScheduled"`
	DatabaseOnDiskThreads                             float64 `json:"DatabaseOnDiskThreads"`
	DatabaseOnDiskThreadsActive                       float64 `json:"DatabaseOnDiskThreadsActive"`
	DatabaseOnDiskThreadsScheduled                    float64 `json:"DatabaseOnDiskThreadsScheduled"`
	DatabaseBackupThreads                             float64 `json:"DatabaseBackupThreads"`
	DatabaseBackupThreadsActive                       float64 `json:"DatabaseBackupThreadsActive"`
	DatabaseBackupThreadsScheduled                    float64 `json:"DatabaseBackupThreadsScheduled"`
	DatabaseCatalogThreads                            float64 `json:"DatabaseCatalogThreads"`
	DatabaseCatalogThreadsActive                      float64 `json:"DatabaseCatalogThreadsActive"`
	DatabaseCatalogThreadsScheduled                   float64 `json:"DatabaseCatalogThreadsScheduled"`
	DestroyAggregatesThreads                          float64 `json:"DestroyAggregatesThreads"`
	DestroyAggregatesThreadsActive                    float64 `json:"DestroyAggregatesThreadsActive"`
	DestroyAggregatesThreadsScheduled                 float64 `json:"DestroyAggregatesThreadsScheduled"`
	ConcurrentHashJoinPoolThreads                     float64 `json:"ConcurrentHashJoinPoolThreads"`
	ConcurrentHashJoinPoolThreadsActive               float64 `json:"ConcurrentHashJoinPoolThreadsActive"`
	ConcurrentHashJoinPoolThreadsScheduled            float64 `json:"ConcurrentHashJoinPoolThreadsScheduled"`
	HashedDictionaryThreads                           float64 `json:"HashedDictionaryThreads"`
	HashedDictionaryThreadsActive                     float64 `json:"HashedDictionaryThreadsActive"`
	HashedDictionaryThreadsScheduled                  float64 `json:"HashedDictionaryThreadsScheduled"`
	CacheDictionaryThreads                            float64 `json:"CacheDictionaryThreads"`
	CacheDictionaryThreadsActive                      float64 `json:"CacheDictionaryThreadsActive"`
	CacheDictionaryThreadsScheduled                   float64 `json:"CacheDictionaryThreadsScheduled"`
	ParallelFormattingOutputFormatThreads             float64 `json:"ParallelFormattingOutputFormatThreads"`
	ParallelFormattingOutputFormatThreadsActive       float64 `json:"ParallelFormattingOutputFormatThreadsActive"`
	ParallelFormattingOutputFormatThreadsScheduled    float64 `json:"ParallelFormattingOutputFormatThreadsScheduled"`
	ParallelParsingInputFormatThreads                 float64 `json:"ParallelParsingInputFormatThreads"`
	ParallelParsingInputFormatThreadsActive           float64 `json:"ParallelParsingInputFormatThreadsActive"`
	ParallelParsingInputFormatThreadsScheduled        float64 `json:"ParallelParsingInputFormatThreadsScheduled"`
	MergeTreeBackgroundExecutorThreads                float64 `json:"MergeTreeBackgroundExecutorThreads"`
	MergeTreeBackgroundExecutorThreadsActive          float64 `json:"MergeTreeBackgroundExecutorThreadsActive"`
	MergeTreeBackgroundExecutorThreadsScheduled       float64 `json:"MergeTreeBackgroundExecutorThreadsScheduled"`
	AsynchronousInsertThreads                         float64 `json:"AsynchronousInsertThreads"`
	AsynchronousInsertThreadsActive                   float64 `json:"AsynchronousInsertThreadsActive"`
	AsynchronousInsertThreadsScheduled                float64 `json:"AsynchronousInsertThreadsScheduled"`
	AsynchronousInsertQueueSize                       float64 `json:"AsynchronousInsertQueueSize"`
	AsynchronousInsertQueueBytes                      float64 `json:"AsynchronousInsertQueueBytes"`
	StartupSystemTablesThreads                        float64 `json:"StartupSystemTablesThreads"`
	StartupSystemTablesThreadsActive                  float64 `json:"StartupSystemTablesThreadsActive"`
	StartupSystemTablesThreadsScheduled               float64 `json:"StartupSystemTablesThreadsScheduled"`
	AggregatorThreads                                 float64 `json:"AggregatorThreads"`
	AggregatorThreadsActive                           float64 `json:"AggregatorThreadsActive"`
	AggregatorThreadsScheduled                        float64 `json:"AggregatorThreadsScheduled"`
	DDLWorkerThreads                                  float64 `json:"DDLWorkerThreads"`
	DDLWorkerThreadsActive                            float64 `json:"DDLWorkerThreadsActive"`
	DDLWorkerThreadsScheduled                         float64 `json:"DDLWorkerThreadsScheduled"`
	StorageDistributedThreads                         float64 `json:"StorageDistributedThreads"`
	StorageDistributedThreadsActive                   float64 `json:"StorageDistributedThreadsActive"`
	StorageDistributedThreadsScheduled                float64 `json:"StorageDistributedThreadsScheduled"`
	DistributedInsertThreads                          float64 `json:"DistributedInsertThreads"`
	DistributedInsertThreadsActive                    float64 `json:"DistributedInsertThreadsActive"`
	DistributedInsertThreadsScheduled                 float64 `json:"DistributedInsertThreadsScheduled"`
	StorageS3Threads                                  float64 `json:"StorageS3Threads"`
	StorageS3ThreadsActive                            float64 `json:"StorageS3ThreadsActive"`
	StorageS3ThreadsScheduled                         float64 `json:"StorageS3ThreadsScheduled"`
	ObjectStorageS3Threads                            float64 `json:"ObjectStorageS3Threads"`
	ObjectStorageS3ThreadsActive                      float64 `json:"ObjectStorageS3ThreadsActive"`
	ObjectStorageS3ThreadsScheduled                   float64 `json:"ObjectStorageS3ThreadsScheduled"`
	StorageObjectStorageThreads                       float64 `json:"StorageObjectStorageThreads"`
	StorageObjectStorageThreadsActive                 float64 `json:"StorageObjectStorageThreadsActive"`
	StorageObjectStorageThreadsScheduled              float64 `json:"StorageObjectStorageThreadsScheduled"`
	ObjectStorageAzureThreads                         float64 `json:"ObjectStorageAzureThreads"`
	ObjectStorageAzureThreadsActive                   float64 `json:"ObjectStorageAzureThreadsActive"`
	ObjectStorageAzureThreadsScheduled                float64 `json:"ObjectStorageAzureThreadsScheduled"`
	BuildVectorSimilarityIndexThreads                 float64 `json:"BuildVectorSimilarityIndexThreads"`
	BuildVectorSimilarityIndexThreadsActive           float64 `json:"BuildVectorSimilarityIndexThreadsActive"`
	BuildVectorSimilarityIndexThreadsScheduled        float64 `json:"BuildVectorSimilarityIndexThreadsScheduled"`
	ObjectStorageQueueRegisteredServers               float64 `json:"ObjectStorageQueueRegisteredServers"`
	IcebergCatalogThreads                             float64 `json:"IcebergCatalogThreads"`
	IcebergCatalogThreadsActive                       float64 `json:"IcebergCatalogThreadsActive"`
	IcebergCatalogThreadsScheduled                    float64 `json:"IcebergCatalogThreadsScheduled"`
	ParallelWithQueryThreads                          float64 `json:"ParallelWithQueryThreads"`
	ParallelWithQueryActiveThreads                    float64 `json:"ParallelWithQueryActiveThreads"`
	ParallelWithQueryScheduledThreads                 float64 `json:"ParallelWithQueryScheduledThreads"`
	DiskPlainRewritableAzureDirectoryMapSize          float64 `json:"DiskPlainRewritableAzureDirectoryMapSize"`
	DiskPlainRewritableAzureFileCount                 float64 `json:"DiskPlainRewritableAzureFileCount"`
	DiskPlainRewritableAzureUniqueFileNamesCount      float64 `json:"DiskPlainRewritableAzureUniqueFileNamesCount"`
	DiskPlainRewritableLocalDirectoryMapSize          float64 `json:"DiskPlainRewritableLocalDirectoryMapSize"`
	DiskPlainRewritableLocalFileCount                 float64 `json:"DiskPlainRewritableLocalFileCount"`
	DiskPlainRewritableLocalUniqueFileNamesCount      float64 `json:"DiskPlainRewritableLocalUniqueFileNamesCount"`
	DiskPlainRewritableS3DirectoryMapSize             float64 `json:"DiskPlainRewritableS3DirectoryMapSize"`
	DiskPlainRewritableS3FileCount                    float64 `json:"DiskPlainRewritableS3FileCount"`
	DiskPlainRewritableS3UniqueFileNamesCount         float64 `json:"DiskPlainRewritableS3UniqueFileNamesCount"`
	MergeTreeFetchPartitionThreads                    float64 `json:"MergeTreeFetchPartitionThreads"`
	MergeTreeFetchPartitionThreadsActive              float64 `json:"MergeTreeFetchPartitionThreadsActive"`
	MergeTreeFetchPartitionThreadsScheduled           float64 `json:"MergeTreeFetchPartitionThreadsScheduled"`
	MergeTreePartsLoaderThreads                       float64 `json:"MergeTreePartsLoaderThreads"`
	MergeTreePartsLoaderThreadsActive                 float64 `json:"MergeTreePartsLoaderThreadsActive"`
	MergeTreePartsLoaderThreadsScheduled              float64 `json:"MergeTreePartsLoaderThreadsScheduled"`
	MergeTreeOutdatedPartsLoaderThreads               float64 `json:"MergeTreeOutdatedPartsLoaderThreads"`
	MergeTreeOutdatedPartsLoaderThreadsActive         float64 `json:"MergeTreeOutdatedPartsLoaderThreadsActive"`
	MergeTreeOutdatedPartsLoaderThreadsScheduled      float64 `json:"MergeTreeOutdatedPartsLoaderThreadsScheduled"`
	MergeTreeUnexpectedPartsLoaderThreads             float64 `json:"MergeTreeUnexpectedPartsLoaderThreads"`
	MergeTreeUnexpectedPartsLoaderThreadsActive       float64 `json:"MergeTreeUnexpectedPartsLoaderThreadsActive"`
	MergeTreeUnexpectedPartsLoaderThreadsScheduled    float64 `json:"MergeTreeUnexpectedPartsLoaderThreadsScheduled"`
	MergeTreePartsCleanerThreads                      float64 `json:"MergeTreePartsCleanerThreads"`
	MergeTreePartsCleanerThreadsActive                float64 `json:"MergeTreePartsCleanerThreadsActive"`
	MergeTreePartsCleanerThreadsScheduled             float64 `json:"MergeTreePartsCleanerThreadsScheduled"`
	DatabaseReplicatedCreateTablesThreads             float64 `json:"DatabaseReplicatedCreateTablesThreads"`
	DatabaseReplicatedCreateTablesThreadsActive       float64 `json:"DatabaseReplicatedCreateTablesThreadsActive"`
	DatabaseReplicatedCreateTablesThreadsScheduled    float64 `json:"DatabaseReplicatedCreateTablesThreadsScheduled"`
	IDiskCopierThreads                                float64 `json:"IDiskCopierThreads"`
	IDiskCopierThreadsActive                          float64 `json:"IDiskCopierThreadsActive"`
	IDiskCopierThreadsScheduled                       float64 `json:"IDiskCopierThreadsScheduled"`
	SystemReplicasThreads                             float64 `json:"SystemReplicasThreads"`
	SystemReplicasThreadsActive                       float64 `json:"SystemReplicasThreadsActive"`
	SystemReplicasThreadsScheduled                    float64 `json:"SystemReplicasThreadsScheduled"`
	RestartReplicaThreads                             float64 `json:"RestartReplicaThreads"`
	RestartReplicaThreadsActive                       float64 `json:"RestartReplicaThreadsActive"`
	RestartReplicaThreadsScheduled                    float64 `json:"RestartReplicaThreadsScheduled"`
	QueryPipelineExecutorThreads                      float64 `json:"QueryPipelineExecutorThreads"`
	QueryPipelineExecutorThreadsActive                float64 `json:"QueryPipelineExecutorThreadsActive"`
	QueryPipelineExecutorThreadsScheduled             float64 `json:"QueryPipelineExecutorThreadsScheduled"`
	ParquetDecoderThreads                             float64 `json:"ParquetDecoderThreads"`
	ParquetDecoderThreadsActive                       float64 `json:"ParquetDecoderThreadsActive"`
	ParquetDecoderThreadsScheduled                    float64 `json:"ParquetDecoderThreadsScheduled"`
	ParquetDecoderIOThreads                           float64 `json:"ParquetDecoderIOThreads"`
	ParquetDecoderIOThreadsActive                     float64 `json:"ParquetDecoderIOThreadsActive"`
	ParquetDecoderIOThreadsScheduled                  float64 `json:"ParquetDecoderIOThreadsScheduled"`
	ParquetEncoderThreads                             float64 `json:"ParquetEncoderThreads"`
	ParquetEncoderThreadsActive                       float64 `json:"ParquetEncoderThreadsActive"`
	ParquetEncoderThreadsScheduled                    float64 `json:"ParquetEncoderThreadsScheduled"`
	MergeTreeSubcolumnsReaderThreads                  float64 `json:"MergeTreeSubcolumnsReaderThreads"`
	MergeTreeSubcolumnsReaderThreadsActive            float64 `json:"MergeTreeSubcolumnsReaderThreadsActive"`
	MergeTreeSubcolumnsReaderThreadsScheduled         float64 `json:"MergeTreeSubcolumnsReaderThreadsScheduled"`
	DWARFReaderThreads                                float64 `json:"DWARFReaderThreads"`
	DWARFReaderThreadsActive                          float64 `json:"DWARFReaderThreadsActive"`
	DWARFReaderThreadsScheduled                       float64 `json:"DWARFReaderThreadsScheduled"`
	OutdatedPartsLoadingThreads                       float64 `json:"OutdatedPartsLoadingThreads"`
	OutdatedPartsLoadingThreadsActive                 float64 `json:"OutdatedPartsLoadingThreadsActive"`
	OutdatedPartsLoadingThreadsScheduled              float64 `json:"OutdatedPartsLoadingThreadsScheduled"`
	PolygonDictionaryThreads                          float64 `json:"PolygonDictionaryThreads"`
	PolygonDictionaryThreadsActive                    float64 `json:"PolygonDictionaryThreadsActive"`
	PolygonDictionaryThreadsScheduled                 float64 `json:"PolygonDictionaryThreadsScheduled"`
	DistributedBytesToInsert                          float64 `json:"DistributedBytesToInsert"`
	BrokenDistributedBytesToInsert                    float64 `json:"BrokenDistributedBytesToInsert"`
	DistributedFilesToInsert                          float64 `json:"DistributedFilesToInsert"`
	BrokenDistributedFilesToInsert                    float64 `json:"BrokenDistributedFilesToInsert"`
	TablesToDropQueueSize                             float64 `json:"TablesToDropQueueSize"`
	MaxDDLEntryID                                     float64 `json:"MaxDDLEntryID"`
	MaxPushedDDLEntryID                               float64 `json:"MaxPushedDDLEntryID"`
	PartsTemporary                                    float64 `json:"PartsTemporary"`
	PartsPreCommitted                                 float64 `json:"PartsPreCommitted"`
	PartsCommitted                                    float64 `json:"PartsCommitted"`
	PartsPreActive                                    float64 `json:"PartsPreActive"`
	PartsActive                                       float64 `json:"PartsActive"`
	AttachedDatabase                                  float64 `json:"AttachedDatabase"`
	AttachedTable                                     float64 `json:"AttachedTable"`
	AttachedReplicatedTable                           float64 `json:"AttachedReplicatedTable"`
	AttachedView                                      float64 `json:"AttachedView"`
	AttachedDictionary                                float64 `json:"AttachedDictionary"`
	PartsOutdated                                     float64 `json:"PartsOutdated"`
	PartsDeleting                                     float64 `json:"PartsDeleting"`
	PartsDeleteOnDestroy                              float64 `json:"PartsDeleteOnDestroy"`
	PartsWide                                         float64 `json:"PartsWide"`
	PartsCompact                                      float64 `json:"PartsCompact"`
	MMappedFiles                                      float64 `json:"MMappedFiles"`
	MMappedFileBytes                                  float64 `json:"MMappedFileBytes"`
	AsynchronousReadWait                              float64 `json:"AsynchronousReadWait"`
	PendingAsyncInsert                                float64 `json:"PendingAsyncInsert"`
	KafkaConsumers                                    float64 `json:"KafkaConsumers"`
	KafkaConsumersWithAssignment                      float64 `json:"KafkaConsumersWithAssignment"`
	KafkaProducers                                    float64 `json:"KafkaProducers"`
	KafkaLibrdkafkaThreads                            float64 `json:"KafkaLibrdkafkaThreads"`
	KafkaBackgroundReads                              float64 `json:"KafkaBackgroundReads"`
	KafkaConsumersInUse                               float64 `json:"KafkaConsumersInUse"`
	KafkaWrites                                       float64 `json:"KafkaWrites"`
	KafkaAssignedPartitions                           float64 `json:"KafkaAssignedPartitions"`
	FilesystemCacheReadBuffers                        float64 `json:"FilesystemCacheReadBuffers"`
	CacheFileSegments                                 float64 `json:"CacheFileSegments"`
	CacheDetachedFileSegments                         float64 `json:"CacheDetachedFileSegments"`
	FilesystemCacheSize                               float64 `json:"FilesystemCacheSize"`
	FilesystemCacheSizeLimit                          float64 `json:"FilesystemCacheSizeLimit"`
	FilesystemCacheElements                           float64 `json:"FilesystemCacheElements"`
	FilesystemCacheDownloadQueueElements              float64 `json:"FilesystemCacheDownloadQueueElements"`
	FilesystemCacheDelayedCleanupElements             float64 `json:"FilesystemCacheDelayedCleanupElements"`
	FilesystemCacheHoldFileSegments                   float64 `json:"FilesystemCacheHoldFileSegments"`
	AsyncInsertCacheSize                              float64 `json:"AsyncInsertCacheSize"`
	SkippingIndexCacheSize                            float64 `json:"SkippingIndexCacheSize"`
	S3Requests                                        float64 `json:"S3Requests"`
	KeeperAliveConnections                            float64 `json:"KeeperAliveConnections"`
	KeeperOutstandingRequests                         float64 `json:"KeeperOutstandingRequests"`
	ThreadsInOvercommitTracker                        float64 `json:"ThreadsInOvercommitTracker"`
	IOUringPendingEvents                              float64 `json:"IOUringPendingEvents"`
	IOUringInFlightEvents                             float64 `json:"IOUringInFlightEvents"`
	ReadTaskRequestsSent                              float64 `json:"ReadTaskRequestsSent"`
	MergeTreeReadTaskRequestsSent                     float64 `json:"MergeTreeReadTaskRequestsSent"`
	MergeTreeAllRangesAnnouncementsSent               float64 `json:"MergeTreeAllRangesAnnouncementsSent"`
	CreatedTimersInQueryProfiler                      float64 `json:"CreatedTimersInQueryProfiler"`
	ActiveTimersInQueryProfiler                       float64 `json:"ActiveTimersInQueryProfiler"`
	RefreshableViews                                  float64 `json:"RefreshableViews"`
	RefreshingViews                                   float64 `json:"RefreshingViews"`
	StorageBufferFlushThreads                         float64 `json:"StorageBufferFlushThreads"`
	StorageBufferFlushThreadsActive                   float64 `json:"StorageBufferFlushThreadsActive"`
	StorageBufferFlushThreadsScheduled                float64 `json:"StorageBufferFlushThreadsScheduled"`
	SharedMergeTreeThreads                            float64 `json:"SharedMergeTreeThreads"`
	SharedMergeTreeThreadsActive                      float64 `json:"SharedMergeTreeThreadsActive"`
	SharedMergeTreeThreadsScheduled                   float64 `json:"SharedMergeTreeThreadsScheduled"`
	SharedMergeTreeFetch                              float64 `json:"SharedMergeTreeFetch"`
	CacheWarmerBytesInProgress                        float64 `json:"CacheWarmerBytesInProgress"`
	DistrCacheOpenedConnections                       float64 `json:"DistrCacheOpenedConnections"`
	DistrCacheUsedConnections                         float64 `json:"DistrCacheUsedConnections"`
	DistrCacheAllocatedConnections                    float64 `json:"DistrCacheAllocatedConnections"`
	DistrCacheBorrowedConnections                     float64 `json:"DistrCacheBorrowedConnections"`
	DistrCacheReadRequests                            float64 `json:"DistrCacheReadRequests"`
	DistrCacheWriteRequests                           float64 `json:"DistrCacheWriteRequests"`
	DistrCacheServerConnections                       float64 `json:"DistrCacheServerConnections"`
	DistrCacheRegisteredServers                       float64 `json:"DistrCacheRegisteredServers"`
	DistrCacheRegisteredServersCurrentAZ              float64 `json:"DistrCacheRegisteredServersCurrentAZ"`
	DistrCacheServerS3CachedClients                   float64 `json:"DistrCacheServerS3CachedClients"`
	SchedulerIOReadScheduled                          float64 `json:"SchedulerIOReadScheduled"`
	SchedulerIOWriteScheduled                         float64 `json:"SchedulerIOWriteScheduled"`
	StorageConnectionsStored                          float64 `json:"StorageConnectionsStored"`
	StorageConnectionsTotal                           float64 `json:"StorageConnectionsTotal"`
	DiskConnectionsStored                             float64 `json:"DiskConnectionsStored"`
	DiskConnectionsTotal                              float64 `json:"DiskConnectionsTotal"`
	HTTPConnectionsStored                             float64 `json:"HTTPConnectionsStored"`
	HTTPConnectionsTotal                              float64 `json:"HTTPConnectionsTotal"`
	AddressesActive                                   float64 `json:"AddressesActive"`
	AddressesBanned                                   float64 `json:"AddressesBanned"`
	FilteringMarksWithPrimaryKey                      float64 `json:"FilteringMarksWithPrimaryKey"`
	FilteringMarksWithSecondaryKeys                   float64 `json:"FilteringMarksWithSecondaryKeys"`
	ConcurrencyControlAcquired                        float64 `json:"ConcurrencyControlAcquired"`
	ConcurrencyControlSoftLimit                       float64 `json:"ConcurrencyControlSoftLimit"`
	DiskS3NoSuchKeyErrors                             float64 `json:"DiskS3NoSuchKeyErrors"`
	SharedCatalogStateApplicationThreads              float64 `json:"SharedCatalogStateApplicationThreads"`
	SharedCatalogStateApplicationThreadsActive        float64 `json:"SharedCatalogStateApplicationThreadsActive"`
	SharedCatalogStateApplicationThreadsScheduled     float64 `json:"SharedCatalogStateApplicationThreadsScheduled"`
	SharedCatalogDropLocalThreads                     float64 `json:"SharedCatalogDropLocalThreads"`
	SharedCatalogDropLocalThreadsActive               float64 `json:"SharedCatalogDropLocalThreadsActive"`
	SharedCatalogDropLocalThreadsScheduled            float64 `json:"SharedCatalogDropLocalThreadsScheduled"`
	SharedCatalogDropZooKeeperThreads                 float64 `json:"SharedCatalogDropZooKeeperThreads"`
	SharedCatalogDropZooKeeperThreadsActive           float64 `json:"SharedCatalogDropZooKeeperThreadsActive"`
	SharedCatalogDropZooKeeperThreadsScheduled        float64 `json:"SharedCatalogDropZooKeeperThreadsScheduled"`
	SharedDatabaseCatalogTablesInLocalDropDetachQueue float64 `json:"SharedDatabaseCatalogTablesInLocalDropDetachQueue"`
	StartupScriptsExecutionState                      float64 `json:"StartupScriptsExecutionState"`
	IsServerShuttingDown                              float64 `json:"IsServerShuttingDown"`
}

type EventsUri struct {
	Query                                                 float64 `json:"Query"`
	SelectQuery                                           float64 `json:"SelectQuery"`
	InitialQuery                                          float64 `json:"InitialQuery"`
	QueriesWithSubqueries                                 float64 `json:"QueriesWithSubqueries"`
	SelectQueriesWithSubqueries                           float64 `json:"SelectQueriesWithSubqueries"`
	FailedQuery                                           float64 `json:"FailedQuery"`
	QueryTimeMicroseconds                                 float64 `json:"QueryTimeMicroseconds"`
	SelectQueryTimeMicroseconds                           float64 `json:"SelectQueryTimeMicroseconds"`
	OtherQueryTimeMicroseconds                            float64 `json:"OtherQueryTimeMicroseconds"`
	FileOpen                                              float64 `json:"FileOpen"`
	Seek                                                  float64 `json:"Seek"`
	ReadBufferFromFileDescriptorRead                      float64 `json:"ReadBufferFromFileDescriptorRead"`
	ReadBufferFromFileDescriptorReadBytes                 float64 `json:"ReadBufferFromFileDescriptorReadBytes"`
	WriteBufferFromFileDescriptorWrite                    float64 `json:"WriteBufferFromFileDescriptorWrite"`
	WriteBufferFromFileDescriptorWriteBytes               float64 `json:"WriteBufferFromFileDescriptorWriteBytes"`
	FileSync                                              float64 `json:"FileSync"`
	FileSyncElapsedMicroseconds                           float64 `json:"FileSyncElapsedMicroseconds"`
	ReadCompressedBytes                                   float64 `json:"ReadCompressedBytes"`
	CompressedReadBufferBlocks                            float64 `json:"CompressedReadBufferBlocks"`
	CompressedReadBufferBytes                             float64 `json:"CompressedReadBufferBytes"`
	OpenedFileCacheHits                                   float64 `json:"OpenedFileCacheHits"`
	OpenedFileCacheMisses                                 float64 `json:"OpenedFileCacheMisses"`
	OpenedFileCacheMicroseconds                           float64 `json:"OpenedFileCacheMicroseconds"`
	IOBufferAllocs                                        float64 `json:"IOBufferAllocs"`
	IOBufferAllocBytes                                    float64 `json:"IOBufferAllocBytes"`
	ArenaAllocChunks                                      float64 `json:"ArenaAllocChunks"`
	ArenaAllocBytes                                       float64 `json:"ArenaAllocBytes"`
	FunctionExecute                                       float64 `json:"FunctionExecute"`
	TableFunctionExecute                                  float64 `json:"TableFunctionExecute"`
	CreatedReadBufferOrdinary                             float64 `json:"CreatedReadBufferOrdinary"`
	DiskReadElapsedMicroseconds                           float64 `json:"DiskReadElapsedMicroseconds"`
	DiskWriteElapsedMicroseconds                          float64 `json:"DiskWriteElapsedMicroseconds"`
	NetworkReceiveElapsedMicroseconds                     float64 `json:"NetworkReceiveElapsedMicroseconds"`
	NetworkSendElapsedMicroseconds                        float64 `json:"NetworkSendElapsedMicroseconds"`
	NetworkReceiveBytes                                   float64 `json:"NetworkReceiveBytes"`
	NetworkSendBytes                                      float64 `json:"NetworkSendBytes"`
	GlobalThreadPoolExpansions                            float64 `json:"GlobalThreadPoolExpansions"`
	GlobalThreadPoolThreadCreationMicroseconds            float64 `json:"GlobalThreadPoolThreadCreationMicroseconds"`
	GlobalThreadPoolLockWaitMicroseconds                  float64 `json:"GlobalThreadPoolLockWaitMicroseconds"`
	GlobalThreadPoolJobs                                  float64 `json:"GlobalThreadPoolJobs"`
	GlobalThreadPoolJobWaitTimeMicroseconds               float64 `json:"GlobalThreadPoolJobWaitTimeMicroseconds"`
	LocalThreadPoolExpansions                             float64 `json:"LocalThreadPoolExpansions"`
	LocalThreadPoolShrinks                                float64 `json:"LocalThreadPoolShrinks"`
	LocalThreadPoolThreadCreationMicroseconds             float64 `json:"LocalThreadPoolThreadCreationMicroseconds"`
	LocalThreadPoolLockWaitMicroseconds                   float64 `json:"LocalThreadPoolLockWaitMicroseconds"`
	LocalThreadPoolJobs                                   float64 `json:"LocalThreadPoolJobs"`
	LocalThreadPoolBusyMicroseconds                       float64 `json:"LocalThreadPoolBusyMicroseconds"`
	LocalThreadPoolJobWaitTimeMicroseconds                float64 `json:"LocalThreadPoolJobWaitTimeMicroseconds"`
	InsertedRows                                          float64 `json:"InsertedRows"`
	InsertedBytes                                         float64 `json:"InsertedBytes"`
	CompileFunction                                       float64 `json:"CompileFunction"`
	CompileExpressionsMicroseconds                        float64 `json:"CompileExpressionsMicroseconds"`
	CompileExpressionsBytes                               float64 `json:"CompileExpressionsBytes"`
	ExternalProcessingFilesTotal                          float64 `json:"ExternalProcessingFilesTotal"`
	JoinBuildTableRowCount                                float64 `json:"JoinBuildTableRowCount"`
	JoinProbeTableRowCount                                float64 `json:"JoinProbeTableRowCount"`
	JoinResultRowCount                                    float64 `json:"JoinResultRowCount"`
	SelectedRows                                          float64 `json:"SelectedRows"`
	SelectedBytes                                         float64 `json:"SelectedBytes"`
	RowsReadByMainReader                                  float64 `json:"RowsReadByMainReader"`
	LoadedDataParts                                       float64 `json:"LoadedDataParts"`
	LoadedDataPartsMicroseconds                           float64 `json:"LoadedDataPartsMicroseconds"`
	WaitMarksLoadMicroseconds                             float64 `json:"WaitMarksLoadMicroseconds"`
	LoadedMarksFiles                                      float64 `json:"LoadedMarksFiles"`
	LoadedMarksCount                                      float64 `json:"LoadedMarksCount"`
	LoadedMarksMemoryBytes                                float64 `json:"LoadedMarksMemoryBytes"`
	Merge                                                 float64 `json:"Merge"`
	MergeSourceParts                                      float64 `json:"MergeSourceParts"`
	MergedRows                                            float64 `json:"MergedRows"`
	MergedColumns                                         float64 `json:"MergedColumns"`
	GatheredColumns                                       float64 `json:"GatheredColumns"`
	MergedUncompressedBytes                               float64 `json:"MergedUncompressedBytes"`
	MergeTotalMilliseconds                                float64 `json:"MergeTotalMilliseconds"`
	MergeExecuteMilliseconds                              float64 `json:"MergeExecuteMilliseconds"`
	MergeHorizontalStageTotalMilliseconds                 float64 `json:"MergeHorizontalStageTotalMilliseconds"`
	MergeHorizontalStageExecuteMilliseconds               float64 `json:"MergeHorizontalStageExecuteMilliseconds"`
	MergeVerticalStageTotalMilliseconds                   float64 `json:"MergeVerticalStageTotalMilliseconds"`
	MergeVerticalStageExecuteMilliseconds                 float64 `json:"MergeVerticalStageExecuteMilliseconds"`
	MergeProjectionStageTotalMilliseconds                 float64 `json:"MergeProjectionStageTotalMilliseconds"`
	MergeProjectionStageExecuteMilliseconds               float64 `json:"MergeProjectionStageExecuteMilliseconds"`
	MergingSortedMilliseconds                             float64 `json:"MergingSortedMilliseconds"`
	GatheringColumnMilliseconds                           float64 `json:"GatheringColumnMilliseconds"`
	MergeTreeDataWriterRows                               float64 `json:"MergeTreeDataWriterRows"`
	MergeTreeDataWriterUncompressedBytes                  float64 `json:"MergeTreeDataWriterUncompressedBytes"`
	MergeTreeDataWriterCompressedBytes                    float64 `json:"MergeTreeDataWriterCompressedBytes"`
	MergeTreeDataWriterBlocks                             float64 `json:"MergeTreeDataWriterBlocks"`
	MergeTreeDataWriterBlocksAlreadySorted                float64 `json:"MergeTreeDataWriterBlocksAlreadySorted"`
	MergeTreeDataWriterSortingBlocksMicroseconds          float64 `json:"MergeTreeDataWriterSortingBlocksMicroseconds"`
	MergeTreeDataWriterMergingBlocksMicroseconds          float64 `json:"MergeTreeDataWriterMergingBlocksMicroseconds"`
	InsertedWideParts                                     float64 `json:"InsertedWideParts"`
	InsertedCompactParts                                  float64 `json:"InsertedCompactParts"`
	MergedIntoWideParts                                   float64 `json:"MergedIntoWideParts"`
	MergedIntoCompactParts                                float64 `json:"MergedIntoCompactParts"`
	ContextLock                                           float64 `json:"ContextLock"`
	ContextLockWaitMicroseconds                           float64 `json:"ContextLockWaitMicroseconds"`
	SystemLogErrorOnFlush                                 float64 `json:"SystemLogErrorOnFlush"`
	RWLockAcquiredReadLocks                               float64 `json:"RWLockAcquiredReadLocks"`
	RWLockReadersWaitMilliseconds                         float64 `json:"RWLockReadersWaitMilliseconds"`
	PartsLockHoldMicroseconds                             float64 `json:"PartsLockHoldMicroseconds"`
	PartsLockWaitMicroseconds                             float64 `json:"PartsLockWaitMicroseconds"`
	RealTimeMicroseconds                                  float64 `json:"RealTimeMicroseconds"`
	UserTimeMicroseconds                                  float64 `json:"UserTimeMicroseconds"`
	SystemTimeMicroseconds                                float64 `json:"SystemTimeMicroseconds"`
	SoftPageFaults                                        float64 `json:"SoftPageFaults"`
	HardPageFaults                                        float64 `json:"HardPageFaults"`
	OSCPUWaitMicroseconds                                 float64 `json:"OSCPUWaitMicroseconds"`
	OSCPUVirtualTimeMicroseconds                          float64 `json:"OSCPUVirtualTimeMicroseconds"`
	OSReadBytes                                           float64 `json:"OSReadBytes"`
	OSWriteBytes                                          float64 `json:"OSWriteBytes"`
	OSReadChars                                           float64 `json:"OSReadChars"`
	OSWriteChars                                          float64 `json:"OSWriteChars"`
	QueryProfilerSignalOverruns                           float64 `json:"QueryProfilerSignalOverruns"`
	QueryProfilerRuns                                     float64 `json:"QueryProfilerRuns"`
	QueryMemoryLimitExceeded                              float64 `json:"QueryMemoryLimitExceeded"`
	ThreadPoolReaderPageCacheHitElapsedMicroseconds       float64 `json:"ThreadPoolReaderPageCacheHitElapsedMicroseconds"`
	ThreadPoolReaderPageCacheMiss                         float64 `json:"ThreadPoolReaderPageCacheMiss"`
	ThreadPoolReaderPageCacheMissBytes                    float64 `json:"ThreadPoolReaderPageCacheMissBytes"`
	ThreadPoolReaderPageCacheMissElapsedMicroseconds      float64 `json:"ThreadPoolReaderPageCacheMissElapsedMicroseconds"`
	SynchronousReadWaitMicroseconds                       float64 `json:"SynchronousReadWaitMicroseconds"`
	MainConfigLoads                                       float64 `json:"MainConfigLoads"`
	ServerStartupMilliseconds                             float64 `json:"ServerStartupMilliseconds"`
	MergerMutatorsGetPartsForMergeElapsedMicroseconds     float64 `json:"MergerMutatorsGetPartsForMergeElapsedMicroseconds"`
	MergerMutatorPrepareRangesForMergeElapsedMicroseconds float64 `json:"MergerMutatorPrepareRangesForMergeElapsedMicroseconds"`
	MergerMutatorSelectPartsForMergeElapsedMicroseconds   float64 `json:"MergerMutatorSelectPartsForMergeElapsedMicroseconds"`
	MergerMutatorRangesForMergeCount                      float64 `json:"MergerMutatorRangesForMergeCount"`
	MergerMutatorPartsInRangesForMergeCount               float64 `json:"MergerMutatorPartsInRangesForMergeCount"`
	MergerMutatorSelectRangePartsCount                    float64 `json:"MergerMutatorSelectRangePartsCount"`
	AsyncLoaderWaitMicroseconds                           float64 `json:"AsyncLoaderWaitMicroseconds"`
	LogTrace                                              float64 `json:"LogTrace"`
	LogDebug                                              float64 `json:"LogDebug"`
	LogInfo                                               float64 `json:"LogInfo"`
	LogWarning                                            float64 `json:"LogWarning"`
	LogError                                              float64 `json:"LogError"`
	LoggerElapsedNanoseconds                              float64 `json:"LoggerElapsedNanoseconds"`
	InterfaceHTTPSendBytes                                float64 `json:"InterfaceHTTPSendBytes"`
	InterfaceHTTPReceiveBytes                             float64 `json:"InterfaceHTTPReceiveBytes"`
	InterfaceNativeSendBytes                              float64 `json:"InterfaceNativeSendBytes"`
	InterfaceNativeReceiveBytes                           float64 `json:"InterfaceNativeReceiveBytes"`
	ConcurrencyControlSlotsGranted                        float64 `json:"ConcurrencyControlSlotsGranted"`
	ConcurrencyControlSlotsAcquired                       float64 `json:"ConcurrencyControlSlotsAcquired"`
	GWPAsanAllocateFailed                                 float64 `json:"GWPAsanAllocateFailed"`
	MemoryWorkerRun                                       float64 `json:"MemoryWorkerRun"`
	MemoryWorkerRunElapsedMicroseconds                    float64 `json:"MemoryWorkerRunElapsedMicroseconds"`
}

func metricName(in string) string {
	// out := toSnake(in)
	// return strings.Replace(out, ".", "_", -1)
	return strings.Replace(in, ".", "_", -1)
}

func parseNumber(s string) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return v, nil
}

func handleResponse(uri string) ([]byte, error) {
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	if user != "" && password != "" {
		req.Header.Set("X-ClickHouse-User", user)
		req.Header.Set("X-ClickHouse-Key", password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error scraping clickhouse: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error().Err(err).Msg("can't close resp.Body")
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		if err != nil {
			data = []byte(err.Error())
		}
		return nil, fmt.Errorf("status %s (%d): %s", resp.Status, resp.StatusCode, data)
	}

	return data, nil
}
// Final commit for clickhouse_exporter/internal/metrics/utils.go
