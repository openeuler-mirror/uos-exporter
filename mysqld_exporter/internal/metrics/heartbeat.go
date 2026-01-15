package metrics

import (
	"database/sql"
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"strconv"
)

const (
	heartbeat      = "heartbeat"
	heartbeatQuery = "SELECT UNIX_TIMESTAMP(ts), UNIX_TIMESTAMP(%s), server_id from `%s`.`%s`"
)

var (
	collectHeartbeatDatabase = kingpin.Flag(
		"collect.heartbeat.database",
		"Database from where to collect heartbeat data",
	).Default("heartbeat").String()
	collectHeartbeatTable = kingpin.Flag(
		"collect.heartbeat.table",
		"Table from where to collect heartbeat data",
	).Default("heartbeat").String()
	collectHeartbeatUtc = kingpin.Flag(
		"collect.heartbeat.utc",
		"Use UTC for timestamps of the current server (`pt-heartbeat` is called with `--utc`)",
	).Bool()
)

type ScrapeHeartbeat struct {
	instance mysql.Instance
	HeartbeatStored
	HeartbeatNow
}

func init() {
	exporter.Register(
		NewScrapeHeartbeat())
}
func NewScrapeHeartbeat() *ScrapeHeartbeat {
	return &ScrapeHeartbeat{
		//instance:        instance,
		HeartbeatStored: *NewHeartbeatStored(),
		HeartbeatNow:    *NewHeartbeatNow(),
	}
}
func (qd ScrapeHeartbeat) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	query := fmt.Sprintf(heartbeatQuery, nowExpr(), *collectHeartbeatDatabase, *collectHeartbeatTable)
	rows, err := db.Query(query)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		now      sql.RawBytes
		ts       sql.RawBytes
		serverId int
	)
	for rows.Next() {
		err := rows.Scan(
			&ts,
			&now,
			&serverId)
		if err != nil {
			logrus.Error(err)
			return
		}
		tsFloatVal, err := strconv.ParseFloat(string(ts),
			64)
		if err != nil {
			logrus.Error(err)
			return
		}
		nowFloatVal, err := strconv.ParseFloat(string(now),
			64)
		if err != nil {
			logrus.Error(err)
			return
		}
		serverId := strconv.Itoa(serverId)
		qd.HeartbeatStored.Collect(ch,
			nowFloatVal,
			[]string{
				serverId,
			})
		qd.HeartbeatNow.Collect(ch,
			tsFloatVal,
			[]string{
				serverId,
			})
	}

}

type HeartbeatStored struct {
	*baseMetrics
}

func NewHeartbeatStored() *HeartbeatStored {
	return &HeartbeatStored{
		NewMetrics(
			"heartbeat_info_stored_timestamp_seconds",
			"Timestamp stored in the heartbeat table.",
			[]string{
				"server_id",
			})}
}
func (qd *HeartbeatStored) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type HeartbeatNow struct {
	*baseMetrics
}

func NewHeartbeatNow() *HeartbeatNow {
	return &HeartbeatNow{
		NewMetrics(
			"heartbeat_now_timestamp_seconds",
			"Timestamp of the current server.",
			[]string{
				"server_id",
			})}
}
func (qd *HeartbeatNow) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

func nowExpr() string {
	if *collectHeartbeatUtc {
		return "UTC_TIMESTAMP(6)"
	}
	return "NOW(6)"
}
