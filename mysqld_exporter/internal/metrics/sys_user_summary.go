package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	sysSchema           = "sys"
	sysUserSummaryQuery = `
	SELECT
		user,
		statements,
		statement_latency,
		table_scans,
		file_ios,
		file_io_latency,
		current_connections,
		total_connections,
		unique_hosts,
		current_memory,
		total_memory_allocated
	FROM
		` + sysSchema + `.x$user_summary
`
	queryResult = `MySQL [(none)]> SELECT * from  sys.x$user_summary\G;
*************************** 1. row ***************************
                  user: root
            statements: 134
     statement_latency: 246341790000
 statement_avg_latency: 1838371567.1642
           table_scans: 6
              file_ios: 128
       file_io_latency: 103067496576
   current_connections: 1
     total_connections: 6
          unique_hosts: 2
        current_memory: 3792202
total_memory_allocated: 12966106
*************************** 2. row ***************************
                  user: background
            statements: 0
     statement_latency: 0
 statement_avg_latency: 0.0000
           table_scans: 0
              file_ios: 1535
       file_io_latency: 4232918471212
   current_connections: 36
     total_connections: 50
          unique_hosts: 0
        current_memory: 2203338
total_memory_allocated: 69262695
*************************** 3. row ***************************
                  user: event_scheduler
            statements: 0
     statement_latency: 0
 statement_avg_latency: 0.0000
           table_scans: 0
              file_ios: 0
       file_io_latency: 0
   current_connections: 1
     total_connections: 1
          unique_hosts: 1
        current_memory: 16665
total_memory_allocated: 16665
3 rows in set (0.008 sec)`
)

func init() {
	exporter.Register(
		NewSysUserSummary())
}

type SysUserSummary struct {
	instance mysql.Instance
	sysUserSummaryStatements
	sysUserSummaryStatementLatency
	sysUserSummaryTableScans
	sysUserSummaryFileIos
	sysUserSummaryFileIoLatency
	sysUserSummaryCurrentConnections
	sysUserSummaryTotalConnections
	sysUserSummaryUniqueHosts
	sysUserSummaryCurrentMemory
	sysUserSummaryTotalMemory
}

func NewSysUserSummary() *SysUserSummary {
	return &SysUserSummary{
		//instance:                         instance,
		sysUserSummaryStatements:         *newSysUserSummaryStatements(),
		sysUserSummaryStatementLatency:   *newSysUserSummaryStatementLatency(),
		sysUserSummaryTableScans:         *newSysUserSummaryTableScans(),
		sysUserSummaryFileIos:            *newSysUserSummaryFileIos(),
		sysUserSummaryFileIoLatency:      *newSysUserSummaryFileIoLatency(),
		sysUserSummaryCurrentConnections: *newSysUserSummaryCurrentConnections(),
		sysUserSummaryTotalConnections:   *newSysUserSummaryTotalConnections(),
		sysUserSummaryUniqueHosts:        *newSysUserSummaryUniqueHosts(),
		sysUserSummaryCurrentMemory:      *newSysUserSummaryCurrentMemory(),
		sysUserSummaryTotalMemory:        *newSysUserSummaryTotalMemory(),
		//sysUserSummaryFileIoLatency:    *newSysUserSummaryFileIoLatency(),
	}
}

func (qd *SysUserSummary) Collect(ch chan<- prometheus.Metric) {
	logrus.Info("Start collecting SlaveStatus metrics")
	qd.instance = *GetInstance()
	db := qd.instance.GetDB()
	userSummaryRows, err := db.Query(sysUserSummaryQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer userSummaryRows.Close()
	var (
		user                   string
		statements             uint64
		statement_latency      float64
		table_scans            uint64
		file_ios               uint64
		file_io_latency        float64
		current_connections    uint64
		total_connections      uint64
		unique_hosts           uint64
		current_memory         uint64
		total_memory_allocated uint64
	)
	for userSummaryRows.Next() {
		err = userSummaryRows.Scan(
			&user,
			&statements,
			&statement_latency,
			&table_scans,
			&file_ios,
			&file_io_latency,
			&current_connections,
			&total_connections,
			&unique_hosts,
			&current_memory,
			&total_memory_allocated)
		if err != nil {
			logrus.Error(err)
			continue
		}
		qd.sysUserSummaryStatements.Collect(ch,
			float64(statements),
			[]string{user})
		qd.sysUserSummaryStatementLatency.Collect(ch,
			statement_latency,
			[]string{user})
		qd.sysUserSummaryTableScans.Collect(ch,
			float64(table_scans),
			[]string{user})
		qd.sysUserSummaryFileIos.Collect(ch,
			float64(file_ios),
			[]string{user})
		qd.sysUserSummaryFileIoLatency.Collect(ch,
			file_io_latency,
			[]string{user})
		qd.sysUserSummaryCurrentConnections.Collect(ch,
			float64(current_connections),
			[]string{user})
		qd.sysUserSummaryTotalConnections.Collect(ch,
			float64(total_connections),
			[]string{user})
		qd.sysUserSummaryUniqueHosts.Collect(ch,
			float64(unique_hosts),
			[]string{user})
		qd.sysUserSummaryCurrentMemory.Collect(ch,
			float64(current_memory),
			[]string{user})
		qd.sysUserSummaryTotalMemory.Collect(ch,
			float64(total_memory_allocated),
			[]string{user})
	}
}

type sysUserSummaryStatements struct {
	*baseMetrics
}

func newSysUserSummaryStatements() *sysUserSummaryStatements {
	return &sysUserSummaryStatements{
		NewMetrics(
			"mysql_sys_statements_total",
			"The total number of statements for the user",
			[]string{"user"})}
}

func (qd *sysUserSummaryStatements) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryStatementLatency struct {
	*baseMetrics
}

func newSysUserSummaryStatementLatency() *sysUserSummaryStatementLatency {
	return &sysUserSummaryStatementLatency{
		NewMetrics(
			"mysql_sys_statement_latency",
			"The total wait time of timed statements for the user",
			[]string{"user"})}
}

func (qd *sysUserSummaryStatementLatency) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryTableScans struct {
	*baseMetrics
}

func newSysUserSummaryTableScans() *sysUserSummaryTableScans {
	return &sysUserSummaryTableScans{
		NewMetrics(
			"mysql_sys_table_scans_total",
			"The total number of table scans for the user",
			[]string{"user"})}
}

func (qd *sysUserSummaryTableScans) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryFileIos struct {
	*baseMetrics
}

func newSysUserSummaryFileIos() *sysUserSummaryFileIos {
	return &sysUserSummaryFileIos{
		NewMetrics(
			"mysql_sys_file_ios_total",
			"The total number of file ios for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryFileIos) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryFileIoLatency struct {
	*baseMetrics
}

func newSysUserSummaryFileIoLatency() *sysUserSummaryFileIoLatency {
	return &sysUserSummaryFileIoLatency{
		NewMetrics(
			"mysql_sys_file_io_latency",
			"The total wait time of file ios for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryFileIoLatency) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryCurrentConnections struct {
	*baseMetrics
}

func newSysUserSummaryCurrentConnections() *sysUserSummaryCurrentConnections {
	return &sysUserSummaryCurrentConnections{
		NewMetrics(
			"mysql_sys_current_connections_total",
			"The total number of current connections for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryCurrentConnections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryTotalConnections struct {
	*baseMetrics
}

func newSysUserSummaryTotalConnections() *sysUserSummaryTotalConnections {
	return &sysUserSummaryTotalConnections{
		NewMetrics(
			"mysql_sys_total_connections_total",
			"The total number of total connections for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryTotalConnections) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryUniqueHosts struct {
	*baseMetrics
}

func newSysUserSummaryUniqueHosts() *sysUserSummaryUniqueHosts {
	return &sysUserSummaryUniqueHosts{
		NewMetrics(
			"mysql_sys_unique_hosts_total",
			"The total number of unique hosts for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryUniqueHosts) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryCurrentMemory struct {
	*baseMetrics
}

func newSysUserSummaryCurrentMemory() *sysUserSummaryCurrentMemory {
	return &sysUserSummaryCurrentMemory{
		NewMetrics(
			"mysql_sys_current_memory_bytes",
			"The total number of current memory for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryCurrentMemory) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type sysUserSummaryTotalMemory struct {
	*baseMetrics
}

func newSysUserSummaryTotalMemory() *sysUserSummaryTotalMemory {
	return &sysUserSummaryTotalMemory{
		NewMetrics(
			"mysql_sys_total_memory_bytes",
			"The total number of total memory for the user",
			[]string{"user"})}
}
func (qd *sysUserSummaryTotalMemory) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
// Part 2 commit for mysqld_exporter/internal/metrics/sys_user_summary.go
