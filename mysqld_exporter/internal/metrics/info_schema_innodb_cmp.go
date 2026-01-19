package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
)

const (
	innodbCmpQuery = `
		SELECT
		  page_size, compress_ops, compress_ops_ok, compress_time, uncompress_ops, uncompress_time
		  FROM information_schema.innodb_cmp
		`
	innodbCmpResult = `
*************************** 1. row ***************************
      page_size: 1024
   compress_ops: 0
compress_ops_ok: 0
  compress_time: 0
 uncompress_ops: 0
uncompress_time: 0
*************************** 2. row ***************************
      page_size: 2048
   compress_ops: 0
compress_ops_ok: 0
  compress_time: 0
 uncompress_ops: 0
uncompress_time: 0
*************************** 3. row ***************************
      page_size: 4096
   compress_ops: 0
compress_ops_ok: 0
  compress_time: 0
 uncompress_ops: 0
uncompress_time: 0
*************************** 4. row ***************************
      page_size: 8192
   compress_ops: 0
compress_ops_ok: 0
  compress_time: 0
 uncompress_ops: 0
uncompress_time: 0
*************************** 5. row ***************************
      page_size: 16384
   compress_ops: 0
compress_ops_ok: 0
  compress_time: 0
 uncompress_ops: 0
uncompress_time: 0
5 rows in set (0.001 sec)

		`
)

type ScrapeInnodbCmp struct {
	instance mysql.Instance
	infoSchemaInnodbCmpCompressOps
	infoSchemaInnodbCmpCompressOpsOk
	infoSchemaInnodbCmpCompressTime
	infoSchemaInnodbCmpUncompressOps
	infoSchemaInnodbCmpUncompressTime
}

func init() {
	exporter.Register(
		NewScrapeInnodbCmp())
}
func NewScrapeInnodbCmp() *ScrapeInnodbCmp {
	return &ScrapeInnodbCmp{
		//instance:                          instance,
		infoSchemaInnodbCmpCompressOps:    *NewinfoSchemaInnodbCmpCompressOps(),
		infoSchemaInnodbCmpCompressOpsOk:  *NewinfoSchemaInnodbCmpCompressOpsOk(),
		infoSchemaInnodbCmpCompressTime:   *NewinfoSchemaInnodbCmpCompressTime(),
		infoSchemaInnodbCmpUncompressOps:  *NewinfoSchemaInnodbCmpUncompressOps(),
		infoSchemaInnodbCmpUncompressTime: *NewinfoSchemaInnodbCmpUncompressTime(),
	}
}

func (qd ScrapeInnodbCmp) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	informationSchemaInnodbCmpRows, err := db.Query(innodbCmpQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer informationSchemaInnodbCmpRows.Close()
	var (
		page_size       string
		compress_ops    float64
		compress_ops_ok float64
		compress_time   float64
		uncompress_ops  float64
		uncompress_time float64
	)
	for informationSchemaInnodbCmpRows.Next() {
		err := informationSchemaInnodbCmpRows.Scan(
			&page_size,
			&compress_ops,
			&compress_ops_ok,
			&compress_time,
			&uncompress_ops,
			&uncompress_time,
		)
		if err != nil {
			logrus.Error(err)
			return
		}
		qd.infoSchemaInnodbCmpCompressOps.Collect(ch,
			compress_ops,
			[]string{
				page_size,
			})
		qd.infoSchemaInnodbCmpCompressOpsOk.Collect(ch,
			compress_ops_ok,
			[]string{
				page_size,
			})

		qd.infoSchemaInnodbCmpCompressTime.Collect(ch,
			compress_time,
			[]string{
				page_size,
			})
		qd.infoSchemaInnodbCmpUncompressOps.Collect(ch,
			uncompress_ops,
			[]string{
				page_size,
			})
		qd.infoSchemaInnodbCmpUncompressTime.Collect(ch,
			uncompress_time,
			[]string{
				page_size,
			})
	}
	return
}

type infoSchemaInnodbCmpCompressOps struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpCompressOps() *infoSchemaInnodbCmpCompressOps {
	return &infoSchemaInnodbCmpCompressOps{
		NewMetrics(
			"info_schema_innodb_cmp_compress_ops_total",
			"Number of times a B-tree page of the size PAGE_SIZE has been compressed.",
			[]string{
				"page_size",
			})}
}
func (qd *infoSchemaInnodbCmpCompressOps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpCompressOpsOk struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpCompressOpsOk() *infoSchemaInnodbCmpCompressOpsOk {
	return &infoSchemaInnodbCmpCompressOpsOk{
		NewMetrics(
			"info_schema_innodb_cmp_compress_ops_ok_total",
			"Number of times a B-tree page of the size PAGE_SIZE has been successfully compressed.",
			[]string{
				"page_size",
			})}
}
func (qd *infoSchemaInnodbCmpCompressOpsOk) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpCompressTime struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpCompressTime() *infoSchemaInnodbCmpCompressTime {
	return &infoSchemaInnodbCmpCompressTime{
		NewMetrics(
			"info_schema_innodb_cmp_compress_time_total",
			"Total time spent in microseconds compressing a B-tree page of the size PAGE_SIZE.",
			[]string{
				"page_size",
			})}
}
func (qd *infoSchemaInnodbCmpCompressTime) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpUncompressOps struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpUncompressOps() *infoSchemaInnodbCmpUncompressOps {
	return &infoSchemaInnodbCmpUncompressOps{
		NewMetrics(
			"info_schema_innodb_cmp_uncompress_ops_total",
			"Number of times a B-tree page of the size PAGE_SIZE has been uncompressed.",
			[]string{
				"page_size",
			})}
}
func (qd *infoSchemaInnodbCmpUncompressOps) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoSchemaInnodbCmpUncompressTime struct {
	*baseMetrics
}

func NewinfoSchemaInnodbCmpUncompressTime() *infoSchemaInnodbCmpUncompressTime {
	return &infoSchemaInnodbCmpUncompressTime{
		NewMetrics(
			"info_schema_innodb_cmp_uncompress_time_total",
			"Total time spent in microseconds uncompressing a B-tree page of the size PAGE_SIZE.",
			[]string{
				"page_size",
			})}
}
func (qd *infoSchemaInnodbCmpUncompressTime) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
