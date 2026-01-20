package metrics

import (
	"fmt"
	"github.com/alecthomas/kingpin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"mysqld_exporter/internal/exporter"
	"mysqld_exporter/internal/mysql"
	"reflect"
	"sort"
	"strings"
)

const (
	infoSchemaProcesslistQuery = `
		  SELECT
		    user,
		    SUBSTRING_INDEX(host, ':', 1) AS host,
		    COALESCE(command, '') AS command,
		    COALESCE(state, '') AS state,
		    COUNT(*) AS processes,
		    SUM(time) AS seconds
		  FROM information_schema.processlist
		  WHERE ID != connection_id()
		    AND TIME >= %d
		  GROUP BY user, host, command, state
	`
)

var (
	processlistMinTime = kingpin.Flag(
		"collect.info_schema.processlist.min_time",
		"Minimum time a thread must be in each state to be counted",
	).
		Default("0").
		Int()
	processesByUserFlag = kingpin.Flag(
		"collect.info_schema.processlist.processes_by_user",
		"Enable collecting the number of processes by user",
	).
		Default("true").
		Bool()
	processesByHostFlag = kingpin.Flag(
		"collect.info_schema.processlist.processes_by_host",
		"Enable collecting the number of processes by host",
	).
		Default("true").
		Bool()
)

type ScrapeProcesslist struct {
	instance mysql.Instance
	infoschemaprocesslistThreads
	infoschemaprocesslistSeconds
	infoschemaprocesslistProcessesByUser
	infoschemaprocesslistProcessesByHost
}

func init() {
	exporter.Register(
		NewScrapeProcesslist())
}
func NewScrapeProcesslist() *ScrapeProcesslist {
	return &ScrapeProcesslist{
		//instance:                             instance,
		infoschemaprocesslistThreads:         *NewInfoschemaprocesslistThreads(),
		infoschemaprocesslistSeconds:         *NewInfoschemaprocesslistSeconds(),
		infoschemaprocesslistProcessesByUser: *NewInfoschemaprocesslistProcessesByUser(),
		infoschemaprocesslistProcessesByHost: *NewInfoschemaprocesslistProcessesByHost(),
	}
}

func (qd ScrapeProcesslist) Collect(ch chan<- prometheus.Metric) {
	qd.instance = *GetInstance()

	processQuery := fmt.Sprintf(
		infoSchemaProcesslistQuery,
		*processlistMinTime,
	)
	if err := qd.instance.Ping(); err != nil {
		logrus.Errorf("ping mysql instance error: %s", err)
		return
	}
	db := instance.GetDB()
	rows, err := db.Query(processQuery)
	if err != nil {
		logrus.Error(err)
		return
	}
	defer rows.Close()
	var (
		user    string
		host    string
		command string
		state   string
		count   uint32
		time    uint32
	)
	stateCounts := make(map[string]map[string]uint32)
	stateTime := make(map[string]map[string]uint32)
	stateHostCounts := make(map[string]uint32)
	stateUserCounts := make(map[string]uint32)

	for rows.Next() {
		err = rows.Scan(
			&user,
			&host,
			&command,
			&state,
			&count,
			&time)
		if err != nil {
			logrus.Error(err)
			return
		}
		command = sanitizeState(command)
		state = sanitizeState(state)
		if host == "" {
			host = "unknown"
		}
		if _, ok := stateCounts[command]; !ok {
			stateCounts[command] = make(map[string]uint32)
			stateTime[command] = make(map[string]uint32)
		}
		if _, ok := stateCounts[command][state]; !ok {
			stateCounts[command][state] = 0
			stateTime[command][state] = 0
		}
		if _, ok := stateHostCounts[host]; !ok {
			stateHostCounts[host] = 0
		}
		if _, ok := stateUserCounts[user]; !ok {
			stateUserCounts[user] = 0
		}

		stateCounts[command][state] += count
		stateTime[command][state] += time
		stateHostCounts[host] += count
		stateUserCounts[user] += count
	}
	for _, command := range sortedMapKeys(stateCounts) {
		for _, state := range sortedMapKeys(stateCounts[command]) {

			qd.infoschemaprocesslistThreads.Collect(ch,
				float64(stateCounts[command][state]),
				[]string{
					command,
					state,
				})
			qd.infoschemaprocesslistSeconds.Collect(ch,
				float64(stateTime[command][state]),
				[]string{
					command,
					state,
				})
		}
	}
	if *processesByHostFlag {
		for _, host := range sortedMapKeys(stateHostCounts) {
			qd.infoschemaprocesslistProcessesByHost.Collect(ch,
				float64(stateHostCounts[host]),
				[]string{
					host,
				})
		}
	}
	if *processesByUserFlag {
		for _, user := range sortedMapKeys(stateUserCounts) {
			qd.infoschemaprocesslistProcessesByUser.Collect(ch,
				float64(stateUserCounts[user]),
				[]string{
					user,
				})
		}
	}
}

type infoschemaprocesslistThreads struct {
	*baseMetrics
}

func NewInfoschemaprocesslistThreads() *infoschemaprocesslistThreads {
	return &infoschemaprocesslistThreads{
		NewMetrics(
			"info_schema_processlist_threads",
			"The number of threads split by current state.",
			[]string{
				"command",
				"state"})}
}

func (qd *infoschemaprocesslistThreads) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoschemaprocesslistSeconds struct {
	*baseMetrics
}

func NewInfoschemaprocesslistSeconds() *infoschemaprocesslistSeconds {
	return &infoschemaprocesslistSeconds{
		NewMetrics(
			"info_schema_processlist_seconds",
			"The total number of seconds split by current state.",
			[]string{
				"command",
				"state"})}
}
func (qd *infoschemaprocesslistSeconds) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoschemaprocesslistProcessesByUser struct {
	*baseMetrics
}

func NewInfoschemaprocesslistProcessesByUser() *infoschemaprocesslistProcessesByUser {
	return &infoschemaprocesslistProcessesByUser{
		NewMetrics(
			"info_schema_processlist_processes_by_user",
			"The number of processes split by user.",
			[]string{
				"user"})}
}
func (qd *infoschemaprocesslistProcessesByUser) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type infoschemaprocesslistProcessesByHost struct {
	*baseMetrics
}

func NewInfoschemaprocesslistProcessesByHost() *infoschemaprocesslistProcessesByHost {
	return &infoschemaprocesslistProcessesByHost{
		NewMetrics(
			"info_schema_processlist_processes_by_host",
			"The number of processes split by host.",
			[]string{
				"host"})}
}
func (qd *infoschemaprocesslistProcessesByHost) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

func sanitizeState(state string) string {
	if state == "" {
		state = "unknown"
	}
	state = strings.ToLower(state)
	replacements := map[string]string{
		";": "",
		",": "",
		":": "",
		".": "",
		"(": "",
		")": "",
		" ": "_",
		"-": "_",
	}
	for r := range replacements {
		state = strings.Replace(state, r, replacements[r], -1)
	}
	return state
}
func sortedMapKeys(m interface{}) []string {
	v := reflect.ValueOf(m)
	keys := make([]string, 0, len(v.MapKeys()))
	for _, key := range v.MapKeys() {
		keys = append(keys, key.String())
	}
	sort.Strings(keys)
	return keys
}
