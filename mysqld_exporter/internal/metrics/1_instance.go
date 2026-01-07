package metrics

import (
	"database/sql"
	"fmt"
	"mysqld_exporter/config"
	"mysqld_exporter/internal/mysql"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	instance *mysql.Instance
	logRE    = regexp.MustCompile(`.+\.(\d+)$`)
)

func ConnectMysql(setting config.MysqlSettings) {
	var err error
	dsn := fmt.Sprintf("%s:%s@%s(%s:%d)/", setting.User, setting.Password, setting.Protocol, setting.Host, setting.Port)
	instance, err = mysql.NewInstance(dsn)
	if err != nil {
		fmt.Printf("mysql init error")
		panic(err)
	}
	err = instance.Ping()
	if err != nil {
		fmt.Printf("mysql ping error")
		panic(err)
	}
}

func GetInstance() *mysql.Instance {
	return instance
}

func parseStatus(data sql.RawBytes) (float64, bool) {
	dataString := strings.ToLower(string(data))
	switch dataString {
	case "yes", "on":
		return 1, true
	case "no", "off", "disabled":
		return 0, true
	// SHOW SLAVE STATUS Slave_IO_Running can return "Connecting" which is a non-running state.
	case "connecting":
		return 0, true
	// SHOW GLOBAL STATUS like 'wsrep_cluster_status' can return "Primary" or "non-Primary"/"Disconnected"
	case "primary":
		return 1, true
	case "non-primary", "disconnected":
		return 0, true
	}
	if ts, err := time.Parse("Jan _2 15:04:05 2006 MST", string(data)); err == nil {
		return float64(ts.Unix()), true
	}
	if ts, err := time.Parse(time.DateTime, string(data)); err == nil {
		return float64(ts.Unix()), true
	}
	if logNum := logRE.Find(data); logNum != nil {
		value, err := strconv.ParseFloat(string(logNum), 64)
		return value, err == nil
	}
	value, err := strconv.ParseFloat(string(data), 64)
	return value, err == nil
}
