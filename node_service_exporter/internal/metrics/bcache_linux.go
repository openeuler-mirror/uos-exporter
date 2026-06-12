//go:build !nobcache && linux
// +build !nobcache,linux

package metrics

import (
	"fmt"
	"node_service_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs/bcache"
)

// Enable priority stats by default for compatibility
var priorityStats = true

func init() {
	exporter.Register(NewBcacheAverageKeySize())
	exporter.Register(NewBcacheBtreeCacheSize())
	exporter.Register(NewBcacheCacheAvailablePercent())
	exporter.Register(NewBcacheCongested())
	exporter.Register(NewBcacheRootUsagePercent())
	exporter.Register(NewBcacheTreeDepth())
	exporter.Register(NewBcacheActiveJournalEntries())
	exporter.Register(NewBcacheBtreeNodes())
	exporter.Register(NewBcacheBtreeReadAverageDuration())
	exporter.Register(NewBcacheCacheReadRaces())
	exporter.Register(NewBcacheDirtyData())
	exporter.Register(NewBcacheDirtyTarget())
	exporter.Register(NewBcacheWritebackRate())
	exporter.Register(NewBcacheWritebackRateProportional())
	exporter.Register(NewBcacheWritebackRateIntegral())
	exporter.Register(NewBcacheWritebackChange())
	exporter.Register(NewBcacheBypassedBytes())
	exporter.Register(NewBcacheCacheHits())
	exporter.Register(NewBcacheCacheMisses())
	exporter.Register(NewBcacheCacheBypassHits())
	exporter.Register(NewBcacheCacheBypassMisses())
	exporter.Register(NewBcacheCacheMissCollisions())
	exporter.Register(NewBcacheCacheReadaheads())
	exporter.Register(NewBcacheIOErrors())
	exporter.Register(NewBcacheMetadataWritten())
	exporter.Register(NewBcacheWritten())
	exporter.Register(NewBcachePriorityStatsUnused())
	exporter.Register(NewBcachePriorityStatsMetadata())
}

var bcacheFS bcache.FS
var bcacheInitialized bool

func initBcacheFS() error {
	if !bcacheInitialized {
		fs, err := bcache.NewFS("/sys")
		if err != nil {
			return fmt.Errorf("failed to open sysfs: %w", err)
		}
		bcacheFS = fs
		bcacheInitialized = true
	}
	return nil
}

// Helper function to get bcache stats
func getBcacheStats() ([]*bcache.Stats, error) {
	if err := initBcacheFS(); err != nil {
		return nil, err
	}

	if priorityStats {
		return bcacheFS.Stats()
	}
	return bcacheFS.StatsWithoutPriority()
}

// Bcache Average Key Size
type BcacheAverageKeySize struct {
	*baseMetrics
}

func NewBcacheAverageKeySize() *BcacheAverageKeySize {
	return &BcacheAverageKeySize{
		NewMetrics("node_bcache_average_key_size_sectors",
			"Average data per key in the btree (sectors).",
			[]string{"uuid"}),
	}
}

func (b *BcacheAverageKeySize) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.AverageKeySize), []string{s.Name})
	}
}

// Bcache Btree Cache Size
type BcacheBtreeCacheSize struct {
	*baseMetrics
}

func NewBcacheBtreeCacheSize() *BcacheBtreeCacheSize {
	return &BcacheBtreeCacheSize{
		NewMetrics("node_bcache_btree_cache_size_bytes",
			"Amount of memory currently used by the btree cache.",
			[]string{"uuid"}),
	}
}

func (b *BcacheBtreeCacheSize) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.BtreeCacheSize), []string{s.Name})
	}
}

// Bcache Cache Available Percent
type BcacheCacheAvailablePercent struct {
	*baseMetrics
}

func NewBcacheCacheAvailablePercent() *BcacheCacheAvailablePercent {
	return &BcacheCacheAvailablePercent{
		NewMetrics("node_bcache_cache_available_percent",
			"Percentage of cache device without dirty data, usable for writeback (may contain clean cached data).",
			[]string{"uuid"}),
	}
}

func (b *BcacheCacheAvailablePercent) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.CacheAvailablePercent), []string{s.Name})
	}
}

// Bcache Congested
type BcacheCongested struct {
	*baseMetrics
}

func NewBcacheCongested() *BcacheCongested {
	return &BcacheCongested{
		NewMetrics("node_bcache_congested",
			"Congestion.",
			[]string{"uuid"}),
	}
}

func (b *BcacheCongested) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.Congested), []string{s.Name})
	}
}

// Bcache Root Usage Percent
type BcacheRootUsagePercent struct {
	*baseMetrics
}

func NewBcacheRootUsagePercent() *BcacheRootUsagePercent {
	return &BcacheRootUsagePercent{
		NewMetrics("node_bcache_root_usage_percent",
			"Percentage of the root btree node in use (tree depth increases if too high).",
			[]string{"uuid"}),
	}
}

func (b *BcacheRootUsagePercent) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.RootUsagePercent), []string{s.Name})
	}
}

// Bcache Tree Depth
type BcacheTreeDepth struct {
	*baseMetrics
}

func NewBcacheTreeDepth() *BcacheTreeDepth {
	return &BcacheTreeDepth{
		NewMetrics("node_bcache_tree_depth",
			"Depth of the btree.",
			[]string{"uuid"}),
	}
}

func (b *BcacheTreeDepth) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.TreeDepth), []string{s.Name})
	}
}

// Bcache Active Journal Entries
type BcacheActiveJournalEntries struct {
	*baseMetrics
}

func NewBcacheActiveJournalEntries() *BcacheActiveJournalEntries {
	return &BcacheActiveJournalEntries{
		NewMetrics("node_bcache_active_journal_entries",
			"Number of journal entries that are newer than the index.",
			[]string{"uuid"}),
	}
}

func (b *BcacheActiveJournalEntries) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.Internal.ActiveJournalEntries), []string{s.Name})
	}
}

// Bcache Btree Nodes
type BcacheBtreeNodes struct {
	*baseMetrics
}

func NewBcacheBtreeNodes() *BcacheBtreeNodes {
	return &BcacheBtreeNodes{
		NewMetrics("node_bcache_btree_nodes",
			"Total nodes in the btree.",
			[]string{"uuid"}),
	}
}

func (b *BcacheBtreeNodes) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collect(ch, float64(s.Bcache.Internal.BtreeNodes), []string{s.Name})
	}
}

// Bcache Btree Read Average Duration
type BcacheBtreeReadAverageDuration struct {
	*baseMetrics
}

func NewBcacheBtreeReadAverageDuration() *BcacheBtreeReadAverageDuration {
	return &BcacheBtreeReadAverageDuration{
		NewMetrics("node_bcache_btree_read_average_duration_seconds",
			"Average btree read duration.",
			[]string{"uuid"}),
	}
}

func (b *BcacheBtreeReadAverageDuration) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		duration := float64(s.Bcache.Internal.BtreeReadAverageDurationNanoSeconds) * 1e-9
		b.baseMetrics.collect(ch, duration, []string{s.Name})
	}
}

// Bcache Cache Read Races
type BcacheCacheReadRaces struct {
	*baseMetrics
}

func NewBcacheCacheReadRaces() *BcacheCacheReadRaces {
	return &BcacheCacheReadRaces{
		NewMetrics("node_bcache_cache_read_races_total",
			"Counts instances where while data was being read from the cache, the bucket was reused and invalidated - i.e. where the pointer was stale after the read completed.",
			[]string{"uuid"}),
	}
}

func (b *BcacheCacheReadRaces) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		b.baseMetrics.collectCounter(ch, float64(s.Bcache.Internal.CacheReadRaces), []string{s.Name})
	}
}

// Bcache Dirty Data
type BcacheDirtyData struct {
	*baseMetrics
}

func NewBcacheDirtyData() *BcacheDirtyData {
	return &BcacheDirtyData{
		NewMetrics("node_bcache_dirty_data_bytes",
			"Amount of dirty data for this backing device in the cache.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheDirtyData) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.DirtyData), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Dirty Target
type BcacheDirtyTarget struct {
	*baseMetrics
}

func NewBcacheDirtyTarget() *BcacheDirtyTarget {
	return &BcacheDirtyTarget{
		NewMetrics("node_bcache_dirty_target_bytes",
			"Current dirty data target threshold for this backing device in bytes.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheDirtyTarget) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.WritebackRateDebug.Target), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Writeback Rate
type BcacheWritebackRate struct {
	*baseMetrics
}

func NewBcacheWritebackRate() *BcacheWritebackRate {
	return &BcacheWritebackRate{
		NewMetrics("node_bcache_writeback_rate",
			"Current writeback rate for this backing device in bytes.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheWritebackRate) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.WritebackRateDebug.Rate), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Writeback Rate Proportional
type BcacheWritebackRateProportional struct {
	*baseMetrics
}

func NewBcacheWritebackRateProportional() *BcacheWritebackRateProportional {
	return &BcacheWritebackRateProportional{
		NewMetrics("node_bcache_writeback_rate_proportional_term",
			"Current result of proportional controller, part of writeback rate",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheWritebackRateProportional) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.WritebackRateDebug.Proportional), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Writeback Rate Integral
type BcacheWritebackRateIntegral struct {
	*baseMetrics
}

func NewBcacheWritebackRateIntegral() *BcacheWritebackRateIntegral {
	return &BcacheWritebackRateIntegral{
		NewMetrics("node_bcache_writeback_rate_integral_term",
			"Current result of integral controller, part of writeback rate",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheWritebackRateIntegral) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.WritebackRateDebug.Integral), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Writeback Change
type BcacheWritebackChange struct {
	*baseMetrics
}

func NewBcacheWritebackChange() *BcacheWritebackChange {
	return &BcacheWritebackChange{
		NewMetrics("node_bcache_writeback_change",
			"Last writeback rate change step for this backing device.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheWritebackChange) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collect(ch, float64(bdev.WritebackRateDebug.Change), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Bypassed Bytes
type BcacheBypassedBytes struct {
	*baseMetrics
}

func NewBcacheBypassedBytes() *BcacheBypassedBytes {
	return &BcacheBypassedBytes{
		NewMetrics("node_bcache_bypassed_bytes_total",
			"Amount of IO (both reads and writes) that has bypassed the cache.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheBypassedBytes) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.Bypassed), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Hits
type BcacheCacheHits struct {
	*baseMetrics
}

func NewBcacheCacheHits() *BcacheCacheHits {
	return &BcacheCacheHits{
		NewMetrics("node_bcache_cache_hits_total",
			"Hits counted per individual IO as bcache sees them.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheHits) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheHits), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Misses
type BcacheCacheMisses struct {
	*baseMetrics
}

func NewBcacheCacheMisses() *BcacheCacheMisses {
	return &BcacheCacheMisses{
		NewMetrics("node_bcache_cache_misses_total",
			"Misses counted per individual IO as bcache sees them.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheMisses) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheMisses), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Bypass Hits
type BcacheCacheBypassHits struct {
	*baseMetrics
}

func NewBcacheCacheBypassHits() *BcacheCacheBypassHits {
	return &BcacheCacheBypassHits{
		NewMetrics("node_bcache_cache_bypass_hits_total",
			"Hits for IO intended to skip the cache.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheBypassHits) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheBypassHits), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Bypass Misses
type BcacheCacheBypassMisses struct {
	*baseMetrics
}

func NewBcacheCacheBypassMisses() *BcacheCacheBypassMisses {
	return &BcacheCacheBypassMisses{
		NewMetrics("node_bcache_cache_bypass_misses_total",
			"Misses for IO intended to skip the cache.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheBypassMisses) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheBypassMisses), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Miss Collisions
type BcacheCacheMissCollisions struct {
	*baseMetrics
}

func NewBcacheCacheMissCollisions() *BcacheCacheMissCollisions {
	return &BcacheCacheMissCollisions{
		NewMetrics("node_bcache_cache_miss_collisions_total",
			"Instances where data insertion from cache miss raced with write (data already present).",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheMissCollisions) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheMissCollisions), []string{s.Name, bdev.Name})
		}
	}
}

// Bcache Cache Readaheads
type BcacheCacheReadaheads struct {
	*baseMetrics
}

func NewBcacheCacheReadaheads() *BcacheCacheReadaheads {
	return &BcacheCacheReadaheads{
		NewMetrics("node_bcache_cache_readaheads_total",
			"Count of times readahead occurred.",
			[]string{"uuid", "backing_device"}),
	}
}

func (b *BcacheCacheReadaheads) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, bdev := range s.Bdevs {
			if bdev.Total.CacheReadaheads != 0 {
				b.baseMetrics.collectCounter(ch, float64(bdev.Total.CacheReadaheads), []string{s.Name, bdev.Name})
			}
		}
	}
}

// Bcache IO Errors
type BcacheIOErrors struct {
	*baseMetrics
}

func NewBcacheIOErrors() *BcacheIOErrors {
	return &BcacheIOErrors{
		NewMetrics("node_bcache_io_errors",
			"Number of errors that have occurred, decayed by io_error_halflife.",
			[]string{"uuid", "cache_device"}),
	}
}

func (b *BcacheIOErrors) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, cache := range s.Caches {
			b.baseMetrics.collect(ch, float64(cache.IOErrors), []string{s.Name, cache.Name})
		}
	}
}

// Bcache Metadata Written
type BcacheMetadataWritten struct {
	*baseMetrics
}

func NewBcacheMetadataWritten() *BcacheMetadataWritten {
	return &BcacheMetadataWritten{
		NewMetrics("node_bcache_metadata_written_bytes_total",
			"Sum of all non data writes (btree writes and all other metadata).",
			[]string{"uuid", "cache_device"}),
	}
}

func (b *BcacheMetadataWritten) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, cache := range s.Caches {
			b.baseMetrics.collectCounter(ch, float64(cache.MetadataWritten), []string{s.Name, cache.Name})
		}
	}
}

// Bcache Written
type BcacheWritten struct {
	*baseMetrics
}

func NewBcacheWritten() *BcacheWritten {
	return &BcacheWritten{
		NewMetrics("node_bcache_written_bytes_total",
			"Sum of all data that has been written to the cache.",
			[]string{"uuid", "cache_device"}),
	}
}

func (b *BcacheWritten) Collect(ch chan<- prometheus.Metric) {
	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, cache := range s.Caches {
			b.baseMetrics.collectCounter(ch, float64(cache.Written), []string{s.Name, cache.Name})
		}
	}
}

// Bcache Priority Stats Unused
type BcachePriorityStatsUnused struct {
	*baseMetrics
}

func NewBcachePriorityStatsUnused() *BcachePriorityStatsUnused {
	return &BcachePriorityStatsUnused{
		NewMetrics("node_bcache_priority_stats_unused_percent",
			"The percentage of the cache that doesn't contain any data.",
			[]string{"uuid", "cache_device"}),
	}
}

func (b *BcachePriorityStatsUnused) Collect(ch chan<- prometheus.Metric) {
	if !priorityStats {
		return
	}

	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, cache := range s.Caches {
			b.baseMetrics.collect(ch, float64(cache.Priority.UnusedPercent), []string{s.Name, cache.Name})
		}
	}
}

// Bcache Priority Stats Metadata
type BcachePriorityStatsMetadata struct {
	*baseMetrics
}

func NewBcachePriorityStatsMetadata() *BcachePriorityStatsMetadata {
	return &BcachePriorityStatsMetadata{
		NewMetrics("node_bcache_priority_stats_metadata_percent",
			"Bcache's metadata overhead.",
			[]string{"uuid", "cache_device"}),
	}
}

func (b *BcachePriorityStatsMetadata) Collect(ch chan<- prometheus.Metric) {
	if !priorityStats {
		return
	}

	stats, err := getBcacheStats()
	if err != nil {
		return
	}

	for _, s := range stats {
		for _, cache := range s.Caches {
			b.baseMetrics.collect(ch, float64(cache.Priority.MetadataPercent), []string{s.Name, cache.Name})
		}
	}
} 