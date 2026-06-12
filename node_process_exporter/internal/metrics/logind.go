package metrics

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"node_process_exporter/internal/exporter"

	"github.com/godbus/dbus/v5"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	logindSubsystem = "logind"
	dbusObject      = "org.freedesktop.login1"
	dbusPath        = "/org/freedesktop/login1"
)

var (
	// Taken from logind as of systemd v229.
	// "other" is the fallback value for unknown values (in case logind gets extended in the future).
	attrRemoteValues = []string{"true", "false"}
	attrTypeValues   = []string{"other", "unspecified", "tty", "x11", "wayland", "mir", "web"}
	attrClassValues  = []string{"other", "user", "greeter", "lock-screen", "background"}
)

func init() {
	exporter.Register(NewLogindCollector())
}

type logindCollector struct {
	*baseMetrics
	sessionsDesc *prometheus.Desc
	logger       *slog.Logger
}

type logindDbus struct {
	conn   *dbus.Conn
	object dbus.BusObject
}

type logindInterface interface {
	listSeats() ([]string, error)
	listSessions() ([]logindSessionEntry, error)
	getSession(logindSessionEntry) *logindSession
}

type logindSession struct {
	seat        string
	remote      string
	sessionType string
	class       string
}

// Struct elements must be public for the reflection magic of godbus to work.
type logindSessionEntry struct {
	SessionID         string
	UserID            uint32
	UserName          string
	SeatID            string
	SessionObjectPath dbus.ObjectPath
}

type logindSeatEntry struct {
	SeatID         string
	SeatObjectPath dbus.ObjectPath
}

func NewLogindCollector() *logindCollector {
	sessionsDesc := prometheus.NewDesc(
		"node_logind_sessions",
		"Number of sessions registered in logind.", []string{"seat", "remote", "type", "class"}, nil,
	)

	return &logindCollector{
		baseMetrics:  NewMetrics("node_logind_collect_errors_total", "Number of errors that occurred during logind collection", []string{}),
		sessionsDesc: sessionsDesc,
		logger:       slog.Default(),
	}
}

func (c *logindCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating logind metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *logindCollector) Update(ch chan<- prometheus.Metric) error {
	dbusConn, err := c.newDbus()
	if err != nil {
		return fmt.Errorf("unable to connect to dbus: %w", err)
	}
	defer dbusConn.conn.Close()

	return c.collectMetrics(ch, dbusConn)
}

func (c *logindCollector) collectMetrics(ch chan<- prometheus.Metric, conn logindInterface) error {
	seats, err := conn.listSeats()
	if err != nil {
		return fmt.Errorf("unable to get seats: %w", err)
	}

	sessionList, err := conn.listSessions()
	if err != nil {
		return fmt.Errorf("unable to get sessions: %w", err)
	}

	sessions := make(map[logindSession]float64)

	for _, s := range sessionList {
		session := conn.getSession(s)
		if session != nil {
			sessions[*session]++
		}
	}

	for _, remote := range attrRemoteValues {
		for _, sessionType := range attrTypeValues {
			for _, class := range attrClassValues {
				for _, seat := range seats {
					count := sessions[logindSession{seat, remote, sessionType, class}]

					ch <- prometheus.MustNewConstMetric(
						c.sessionsDesc, prometheus.GaugeValue, count,
						seat, remote, sessionType, class)
				}
			}
		}
	}

	return nil
}

func (c *logindCollector) knownStringOrOther(value string, known []string) string {
	for i := range known {
		if value == known[i] {
			return value
		}
	}

	return "other"
}

func (c *logindCollector) newDbus() (*logindDbus, error) {
	conn, err := dbus.SystemBusPrivate()
	if err != nil {
		return nil, err
	}

	methods := []dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Getuid()))}

	err = conn.Auth(methods)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	err = conn.Hello()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	object := conn.Object(dbusObject, dbus.ObjectPath(dbusPath))

	return &logindDbus{
		conn:   conn,
		object: object,
	}, nil
}

func (c *logindDbus) listSeats() ([]string, error) {
	var result [][]interface{}
	err := c.object.Call(dbusObject+".Manager.ListSeats", 0).Store(&result)
	if err != nil {
		return nil, err
	}

	resultInterface := make([]interface{}, len(result))
	for i := range result {
		resultInterface[i] = result[i]
	}

	seats := make([]logindSeatEntry, len(result))
	seatsInterface := make([]interface{}, len(seats))
	for i := range seats {
		seatsInterface[i] = &seats[i]
	}

	err = dbus.Store(resultInterface, seatsInterface...)
	if err != nil {
		return nil, err
	}

	ret := make([]string, len(seats)+1)
	for i := range seats {
		ret[i] = seats[i].SeatID
	}
	// Always add the empty seat, which is used for remote sessions like SSH
	ret[len(seats)] = ""

	return ret, nil
}

func (c *logindDbus) listSessions() ([]logindSessionEntry, error) {
	var result [][]interface{}
	err := c.object.Call(dbusObject+".Manager.ListSessions", 0).Store(&result)
	if err != nil {
		return nil, err
	}

	resultInterface := make([]interface{}, len(result))
	for i := range result {
		resultInterface[i] = result[i]
	}

	sessions := make([]logindSessionEntry, len(result))
	sessionsInterface := make([]interface{}, len(sessions))
	for i := range sessions {
		sessionsInterface[i] = &sessions[i]
	}

	err = dbus.Store(resultInterface, sessionsInterface...)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func (c *logindDbus) getSession(session logindSessionEntry) *logindSession {
	object := c.conn.Object(dbusObject, session.SessionObjectPath)

	remote, err := object.GetProperty(dbusObject + ".Session.Remote")
	if err != nil {
		return nil
	}

	sessionType, err := object.GetProperty(dbusObject + ".Session.Type")
	if err != nil {
		return nil
	}

	sessionTypeStr, ok := sessionType.Value().(string)
	if !ok {
		return nil
	}

	class, err := object.GetProperty(dbusObject + ".Session.Class")
	if err != nil {
		return nil
	}

	classStr, ok := class.Value().(string)
	if !ok {
		return nil
	}

	// 创建一个临时的logindCollector实例来调用knownStringOrOther方法
	tempCollector := &logindCollector{}
	return &logindSession{
		seat:        session.SeatID,
		remote:      remote.String(),
		sessionType: tempCollector.knownStringOrOther(sessionTypeStr, attrTypeValues),
		class:       tempCollector.knownStringOrOther(classStr, attrClassValues),
	}
}
