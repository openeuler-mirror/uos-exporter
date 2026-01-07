package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strconv"
	"strings"
)

const (
	binlog       = "binlog"
	logbinQuery  = `SELECT @@log_bin`
	binlogQuery  = `SHOW BINARY LOGS`
	binlogResult = `
+---------------+-----------+-----------+
| Log_name      | File_size | Encrypted |
+---------------+-----------+-----------+
| binlog.000001 |       894 | No        |
| binlog.000002 |      1078 | No        |
| binlog.000003 |       157 | No        |
+---------------+-----------+-----------+
`
)

type ScrapeBinlogSize struct {
	instance mysql.Instance
	binlogSize
	binlogFiles
	binlogFileNumber
}

func init() {
	exporter.Register(
		NewScrapeBinlogSize())
}
func NewScrapeBinlogSize() *ScrapeBinlogSize {
	return &ScrapeBinlogSize{
		binlogSize:       *NewbinlogSize(),
		binlogFiles:      *NewbinlogFiles(),
		binlogFileNumber: *NewbinlogFileNumber(),
	}
}

func (qd ScrapeBinlogSize) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	var (
		logBin uint8
	)
	err := db.QueryRow(logbinQuery).Scan(&logBin)
	if err != nil {
		logrus.Error(err)
		return
	}
	if logBin == 0 {
		logrus.Debugf("logbin is %d", logBin)
		logrus.Debug("skip collect binlog")
		return
	}
	masterLogRows, err := db.Query(binlogQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer masterLogRows.Close()
	var (
		size      uint64
		count     uint64
		filename  string
		filesize  uint64
		encrypted string
	)
	size = 0
	count = 0

	columns, err := masterLogRows.Columns()
	if err != nil {
		logrus.Error(err)
		return
	}
	columnCount := len(columns)
	for masterLogRows.Next() {
		switch columnCount {
		case 2:
			err := masterLogRows.Scan(&filename, &filesize)
			if err != nil {
				logrus.Error(err)
				return
			}
		case 3:
			err := masterLogRows.Scan(&filename, &filesize, &encrypted)
			if err != nil {
				logrus.Error(err)
				return
			}
		default:
			logrus.Errorf("invalid number of columns: %q", columnCount)
			return
		}

		size += filesize
		count++
	}
	qd.binlogSize.Collect(ch,
		float64(size),
		nil)
	qd.binlogFiles.Collect(ch,
		float64(count),
		nil)
	value, err := strconv.ParseFloat(strings.Split(filename, ".")[1], 64)
	if err != nil {
		logrus.Error(err)
		return
	}
	qd.binlogFileNumber.Collect(ch,
		value,
		nil)
	return
}

type binlogSize struct {
	*baseMetrics
}

func NewbinlogSize() *binlogSize {
	return &binlogSize{
		NewMetrics(
			"binlog_size_bytes",
			"Total number of executed MySQL commands.",
			nil)}
}
func (qd *binlogSize) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type binlogFiles struct {
	*baseMetrics
}

func NewbinlogFiles() *binlogFiles {
	return &binlogFiles{
		NewMetrics(
			"binlog_files",
			"Number of registered binlog files.",
			nil)}
}
func (qd *binlogFiles) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type binlogFileNumber struct {
	*baseMetrics
}

func NewbinlogFileNumber() *binlogFileNumber {
	return &binlogFileNumber{
		NewMetrics(
			"binlog_file_number",
			"The last binlog file number.",
			nil)}
}
func (qd *binlogFileNumber) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
