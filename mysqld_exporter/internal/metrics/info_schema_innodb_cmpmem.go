package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	innodbCmpMemQuery = `
                SELECT
                  page_size, buffer_pool_instance, pages_used, pages_free, relocation_ops, relocation_time
                  FROM information_schema.innodb_cmpmem
                `
	innodbCmpMemQueryResult = `
*************************** 1. row ***************************
           page_size: 1024
buffer_pool_instance: 0
          pages_used: 0
          pages_free: 0
      relocation_ops: 0
     relocation_time: 0
*************************** 2. row ***************************
           page_size: 2048
buffer_pool_instance: 0
          pages_used: 0
          pages_free: 0
      relocation_ops: 0
     relocation_time: 0
*************************** 3. row ***************************
           page_size: 4096
buffer_pool_instance: 0
          pages_used: 0
          pages_free: 0
      relocation_ops: 0
     relocation_time: 0
*************************** 4. row ***************************
           page_size: 8192
buffer_pool_instance: 0
          pages_used: 0
          pages_free: 0
      relocation_ops: 0
     relocation_time: 0
*************************** 5. row ***************************
           page_size: 16384
buffer_pool_instance: 0
          pages_used: 0
          pages_free: 0
      relocation_ops: 0
     relocation_time: 0
5 rows in set (0.001 sec)
`
)

type ScrapeInnodbCmpMem struct {
	instance mysql.Instance
	infoSchemaInnodbCmpMemPagesRead
	infoSchemaInnodbCmpMemPagesFree
	infoSchemaInnodbCmpMemRelocationOps
	infoSchemaInnodbCmpMemRelocationTime
}

func init() {
	exporter.Register(
		NewScrapeInnodbCmpMem())
}
func NewScrapeInnodbCmpMem() *ScrapeInnodbCmpMem {
	return &ScrapeInnodbCmpMem{
		//instance:                             instance,
		infoSchemaInnodbCmpMemPagesRead:      *NewinfoSchemaInnodbCmpMemPagesRead(),
		infoSchemaInnodbCmpMemPagesFree:      *NewinfoSchemaInnodbCmpMemPagesFree(),
		infoSchemaInnodbCmpMemRelocationOps:  *NewinfoSchemaInnodbCmpMemRelocationOps(),
		infoSchemaInnodbCmpMemRelocationTime: *NewinfoSchemaInnodbCmpMemRelocationTime(),
	}
}

func (qd ScrapeInnodbCmpMem) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	informationSchemaInnodbCmpMemRows, err := db.Query(innodbCmpMemQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer informationSchemaInnodbCmpMemRows.Close()
	var (
		page_size       string
		buffer_pool     string
		pages_used      float64
		pages_free      float64
		relocation_ops  float64
		relocation_time float64
	)
	for informationSchemaInnodbCmpMemRows.Next() {
		err = informationSchemaInnodbCmpMemRows.Scan(
			&page_size,
			&buffer_pool,
			&pages_used,
			&pages_free,
			&relocation_ops,
			&relocation_time,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.infoSchemaInnodbCmpMemPagesRead.Collect(ch,
			pages_used,
			[]string{
				page_size,
				buffer_pool,
			})

		qd.infoSchemaInnodbCmpMemPagesFree.Collect(ch,
			pages_free,
			[]string{
				page_size,
				buffer_pool,
			})
		qd.infoSchemaInnodbCmpMemRelocationOps.Collect(ch,
			relocation_ops,
			[]string{
				page_size,
				buffer_pool,
			})

		qd.infoSchemaInnodbCmpMemRelocationTime.Collect(ch,
			(relocation_time / 1000),
			[]string{
				page_size,
				buffer_pool,
			})
	}
}

type infoSchemaInnodbCmpMemPagesRead struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpMemPagesRead() *infoSchemaInnodbCmpMemPagesRead {
	return &infoSchemaInnodbCmpMemPagesRead{
		NewMetrics(
			"info_schema_innodb_cmpmem_pages_used_total",
			"Number of blocks of the size PAGE_SIZE that are currently in use.",
			[]string{
				"page_size",
				"buffer_pool",
			})}
}
func (qd *infoSchemaInnodbCmpMemPagesRead) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpMemPagesFree struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpMemPagesFree() *infoSchemaInnodbCmpMemPagesFree {
	return &infoSchemaInnodbCmpMemPagesFree{
		NewMetrics(
			"info_schema_innodb_cmpmem_pages_free_total",
			"Number of blocks of the size PAGE_SIZE that are currently free.",
			[]string{
				"page_size",
				"buffer_pool",
			})}
}
func (qd *infoSchemaInnodbCmpMemPagesFree) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpMemRelocationOps struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpMemRelocationOps() *infoSchemaInnodbCmpMemRelocationOps {
	return &infoSchemaInnodbCmpMemRelocationOps{
		NewMetrics(
			"info_schema_innodb_cmpmem_relocation_ops_total",
			"Number of relocation operations.",
			[]string{
				"page_size",
				"buffer_pool",
			})}
}
func (qd *infoSchemaInnodbCmpMemRelocationOps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpMemRelocationTime struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpMemRelocationTime() *infoSchemaInnodbCmpMemRelocationTime {
	return &infoSchemaInnodbCmpMemRelocationTime{
		NewMetrics(
			"info_schema_innodb_cmpmem_relocation_time_total",
			"Total time spent in relocation operations.",
			[]string{
				"page_size",
				"buffer_pool",
			})}
}
func (qd *infoSchemaInnodbCmpMemRelocationTime) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
