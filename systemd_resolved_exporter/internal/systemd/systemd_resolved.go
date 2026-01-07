package systemd

import (
	"bufio"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"

	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	namespace       = "systemd_resolved"
	resolvedCommand = "systemd-resolve"
	resolvedArgs    = "--statistics"
)

var (
	statusLineRegex = regexp.MustCompile(`[a-zA-Z ]+: ?[0-9]+`)
	cachePath       = "org.freedesktop.resolve1.Manager.CacheStatistics"
	transactionPath = "org.freedesktop.resolve1.Manager.TransactionStatistics"
	dnssecPath      = "org.freedesktop.resolve1.Manager.DNSSECStatistics"
)

func GetSystemdResolvedStats() map[string]float64 {
	stats := make(map[string]float64)
	cmd := exec.Command(resolvedCommand, resolvedArgs)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
		return stats
	}
	if err := cmd.Start(); err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
		return stats
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if statusLineRegex.MatchString(line) {
			parts := strings.Split(line, ":")
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value64, err := strconv.ParseFloat(value,
				64)
			if err != nil {
				logrus.Error("Failed to parse "+
					"systemd-resolved stats: ",
					err)
				continue
			}
			stats[key] = value64
		}
	}
	err = cmd.Wait()
	if err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
	}
	return stats
}

func GetSystemdResolvedStatsWithDbus() map[string]float64 {
	stats := make(map[string]float64)
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		logrus.Error("Failed to connect to system bus: ", err)
		return stats
	}
	defer conn.Close()
	obj := conn.Object("org.freedesktop.resolve1",
		"/org/freedesktop/resolve1")
	variant, err := obj.GetProperty(cachePath)
	if err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
		return stats
	}
	var (
		cacheStats []float64
	)
	for _, v := range variant.Value().([]interface{}) {
		i := v.(uint64)
		cacheStats = append(cacheStats,
			float64(i))
	}
	stats["Current Cache Size"] = cacheStats[0]
	stats["Cache Hits"] = cacheStats[1]
	stats["Cache Misses"] = cacheStats[2]
	variant, err = obj.GetProperty(transactionPath)
	if err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
		return stats
	}
	var (
		transactionStats []float64
	)
	for _, v := range variant.Value().([]interface{}) {
		i := v.(uint64)
		transactionStats = append(transactionStats,
			float64(i))
	}
	stats["Current Transactions"] = transactionStats[0]
	stats["Total Transactions"] = transactionStats[1]
	variant, err = obj.GetProperty(dnssecPath)
	if err != nil {
		logrus.Error("Failed to get systemd-resolved stats: ", err)
		return stats
	}
	var (
		dnssecStats []float64
	)
	for _, v := range variant.Value().([]interface{}) {
		i := v.(uint64)
		dnssecStats = append(dnssecStats,
			float64(i))
	}
	stats["Secure"] = transactionStats[0]
	stats["Insecure"] = transactionStats[1]
	return stats

}
