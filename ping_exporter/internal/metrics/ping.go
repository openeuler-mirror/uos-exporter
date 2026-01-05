package metrics

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"
	log "github.com/sirupsen/logrus"
)

var (
	targetFlag     = []string{"8.8.8.8", "1.1.1.1", "github.com"}
	pingInterval   = "5s"
	pingTimeout    = "4s"
	pingSize       = uint16(56)
	historySize    = 10
	dnsRefresh     = "1m"
	dnsNameServer  = ""
	disableIPv6    = true
	disableIPv4    = false
	desiredTargets = &targets{}
)

func startMonitor(cfg *Config, resolver *net.Resolver) (*mon.Monitor, error) {
	var bind4, bind6 string
	if ln, err := net.Listen("tcp4", "127.0.0.1:0"); err == nil {
		// ipv4 enabled
		_ = ln.Close()
		bind4 = "0.0.0.0"
	}
	if ln, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		// ipv6 enabled
		_ = ln.Close()
		bind6 = "::"
	}
	pinger, err := ping.New(bind4, bind6)
	if err != nil {
		return nil, fmt.Errorf("cannot start monitoring: %w", err)
	}

	if pinger.PayloadSize() != cfg.Ping.Size {
		pinger.SetPayloadSize(cfg.Ping.Size)
	}

	monitor := mon.New(pinger,
		cfg.Ping.Interval.Duration(),
		cfg.Ping.Timeout.Duration())
	monitor.HistorySize = cfg.Ping.History

	err = upsertTargets(desiredTargets, resolver, cfg, monitor)
	if err != nil {
		log.Fatalln(err)
	}

	go startDNSAutoRefresh(cfg.DNS.Refresh.Duration(), desiredTargets, monitor, cfg)
	return monitor, nil
}

func upsertTargets(globalTargets *targets, resolver *net.Resolver, cfg *Config, monitor *mon.Monitor) error {
	oldTargets := globalTargets.Targets()
	newTargets := make([]*target, len(cfg.Targets))
	var wg sync.WaitGroup
	for i, t := range cfg.Targets {
		newTarget := globalTargets.Get(t.Addr)
		if newTarget == nil {
			newTarget = &target{
				host:      t.Addr,
				addresses: make([]net.IPAddr, 0),
				delay:     time.Duration(10*i) * time.Millisecond,
				resolver:  resolver,
			}
		}

		newTargets[i] = newTarget

		wg.Add(1)
		go func() {
			err := newTarget.addOrUpdateMonitor(monitor, targetOpts{
				disableIPv4: cfg.Options.DisableIPv4,
				disableIPv6: cfg.Options.DisableIPv6,
			})
			if err != nil {
				log.Errorf("failed to setup target: %v", err)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	globalTargets.SetTargets(newTargets)

	removed := removedTargets(oldTargets, globalTargets)
	for _, removedTarget := range removed {
		log.Infof("remove target: %s", removedTarget.host)
		removedTarget.removeFromMonitor(monitor)
	}
	return nil
}

func removedTargets(old []*target, new *targets) []*target {
	var ret []*target
	for _, oldTarget := range old {
		if !new.Contains(oldTarget) {
			ret = append(ret, oldTarget)
		}
	}
	return ret
}

func startDNSAutoRefresh(interval time.Duration, tar *targets, monitor *mon.Monitor, cfg *Config) {
	if interval <= 0 {
		return
	}

	for range time.NewTicker(interval).C {
		refreshDNS(tar, monitor, cfg)
	}
}

func refreshDNS(tar *targets, monitor *mon.Monitor, cfg *Config) {
	log.Infoln("refreshing DNS")
	for _, t := range tar.Targets() {
		go func(ta *target) {
			err := ta.addOrUpdateMonitor(monitor, targetOpts{
				disableIPv4: cfg.Options.DisableIPv4,
				disableIPv6: cfg.Options.DisableIPv6,
			})
			if err != nil {
				log.Errorf("could not refresh dns: %v", err)
			}
		}(t)
	}
}

// addFlagToConfig updates cfg with command line flag values, unless the
// config has non-zero values.
func addFlagToConfig(cfg *Config) {
	if len(cfg.Targets) == 0 {
		cfg.Targets = make([]TargetConfig, len(targetFlag))
		for i, t := range targetFlag {
			cfg.Targets[i] = TargetConfig{
				Addr: t,
			}
		}
	}

	pingInterval, err := time.ParseDuration(pingInterval)
	if err != nil {
		log.Fatal("failed to get duration time")
	}

	pingTimeout, err := time.ParseDuration(pingTimeout)
	if err != nil {
		log.Fatal("failed to get duration time")
	}

	dnsRefresh, err := time.ParseDuration(dnsRefresh)
	if err != nil {
		log.Fatal("failed to get duration time")
	}

	if cfg.Ping.History == 0 {
		cfg.Ping.History = historySize
	}
	if cfg.Ping.Interval == 0 {
		cfg.Ping.Interval.Set(pingInterval)
	}
	if cfg.Ping.Timeout == 0 {
		cfg.Ping.Timeout.Set(pingTimeout)
	}
	if cfg.Ping.Size == 0 {
		cfg.Ping.Size = pingSize
	}
	if cfg.DNS.Refresh == 0 {
		cfg.DNS.Refresh.Set(dnsRefresh)
	}
	if cfg.DNS.Nameserver == "" {
		cfg.DNS.Nameserver = dnsNameServer
	}
	if !cfg.Options.DisableIPv6 {
		cfg.Options.DisableIPv6 = disableIPv6
	}
	if !cfg.Options.DisableIPv4 {
		cfg.Options.DisableIPv4 = disableIPv4
	}
}

func setupResolver(cfg *Config) *net.Resolver {
	if cfg.DNS.Nameserver == "" {
		return net.DefaultResolver
	}

	if !strings.HasSuffix(cfg.DNS.Nameserver, ":53") {
		cfg.DNS.Nameserver += ":53"
	}
	dialer := func(ctx context.Context, _, _ string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "udp", cfg.DNS.Nameserver)
	}

	return &net.Resolver{PreferGo: true, Dial: dialer}
}

func setLogLevel(l string) {
	switch l {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
}
