package metrics

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/itchyny/timefmt-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	// 64-bit float mantissa: https://en.wikipedia.org/wiki/Double-precision_floating-point_format
	float64Mantissa uint64 = 9007199254740992
	wrapCounters           = true
	srcAddress             = ""
)

// Types preceded by an enum with their actual type.
var combinedTypeMapping = map[string]map[int]string{
	"InetAddress": {
		1: "InetAddressIPv4",
		2: "InetAddressIPv6",
	},
	"InetAddressMissingSize": {
		1: "InetAddressIPv4",
		2: "InetAddressIPv6",
	},
	"LldpPortId": {
		1: "DisplayString",
		2: "DisplayString",
		3: "PhysAddress48",
		5: "DisplayString",
		7: "DisplayString",
	},
}

func oidToList(oid string) []int {
	result := []int{}
	for _, x := range strings.Split(oid, ".") {
		o, _ := strconv.Atoi(x)
		result = append(result, o)
	}
	return result
}

func listToOid(l []int) string {
	var result []string
	for _, o := range l {
		result = append(result, strconv.Itoa(o))
	}
	return strings.Join(result, ".")
}

type ScrapeResults struct {
	pdus    []gosnmp.SnmpPDU
	packets uint64
	retries uint64
}

func ScrapeTarget(snmp SNMPScraper, target string, auth *Auth, module *Module, logger *logrus.Entry, metrics Metrics) (ScrapeResults, error) {
	results := ScrapeResults{}

	// Process filters and update config
	newGet, newWalk := processFilters(snmp, module, logger, metrics)

	// Process get requests
	if err := processGetRequests(snmp, auth.Version, module.WalkParams.MaxRepetitions, newGet, &results, logger); err != nil {
		return results, err
	}

	// Process walk requests
	if err := processWalkRequests(snmp, newWalk, &results); err != nil {
		return results, err
	}

	return results, nil
}

func processFilters(snmp SNMPScraper, module *Module, logger *logrus.Entry, metrics Metrics) ([]string, []string) {
	newGet := module.Get
	newWalk := module.Walk

	for _, filter := range module.Filters {
		pdus, err := snmp.WalkAll(filter.Oid)
		if err != nil {
			logger.Info("Error getting OID, won't do any filter on this oid", "oid", filter.Oid)
			continue
		}

		allowedList := filterAllowedIndices(logger, filter, pdus, []string{}, metrics)
		newWalk = updateWalkConfig(newWalk, filter, logger)
		newGet = updateGetConfig(newGet, filter, logger)
		newGet = addAllowedIndices(filter, allowedList, logger, newGet)
	}

	return newGet, newWalk
}

func processGetRequests(snmp SNMPScraper, version int, maxRepetitions uint32, getOids []string, results *ScrapeResults, logger *logrus.Entry) error {
	maxOids := int(maxRepetitions)
	if maxOids == 0 || version == 1 {
		maxOids = 1
	}

	for len(getOids) > 0 {
		oids := len(getOids)
		if oids > maxOids {
			oids = maxOids
		}

		packet, err := snmp.Get(getOids[:oids])
		if err != nil {
			return err
		}

		if err := processPacketResponse(packet, version, getOids[:oids], results, logger); err != nil {
			return err
		}

		getOids = getOids[oids:]
	}
	return nil
}

func processPacketResponse(packet *gosnmp.SnmpPacket, version int, oids []string, results *ScrapeResults, logger *logrus.Entry) error {
	if packet.Error == gosnmp.NoSuchName && version == 1 {
		logger.Debug("OID not supported by target", "oids", oids[0])
		return nil
	}

	if packet.Error != gosnmp.NoError {
		return fmt.Errorf("error reported by target: Error Status %d", packet.Error)
	}

	for _, v := range packet.Variables {
		if v.Type == gosnmp.NoSuchObject || v.Type == gosnmp.NoSuchInstance {
			logger.Debug("OID not supported by target", "oids", v.Name)
			continue
		}
		results.pdus = append(results.pdus, v)
	}
	return nil
}

func processWalkRequests(snmp SNMPScraper, walkOids []string, results *ScrapeResults) error {
	for _, subtree := range walkOids {
		pdus, err := snmp.WalkAll(subtree)
		if err != nil {
			return err
		}
		results.pdus = append(results.pdus, pdus...)
	}
	return nil
}

func configureTarget(g *gosnmp.GoSNMP, target string) error {
	// 处理传输协议
	if s := strings.SplitN(target, "://", 2); len(s) == 2 {
		g.Transport = s[0]
		target = s[1]
	}

	// 设置默认端口
	g.Target = target
	g.Port = 161

	// 解析主机和端口
	// host, port, err := parseHostPort(target, g)
	err := parseHostPort(target, g)
	if err != nil {
		return err
	}
	// g.Target = host
	// g.Port = port

	return nil
}

func parseHostPort(target string, g *gosnmp.GoSNMP) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		// return "", 0, err
		return nil
	}
	g.Target = host
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("error converting port number to int for target %q: %w", target, err)
	}
	// 安全检查：确保端口在有效范围内
	if port < 0 || port > 65535 {
		return fmt.Errorf("port number %d out of range [0-65535] for target %q", port, target)
	}
	g.Port = uint16(port)
	return nil
}

func filterAllowedIndices(logger *logrus.Entry, filter DynamicFilter, pdus []gosnmp.SnmpPDU, allowedList []string, metrics Metrics) []string {
	logger.Debug("Evaluating rule for oid", "oid", filter.Oid)
	for _, pdu := range pdus {
		if isPduAllowed(logger, filter, &pdu, metrics) {
			index := extractIndexFromPdu(&pdu)
			logger.Debug("Caching index", "index", index)
			allowedList = append(allowedList, index)
		}
	}
	return allowedList
}

// isPduAllowed 检查 PDU 是否满足过滤条件
func isPduAllowed(logger *logrus.Entry, filter DynamicFilter, pdu *gosnmp.SnmpPDU, metrics Metrics) bool {
	for _, val := range filter.Values {
		snmpval := pduValueAsString(pdu, "DisplayString", metrics)
		logger.Debug("evaluating filters", "config value", val, "snmp value", snmpval)

		if regexp.MustCompile(val).MatchString(snmpval) {
			return true
		}
	}
	return false
}

// extractIndexFromPdu 从 PDU 中提取索引
func extractIndexFromPdu(pdu *gosnmp.SnmpPDU) string {
	pduArray := strings.Split(pdu.Name, ".")
	return pduArray[len(pduArray)-1]
}

func updateWalkConfig(walkConfig []string, filter DynamicFilter, logger *logrus.Entry) []string {
	newCfg := []string{}
	for _, elem := range walkConfig {
		if !isOidInTargets(elem, filter.Targets, logger) {
			newCfg = append(newCfg, elem)
		}
	}
	return newCfg
}

// isOidInTargets 检查 OID 是否在目标列表中
func isOidInTargets(oid string, targets []string, logger *logrus.Entry) bool {
	for _, targetOid := range targets {
		if oid == targetOid {
			logger.Debug("Deleting for walk configuration", "oid", targetOid)
			return true
		}
	}
	return false
}

func updateGetConfig(getConfig []string, filter DynamicFilter, logger *logrus.Entry) []string {
	newCfg := []string{}
	for _, elem := range getConfig {
		found := false
		for _, targetOid := range filter.Targets {
			if strings.HasPrefix(elem, targetOid) {
				found = true
				break
			}
		}
		// Oid not found in targets, we keep it.
		if !found {
			logger.Debug("Keeping get configuration", "oid", elem)
			newCfg = append(newCfg, elem)
		}
	}
	return newCfg
}

func addAllowedIndices(filter DynamicFilter, allowedList []string, logger *logrus.Entry, newCfg []string) []string {
	for _, targetOid := range filter.Targets {
		for _, index := range allowedList {
			logger.Debug("Adding get configuration", "oid", targetOid+"."+index)
			newCfg = append(newCfg, targetOid+"."+index)
		}
	}
	return newCfg
}

type MetricNode struct {
	metric *Metric

	children map[int]*MetricNode
}

// Build a tree of metrics from the config, for fast lookup when there's lots of them.
func buildMetricTree(metrics []*Metric) *MetricNode {
	metricTree := &MetricNode{children: map[int]*MetricNode{}}
	for _, metric := range metrics {
		head := metricTree
		for _, o := range oidToList(metric.Oid) {
			_, ok := head.children[o]
			if !ok {
				head.children[o] = &MetricNode{children: map[int]*MetricNode{}}
			}
			head = head.children[o]
		}
		head.metric = metric
	}
	return metricTree
}

type Metrics struct {
	SNMPCollectionDuration *prometheus.HistogramVec
	SNMPUnexpectedPduType  prometheus.Counter
	SNMPDuration           prometheus.Histogram
	SNMPPackets            prometheus.Counter
	SNMPRetries            prometheus.Counter
	SNMPInflight           prometheus.Gauge
}

type NamedModule struct {
	*Module
	name string
}

func NewNamedModule(name string, module *Module) *NamedModule {
	return &NamedModule{
		Module: module,
		name:   name,
	}
}

type Collector struct {
	ctx         context.Context
	target      string
	auth        *Auth
	authName    string
	modules     []*NamedModule
	logger      *logrus.Entry
	metrics     Metrics
	concurrency int
	snmpContext string
	debugSNMP   bool
}

func SnmpCollectorNew(ctx context.Context, target, authName, snmpContext string, auth *Auth, modules []*NamedModule, logger *logrus.Entry, metrics Metrics, conc int, debugSNMP bool) *Collector {
	return &Collector{
		ctx:         ctx,
		target:      target,
		authName:    authName,
		auth:        auth,
		modules:     modules,
		snmpContext: snmpContext,
		logger:      logger.WithField("source_address", srcAddress),
		metrics:     metrics,
		concurrency: conc,
		debugSNMP:   debugSNMP,
	}
}

// Describe implements Prometheus.Collector.
func (c Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c Collector) collect(ch chan<- prometheus.Metric, logger *logrus.Entry, client SNMPScraper, module *NamedModule) {
	var (
		packets uint64
		retries uint64
	)

	// 设置客户端选项
	c.configureClientOptions(client, module, packets, retries)

	// 执行SNMP采集
	results, err := c.scrapeTarget(client, module, logger)
	if err != nil {
		c.handleScrapeError(ch, module, logger, err)
		return
	}

	results.packets = packets
	results.retries = retries

	// 处理采集结果并生成指标
	c.processScrapeResults(ch, module, results, logger)
}

// configureClientOptions 配置SNMP客户端选项
func (c Collector) configureClientOptions(client SNMPScraper, module *NamedModule, packets uint64, retries uint64) {
	// var (
	// 	packets uint64
	// 	retries uint64
	// )

	client.SetOptions(
		// 设置指标选项
		func(g *gosnmp.GoSNMP) {
			var sent time.Time
			g.OnSent = func(x *gosnmp.GoSNMP) {
				sent = time.Now()
				c.metrics.SNMPPackets.Inc()
				packets++
			}
			g.OnRecv = func(x *gosnmp.GoSNMP) {
				c.metrics.SNMPDuration.Observe(time.Since(sent).Seconds())
			}
			g.OnRetry = func(x *gosnmp.GoSNMP) {
				c.metrics.SNMPRetries.Inc()
				retries++
			}
		},
		// 设置Walk选项
		func(g *gosnmp.GoSNMP) {
			g.Retries = *module.WalkParams.Retries
			g.Timeout = module.WalkParams.Timeout
			g.MaxRepetitions = module.WalkParams.MaxRepetitions
			g.UseUnconnectedUDPSocket = module.WalkParams.UseUnconnectedUDPSocket
			if module.WalkParams.AllowNonIncreasingOIDs {
				g.AppOpts = map[string]interface{}{"c": true}
			}
		},
	)
}

// scrapeTarget 执行SNMP采集
func (c Collector) scrapeTarget(client SNMPScraper, module *NamedModule, logger *logrus.Entry) (ScrapeResults, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		logger.Debug("Finished scrape", "duration_seconds", duration)
		c.metrics.SNMPCollectionDuration.WithLabelValues(module.name).Observe(duration)
	}()

	c.metrics.SNMPInflight.Inc()
	defer c.metrics.SNMPInflight.Dec()

	return ScrapeTarget(client, c.target, c.auth, module.Module, logger, c.metrics)
}

// handleScrapeError 处理采集错误
func (c Collector) handleScrapeError(ch chan<- prometheus.Metric, module *NamedModule, logger *logrus.Entry, err error) {
	logger.Info("Error scraping target", "err", err)
	moduleLabel := prometheus.Labels{"module": module.name}
	ch <- prometheus.NewInvalidMetric(
		prometheus.NewDesc("snmp_error", "Error scraping target", nil, moduleLabel),
		err,
	)
}

// processScrapeResults 处理采集结果并生成指标
func (c Collector) processScrapeResults(ch chan<- prometheus.Metric, module *NamedModule, results ScrapeResults, logger *logrus.Entry) {
	start := time.Now()

	// 发送采集过程指标
	c.sendScrapeMetrics(ch, module, results, start)

	// 处理PDU数据并生成业务指标
	c.processPduData(ch, module, results, logger)

	// 发送总耗时指标
	ch <- prometheus.MustNewConstMetric(
		prometheus.NewDesc("snmp_scrape_duration_seconds", "Total SNMP time scrape took (walk and processing).", nil, prometheus.Labels{"module": module.name}),
		prometheus.GaugeValue,
		time.Since(start).Seconds(),
	)
}

// sendScrapeMetrics 发送采集过程相关指标
func (c Collector) sendScrapeMetrics(ch chan<- prometheus.Metric, module *NamedModule, results ScrapeResults, start time.Time) {
	moduleLabel := prometheus.Labels{"module": module.name}

	metrics := []struct {
		desc  *prometheus.Desc
		value float64
	}{
		{
			prometheus.NewDesc("snmp_scrape_walk_duration_seconds", "Time SNMP walk/bulkwalk took.", nil, moduleLabel),
			time.Since(start).Seconds(),
		},
		{
			prometheus.NewDesc("snmp_scrape_packets_sent", "Packets sent for get, bulkget, and walk; including retries.", nil, moduleLabel),
			float64(results.packets),
		},
		{
			prometheus.NewDesc("snmp_scrape_packets_retried", "Packets retried for get, bulkget, and walk.", nil, moduleLabel),
			float64(results.retries),
		},
		{
			prometheus.NewDesc("snmp_scrape_pdus_returned", "PDUs returned from get, bulkget, and walk.", nil, moduleLabel),
			float64(len(results.pdus)),
		},
	}

	for _, m := range metrics {
		ch <- prometheus.MustNewConstMetric(m.desc, prometheus.GaugeValue, m.value)
	}
}

// processPduData 处理PDU数据并生成业务指标
func (c Collector) processPduData(ch chan<- prometheus.Metric, module *NamedModule, results ScrapeResults, logger *logrus.Entry) {
	// 构建OID到PDU的映射
	oidToPdu := make(map[string]gosnmp.SnmpPDU, len(results.pdus))
	for _, pdu := range results.pdus {
		oidToPdu[pdu.Name[1:]] = pdu
	}

	// 构建指标树
	metricTree := buildMetricTree(module.Metrics)

	// 处理每个PDU
	for oid, pdu := range oidToPdu {
		c.processSinglePdu(ch, metricTree, oid, pdu, oidToPdu, logger)
	}
}

// processSinglePdu 处理单个PDU数据
func (c Collector) processSinglePdu(ch chan<- prometheus.Metric, metricTree *MetricNode, oid string, pdu gosnmp.SnmpPDU, oidToPdu map[string]gosnmp.SnmpPDU, logger *logrus.Entry) {
	head := metricTree
	oidList := oidToList(oid)

	for i, o := range oidList {
		var ok bool
		head, ok = head.children[o]
		if !ok {
			break
		}

		if head.metric != nil {
			// 找到匹配的指标
			samples := pduToSamples(oidList[i+1:], &pdu, head.metric, oidToPdu, logger, c.metrics)
			for _, sample := range samples {
				ch <- sample
			}
			break
		}
	}
}

// Collect implements Prometheus.Collector.
func (c Collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()

	wg := sync.WaitGroup{}
	workerChan := make(chan *NamedModule)

	// 创建工作协程
	c.startWorkers(ch, workerChan, &wg)

	// 分发任务
	c.dispatchModules(ctx, workerChan)

	// 等待所有工作完成
	close(workerChan)
	wg.Wait()
}

// startWorkers 创建并启动工作协程
func (c Collector) startWorkers(ch chan<- prometheus.Metric, workerChan chan *NamedModule, wg *sync.WaitGroup) {
	workerCount := c.getWorkerCount()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			c.workerLoop(ch, workerChan, workerID)
		}(i)
	}
}

// getWorkerCount 获取有效的worker数量
func (c Collector) getWorkerCount() int {
	if c.concurrency < 1 {
		return 1
	}
	return c.concurrency
}

// workerLoop 工作协程的主循环
func (c Collector) workerLoop(ch chan<- prometheus.Metric, workerChan chan *NamedModule, workerID int) {
	logger := c.logger.WithField("worker", workerID)
	client, err := c.createSNMPClient(logger)
	if err != nil {
		c.handleWorkerError(ch, logger, err)
		return
	}
	defer client.Close()

	for module := range workerChan {
		c.processModule(ch, logger, client, module)
	}
}

// createSNMPClient 创建SNMP客户端
func (c Collector) createSNMPClient(logger *logrus.Entry) (SNMPScraper, error) {
	client, err := NewGoSNMP(logger, c.target, srcAddress, c.debugSNMP)
	if err != nil {
		return nil, err
	}

	useUnconnected := c.checkUnconnectedSocketOption()
	client.SetOptions(func(g *gosnmp.GoSNMP) {
		g.Context = c.ctx
		g.UseUnconnectedUDPSocket = useUnconnected
		c.auth.ConfigureSNMP(g, c.snmpContext)
	})

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// checkUnconnectedSocketOption 检查是否需要使用非连接模式
func (c Collector) checkUnconnectedSocketOption() bool {
	for _, m := range c.modules {
		if m.WalkParams.UseUnconnectedUDPSocket {
			return true
		}
	}
	return false
}

// handleWorkerError 处理工作协程错误
func (c Collector) handleWorkerError(ch chan<- prometheus.Metric, logger *logrus.Entry, err error) {
	logger.Info("Failed to create SNMP client", "err", err)
	ch <- prometheus.NewInvalidMetric(
		prometheus.NewDesc("snmp_error", "Error during worker initialization", nil, nil),
		err,
	)
}

// processModule 处理单个模块的采集任务
func (c Collector) processModule(ch chan<- prometheus.Metric, logger *logrus.Entry, client SNMPScraper, module *NamedModule) {
	moduleLogger := logger.WithField("module", module.name)
	moduleLogger.Debug("Starting scrape")

	start := time.Now()
	c.collect(ch, moduleLogger, client, module)

	duration := time.Since(start).Seconds()
	moduleLogger.Debug("Finished scrape", "duration_seconds", duration)

	c.metrics.SNMPCollectionDuration.WithLabelValues(module.name).Observe(duration)
}

// dispatchModules 分发模块到工作协程
func (c Collector) dispatchModules(ctx context.Context, workerChan chan *NamedModule) {
	for _, module := range c.modules {
		select {
		case <-ctx.Done():
			c.logger.Debug("Context canceled", "err", ctx.Err(), "module", module.name)
			return
		case workerChan <- module:
			c.logger.Debug("Sent module to worker", "module", module.name)
		}
	}
}

func getPduValue(pdu *gosnmp.SnmpPDU, wrapCount bool) float64 {
	switch pdu.Type {
	case gosnmp.Counter64:
		if wrapCount {
			// Wrap by 2^53.
			fmt.Printf("wrapCounters is true, wrapping counter64 value, value: %v\n", gosnmp.ToBigInt(pdu.Value).Uint64())
			return float64(gosnmp.ToBigInt(pdu.Value).Uint64() % float64Mantissa)
		}
		return float64(gosnmp.ToBigInt(pdu.Value).Uint64())
	case gosnmp.OpaqueFloat:
		return float64(pdu.Value.(float32))
	case gosnmp.OpaqueDouble:
		return pdu.Value.(float64)
	default:
		return float64(gosnmp.ToBigInt(pdu.Value).Int64())
	}
}

func parseDateAndTime(pdu *gosnmp.SnmpPDU) (float64, error) {
	// 1. 验证和提取PDU值
	v, err := extractPDUValue(pdu)
	if err != nil {
		return 0, err
	}

	// 2. 解析时区信息
	tz, err := parseTimeZone(v)
	if err != nil {
		return 0, err
	}

	// 3. 解析日期时间并转换为Unix时间戳
	timestamp, err := parseDateTime(v, tz)
	if err != nil {
		return 0, err
	}

	return timestamp, nil
}

// extractPDUValue 验证并提取PDU中的字节数据
func extractPDUValue(pdu *gosnmp.SnmpPDU) ([]byte, error) {
	v, ok := pdu.Value.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid DateAndTime type %T", pdu.Value)
	}

	if len(v) != 8 && len(v) != 11 {
		return nil, fmt.Errorf("invalid DateAndTime length %d", len(v))
	}

	return v, nil
}

// parseTimeZone 解析时区信息
func parseTimeZone(v []byte) (*time.Location, error) {
	if len(v) == 8 {
		return time.UTC, nil
	}

	locString := fmt.Sprintf("%s%02d%02d", string(v[8]), v[9], v[10])
	loc, err := time.Parse("-0700", locString)
	if err != nil {
		return nil, fmt.Errorf("error parsing timezone: %q, error: %w", locString, err)
	}
	return loc.Location(), nil
}

// parseDateTime 解析日期时间并转换为Unix时间戳
func parseDateTime(v []byte, tz *time.Location) (float64, error) {
	year := int(binary.BigEndian.Uint16(v[0:2]))
	month := time.Month(v[2])
	day := int(v[3])
	hour := int(v[4])
	minute := int(v[5])
	second := int(v[6])
	nanosecond := int(v[7]) * 1e8

	t := time.Date(
		year, month, day,
		hour, minute, second, nanosecond,
		tz,
	)

	return float64(t.Unix()), nil
}

func parseDateAndTimeWithPattern(metric *Metric, pdu *gosnmp.SnmpPDU, metrics Metrics) (float64, error) {
	pduValue := pduValueAsString(pdu, "DisplayString", metrics)
	t, err := timefmt.Parse(pduValue, metric.DateTimePattern)
	if err != nil {
		return 0, fmt.Errorf("error parsing date and time %q", err)
	}
	return float64(t.Unix()), nil
}

func parseNtpTimestamp(pdu *gosnmp.SnmpPDU) (float64, error) {
	var data = pdu.Value.([]byte)

	// Prometheus 使用 Unix 时间纪元(从1970年开始计算的秒数)
	// NTP 时间是从1900年开始计算的秒数，因此需要进行校正
	// 需要减去70年的秒数(1970-1900)，即2208988800秒
	secs := int64(binary.BigEndian.Uint32(data[:4])) - 2208988800
	nanos := (int64(binary.BigEndian.Uint32(data[4:])) * 1e9) >> 32

	t := time.Unix(secs, nanos)
	return float64(t.Unix()), nil
}

func pduToSamples(indexOids []int, pdu *gosnmp.SnmpPDU, metric *Metric, oidToPdu map[string]gosnmp.SnmpPDU, logger *logrus.Entry, metrics Metrics) []prometheus.Metric {
	// 1. 准备标签数据
	labels := indexesToLabels(indexOids, metric, oidToPdu, metrics)
	labelnames, labelvalues := prepareLabels(labels)

	// 2. 处理不同类型的指标
	switch {
	case isStandardMetricType(metric.Type):
		return handleStandardMetric(metric, pdu, labelnames, labelvalues, logger)
	case isComplexMetricType(metric.Type):
		return handleComplexMetric(metric, pdu, labelnames, labelvalues, logger, metrics, indexOids, oidToPdu)
	default:
		return handleStringMetric(metric, pdu, labelnames, labelvalues, labels, logger, metrics, indexOids, oidToPdu)
	}
}

// prepareLabels 准备标签名称和值
func prepareLabels(labels map[string]string) ([]string, []string) {
	labelnames := make([]string, 0, len(labels)+1)
	labelvalues := make([]string, 0, len(labels)+1)
	for k, v := range labels {
		labelnames = append(labelnames, k)
		labelvalues = append(labelvalues, v)
	}
	return labelnames, labelvalues
}

// isStandardMetricType 检查是否是标准指标类型
func isStandardMetricType(metricType string) bool {
	switch metricType {
	case "counter", "gauge", "Float", "Double":
		return true
	default:
		return false
	}
}

// handleStandardMetric 处理标准指标类型
func handleStandardMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, logger *logrus.Entry) []prometheus.Metric {
	value := getPduValue(pdu, wrapCounters)
	t := getValueType(metric.Type)

	if metric.Scale != 0.0 {
		value *= metric.Scale
	}
	value += metric.Offset
	logger.Debug("Handling standard metric")

	return createMetric(metric, t, value, labelnames, labelvalues)
}

// isComplexMetricType 检查是否是需要特殊处理的指标类型
func isComplexMetricType(metricType string) bool {
	switch metricType {
	case "DateAndTime", "ParseDateAndTime", "NTPTimeStamp", "EnumAsInfo", "EnumAsStateSet", "Bits":
		return true
	default:
		return false
	}
}

// handleComplexMetric 处理特殊指标类型
func handleComplexMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, logger *logrus.Entry, metrics Metrics, indexOids []int, oidToPdu map[string]gosnmp.SnmpPDU) []prometheus.Metric {
	switch metric.Type {
	case "DateAndTime":
		return handleDateTimeMetric(metric, pdu, labelnames, labelvalues, logger)
	case "ParseDateAndTime":
		return handleParseDateTimeMetric(metric, pdu, labelnames, labelvalues, logger, metrics)
	case "NTPTimeStamp":
		return handleNtpTimestampMetric(metric, pdu, labelnames, labelvalues, logger)
	case "EnumAsInfo":
		return enumAsInfo(metric, int(getPduValue(pdu, wrapCounters)), labelnames, labelvalues)
	case "EnumAsStateSet":
		return enumAsStateSet(metric, int(getPduValue(pdu, wrapCounters)), labelnames, labelvalues)
	case "Bits":
		return bits(metric, pdu.Value, labelnames, labelvalues)
	default:
		return nil
	}
}

// handleStringMetric 处理字符串类型指标
func handleStringMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, labels map[string]string, logger *logrus.Entry, metrics Metrics, indexOids []int, oidToPdu map[string]gosnmp.SnmpPDU) []prometheus.Metric {
	metricType := determineMetricType(metric, indexOids, oidToPdu, logger)

	if len(metric.RegexpExtracts) > 0 {
		return applyRegexExtracts(metric, pduValueAsString(pdu, metricType, metrics), labelnames, labelvalues, logger)
	}

	if _, ok := labels[metric.Name]; !ok {
		labelnames = append(labelnames, metric.Name)
		labelvalues = append(labelvalues, pduValueAsString(pdu, metricType, metrics))
	}

	return createMetric(metric, prometheus.GaugeValue, 1.0, labelnames, labelvalues)
}

func getValueType(metricType string) prometheus.ValueType {
	switch metricType {
	case "counter":
		return prometheus.CounterValue
	default: // "gauge", "Float", "Double"
		return prometheus.GaugeValue
	}
}

// handleNtpTimestampMetric 处理NTP时间戳类型的指标
func handleNtpTimestampMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, logger *logrus.Entry) []prometheus.Metric {
	value, err := parseNtpTimestamp(pdu)
	if err != nil {
		logger.Debug("Error parsing NTPTimeStamp", "err", err)
		return nil
	}
	return createMetric(metric, prometheus.GaugeValue, value, labelnames, labelvalues)
}

func handleDateTimeMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, logger *logrus.Entry) []prometheus.Metric {
	value, err := parseDateAndTime(pdu)
	if err != nil {
		logger.Debug("Error parsing DateAndTime", "err", err)
		return nil
	}
	return createMetric(metric, prometheus.GaugeValue, value, labelnames, labelvalues)
}

// handleParseDateTimeMetric 处理ParseDateAndTime类型的指标
func handleParseDateTimeMetric(metric *Metric, pdu *gosnmp.SnmpPDU, labelnames, labelvalues []string, logger *logrus.Entry, metrics Metrics) []prometheus.Metric {
	value, err := parseDateAndTimeWithPattern(metric, pdu, metrics)
	if err != nil {
		logger.Debug("Error parsing ParseDateAndTime", "err", err)
		return nil
	}
	return createMetric(metric, prometheus.GaugeValue, value, labelnames, labelvalues)
}

func determineMetricType(metric *Metric, indexOids []int, oidToPdu map[string]gosnmp.SnmpPDU, logger *logrus.Entry) string {
	metricType := metric.Type
	if typeMapping, ok := combinedTypeMapping[metricType]; ok {
		prevOid := fmt.Sprintf("%s.%s", getPrevOid(metric.Oid), listToOid(indexOids))
		if prevPdu, ok := oidToPdu[prevOid]; ok {
			if t, ok := typeMapping[int(getPduValue(&prevPdu, wrapCounters))]; ok {
				return t
			}
		}
		return "OctetString"
	}
	logger.Debug("determining metric type")
	return metricType
}

func createMetric(metric *Metric, valueType prometheus.ValueType, value float64, labelnames, labelvalues []string) []prometheus.Metric {
	sample, err := prometheus.NewConstMetric(
		prometheus.NewDesc(metric.Name, metric.Help, labelnames, nil),
		valueType, value, labelvalues...)
	if err != nil {
		sample = prometheus.NewInvalidMetric(
			prometheus.NewDesc("snmp_error", "Error calling NewConstMetric", nil, nil),
			fmt.Errorf("error for metric %s with labels %v: %v", metric.Name, labelvalues, err))
	}
	return []prometheus.Metric{sample}
}

func applyRegexExtracts(metric *Metric, pduValue string, labelnames, labelvalues []string, logger *logrus.Entry) []prometheus.Metric {
	var results []prometheus.Metric

	for name, strMetrics := range metric.RegexpExtracts {
		if metric := extractMetricFromRegexes(strMetrics, pduValue, metric, labelnames, labelvalues, logger, name); metric != nil {
			results = append(results, metric)
		}
	}

	return results
}

func extractMetricFromRegexes(strMetrics []RegexpExtract, pduValue string, metric *Metric, labelnames, labelvalues []string, logger *logrus.Entry, name string) prometheus.Metric {
	for _, strMetric := range strMetrics {
		value, err := extractAndParseRegexMatch(strMetric, pduValue, logger, metric.Name)
		if err != nil {
			continue
		}

		return createRegexMetric(metric.Name+name, metric.Help, value, labelnames, labelvalues)
	}
	return nil
}

func extractAndParseRegexMatch(strMetric RegexpExtract, pduValue string, logger *logrus.Entry, metricName string) (float64, error) {
	indexes := strMetric.Regex.FindStringSubmatchIndex(pduValue)
	if indexes == nil {
		logger.Debug("No match found for regexp",
			"metric", metricName,
			"value", pduValue,
			"regex", strMetric.Regex.String())
		return 0, fmt.Errorf("no match")
	}

	res := strMetric.Regex.ExpandString(nil, strMetric.Value, pduValue, indexes)
	return strconv.ParseFloat(string(res), 64)
}

func createRegexMetric(name, help string, value float64, labelnames, labelvalues []string) prometheus.Metric {
	desc := prometheus.NewDesc(
		name,
		help+" (regex extracted)",
		labelnames,
		nil,
	)

	metric, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		labelvalues...,
	)

	if err != nil {
		return prometheus.NewInvalidMetric(
			prometheus.NewDesc("snmp_error", "Error calling NewConstMetric for regex_extract", nil, nil),
			fmt.Errorf("error for metric %s with labels %v: %v", name, labelvalues, err),
		)
	}

	return metric
}

func enumAsInfo(metric *Metric, value int, labelnames, labelvalues []string) []prometheus.Metric {
	// Lookup enum, default to the value.
	state, ok := metric.EnumValues[int(value)]
	if !ok {
		state = strconv.Itoa(int(value))
	}
	labelnames = append(labelnames, metric.Name)
	labelvalues = append(labelvalues, state)

	newMetric, err := prometheus.NewConstMetric(prometheus.NewDesc(metric.Name+"_info", metric.Help+" (EnumAsInfo)", labelnames, nil),
		prometheus.GaugeValue, 1.0, labelvalues...)
	if err != nil {
		newMetric = prometheus.NewInvalidMetric(prometheus.NewDesc("snmp_error", "Error calling NewConstMetric for EnumAsInfo", nil, nil),
			fmt.Errorf("error for metric %s with labels %v: %v", metric.Name, labelvalues, err))
	}
	return []prometheus.Metric{newMetric}
}

func enumAsStateSet(metric *Metric, value int, labelnames, labelvalues []string) []prometheus.Metric {
	// 准备最终的标签名
	finalLabels := append(labelnames, metric.Name)
	results := make([]prometheus.Metric, 0, len(metric.EnumValues))

	// 获取当前状态值
	currentState := getStateString(metric, value)

	// 添加当前状态的metric (值为1)
	results = appendMetric(results, metric, finalLabels, labelvalues, currentState, 1.0)

	// 添加其他状态的metric (值为0)
	for stateVal, stateStr := range metric.EnumValues {
		if stateVal != value {
			results = appendMetric(results, metric, finalLabels, labelvalues, stateStr, 0.0)
		}
	}

	return results
}

// getStateString 获取状态对应的字符串表示
func getStateString(metric *Metric, value int) string {
	if state, ok := metric.EnumValues[value]; ok {
		return state
	}
	return strconv.Itoa(value)
}

// appendMetric 创建并添加metric到结果集
func appendMetric(results []prometheus.Metric, metric *Metric, labelnames, labelvalues []string, state string, value float64) []prometheus.Metric {
	// 准备完整的标签值
	fullLabelvalues := append(labelvalues, state)

	// 创建metric描述
	desc := prometheus.NewDesc(
		metric.Name,
		metric.Help+" (EnumAsStateSet)",
		labelnames,
		nil,
	)

	var newMetric prometheus.Metric

	// 尝试创建metric
	newMetric, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		value,
		fullLabelvalues...,
	)

	// 处理错误情况
	if err != nil {
		newMetric = prometheus.NewInvalidMetric(
			prometheus.NewDesc("snmp_error", "Error calling NewConstMetric for EnumAsStateSet", nil, nil),
			fmt.Errorf("error for metric %s with labels %v: %v", metric.Name, fullLabelvalues, err),
		)
	}

	return append(results, newMetric)
}

func bits(metric *Metric, value interface{}, labelnames, labelvalues []string) []prometheus.Metric {
	// 1. 类型检查
	bytes, ok := validateBitStringType(value)
	if !ok {
		return createInvalidBitMetric(metric, labelvalues, value)
	}

	// 2. 准备指标标签
	labelnames = append(labelnames, metric.Name)
	results := make([]prometheus.Metric, 0, len(metric.EnumValues))

	// 3. 处理每个bit位
	for bitPosition, bitName := range metric.EnumValues {
		bitValue := calculateBitValue(bytes, bitPosition)
		results = appendBitMetric(results, metric, labelnames, labelvalues, bitName, bitValue)
	}

	return results
}

// validateBitStringType 检查输入是否为有效的字节数组
func validateBitStringType(value interface{}) ([]byte, bool) {
	bytes, ok := value.([]byte)
	return bytes, ok
}

// createInvalidBitMetric 创建无效指标的提示
func createInvalidBitMetric(metric *Metric, labelvalues []string, value interface{}) []prometheus.Metric {
	errDesc := prometheus.NewDesc(
		"snmp_error",
		"BITS type was not a BITSTRING on the wire.",
		nil, nil,
	)
	err := fmt.Errorf("error for metric %s with labels %v: %T",
		metric.Name, labelvalues, value)

	return []prometheus.Metric{prometheus.NewInvalidMetric(errDesc, err)}
}

// calculateBitValue 计算指定位的值(0或1)
func calculateBitValue(bytes []byte, position int) float64 {
	if position >= len(bytes)*8 {
		return 0.0
	}

	byteIndex := position / 8
	bitOffset := position % 8
	mask := byte(128 >> bitOffset)

	if bytes[byteIndex]&mask != 0 {
		return 1.0
	}
	return 0.0
}

// appendBitMetric 创建并添加单个bit位的指标
func appendBitMetric(
	results []prometheus.Metric,
	metric *Metric,
	labelnames, labelvalues []string,
	bitName string,
	bitValue float64,
) []prometheus.Metric {
	desc := prometheus.NewDesc(
		metric.Name,
		metric.Help+" (Bits)",
		labelnames,
		nil,
	)

	metricLabels := append(labelvalues, bitName)
	newMetric, err := prometheus.NewConstMetric(
		desc,
		prometheus.GaugeValue,
		bitValue,
		metricLabels...,
	)

	if err != nil {
		errDesc := prometheus.NewDesc(
			"snmp_error",
			"Error calling NewConstMetric for Bits",
			nil, nil,
		)
		newMetric = prometheus.NewInvalidMetric(
			errDesc,
			fmt.Errorf("error for metric %s with labels %v: %v",
				metric.Name, metricLabels, err),
		)
	}

	return append(results, newMetric)
}

// Right pad oid with zeros, and split at the given point.
// Some routers exclude trailing 0s in responses.
func splitOid(oid []int, count int) ([]int, []int) {
	head := make([]int, count)
	tail := []int{}
	for i, v := range oid {
		if i < count {
			head[i] = v
		} else {
			tail = append(tail, v)
		}
	}
	return head, tail
}

// This mirrors decodeValue in gosnmp's helper.go.
func pduValueAsString(pdu *gosnmp.SnmpPDU, typ string, metrics Metrics) string {
	switch pdu.Value.(type) {
	case int:
		return strconv.Itoa(pdu.Value.(int))
	case uint:
		return strconv.FormatUint(uint64(pdu.Value.(uint)), 10)
	case uint64:
		return strconv.FormatUint(pdu.Value.(uint64), 10)
	case float32:
		return strconv.FormatFloat(float64(pdu.Value.(float32)), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(pdu.Value.(float64), 'f', -1, 64)
	case string:
		if pdu.Type == gosnmp.ObjectIdentifier {
			// Trim leading period.
			return pdu.Value.(string)[1:]
		}
		// DisplayString.
		return strings.ToValidUTF8(pdu.Value.(string), "�")
	case []byte:
		if typ == "" || typ == "Bits" {
			typ = "OctetString"
		}
		// Reuse the OID index parsing code.
		parts := make([]int, len(pdu.Value.([]byte)))
		for i, o := range pdu.Value.([]byte) {
			parts[i] = int(o)
		}
		if typ == "OctetString" || typ == "DisplayString" {
			// Prepend the length, as it is explicit in an index.
			parts = append([]int{len(pdu.Value.([]byte))}, parts...)
		}
		str, _, _ := indexOidsAsString(parts, typ, 0, false, nil)
		return strings.ToValidUTF8(str, "�")
	case nil:
		return ""
	default:
		// This shouldn't happen.
		metrics.SNMPUnexpectedPduType.Inc()
		return fmt.Sprintf("%s", pdu.Value)
	}
}

func indexOidsAsString(indexOids []int, typ string, fixedSize int, implied bool, enumValues map[int]string) (string, []int, []int) {
	// 处理复合类型映射
	if str, used, remaining, ok := handleCompositeType(typ, indexOids, enumValues); ok {
		return str, used, remaining
	}

	// 处理基本类型
	switch typ {
	case "Integer32", "Integer", "gauge", "counter":
		return handleNumericType(indexOids)
	case "PhysAddress48":
		return handleMACAddressType(indexOids)
	case "OctetString":
		return handleOctetStringType(indexOids, fixedSize, implied)
	case "DisplayString":
		return handleDisplayStringType(indexOids, fixedSize, implied)
	case "InetAddressIPv4":
		return handleIPv4Type(indexOids)
	case "InetAddressIPv6":
		return handleIPv6Type(indexOids)
	case "EnumAsInfo":
		return handleEnumType(indexOids, enumValues)
	default:
		panic(fmt.Sprintf("Unknown index type %s", typ))
	}
}

// handleCompositeType 处理复合类型映射
func handleCompositeType(typ string, indexOids []int, enumValues map[int]string) (string, []int, []int, bool) {
	typeMapping, ok := combinedTypeMapping[typ]
	if !ok {
		return "", nil, nil, false
	}

	splitCount := 2
	if typ == "InetAddressMissingSize" {
		splitCount = 1
	}
	subOid, valueOids := splitOid(indexOids, splitCount)

	// 处理已知子类型
	if t, ok := typeMapping[subOid[0]]; ok {
		str, used, remaining := indexOidsAsString(valueOids, t, 0, false, enumValues)
		return str, append(subOid, used...), remaining, true
	}

	// 特殊处理InetAddressMissingSize类型
	if typ == "InetAddressMissingSize" {
		str, used, remaining := indexOidsAsString(indexOids, "OctetString", 0, true, enumValues)
		return str, used, remaining, ok
	}

	// 默认处理为OctetString，第2个oid是长度
	str, used, remaining := indexOidsAsString(indexOids, "OctetString", subOid[1]+2, false, enumValues)
	return str, used, remaining, ok
}

// handleNumericType 处理数值类型
func handleNumericType(indexOids []int) (string, []int, []int) {
	subOid, remaining := splitOid(indexOids, 1)
	return fmt.Sprintf("%d", subOid[0]), subOid, remaining
}

// handleMACAddressType 处理MAC地址类型
func handleMACAddressType(indexOids []int) (string, []int, []int) {
	subOid, remaining := splitOid(indexOids, 6)
	parts := make([]string, 6)
	for i, o := range subOid {
		parts[i] = fmt.Sprintf("%02X", o)
	}
	return strings.Join(parts, ":"), subOid, remaining
}

// handleOctetStringType 处理八位组字符串类型
func handleOctetStringType(indexOids []int, fixedSize int, implied bool) (string, []int, []int) {
	length := calculateLength(indexOids, fixedSize, implied)
	subOid, content, remaining := extractStringContent(indexOids, length)

	parts := make([]byte, len(content))
	for i, o := range content {
		parts[i] = byte(o)
	}

	if len(parts) == 0 {
		return "", subOid, remaining
	}
	return fmt.Sprintf("0x%X", string(parts)), append(subOid, content...), remaining
}

// handleDisplayStringType 处理显示字符串类型
func handleDisplayStringType(indexOids []int, fixedSize int, implied bool) (string, []int, []int) {
	length := calculateLength(indexOids, fixedSize, implied)
	subOid, content, remaining := extractStringContent(indexOids, length)

	parts := make([]byte, len(content))
	for i, o := range content {
		parts[i] = byte(o)
	}
	return string(parts), append(subOid, content...), remaining
}

// handleIPv4Type 处理IPv4地址类型
func handleIPv4Type(indexOids []int) (string, []int, []int) {
	subOid, remaining := splitOid(indexOids, 4)
	parts := make([]string, 4)
	for i, o := range subOid {
		parts[i] = strconv.Itoa(o)
	}
	return strings.Join(parts, "."), subOid, remaining
}

// handleIPv6Type 处理IPv6地址类型
func handleIPv6Type(indexOids []int) (string, []int, []int) {
	subOid, remaining := splitOid(indexOids, 16)
	parts := make([]interface{}, 16)
	for i, o := range subOid {
		parts[i] = o
	}
	return fmt.Sprintf("%02X%02X:%02X%02X:%02X%02X:%02X%02X:%02X%02X:%02X%02X:%02X%02X:%02X%02X", parts...), subOid, remaining
}

// handleEnumType 处理枚举类型
func handleEnumType(indexOids []int, enumValues map[int]string) (string, []int, []int) {
	subOid, remaining := splitOid(indexOids, 1)
	if value, ok := enumValues[subOid[0]]; ok {
		return value, subOid, remaining
	}
	return fmt.Sprintf("%d", subOid[0]), subOid, remaining
}

// calculateLength 计算字符串长度
func calculateLength(indexOids []int, fixedSize int, implied bool) int {
	if implied {
		return len(indexOids)
	}
	return fixedSize
}

// extractStringContent 提取字符串内容
func extractStringContent(indexOids []int, length int) ([]int, []int, []int) {
	var subOid []int
	if length == 0 {
		subOid, indexOids = splitOid(indexOids, 1)
		length = subOid[0]
	}
	content, remaining := splitOid(indexOids, length)
	return subOid, content, remaining
}

func getPrevOid(oid string) string {
	oids := strings.Split(oid, ".")
	i, _ := strconv.Atoi(oids[len(oids)-1])
	oids[len(oids)-1] = strconv.Itoa(i - 1)
	return strings.Join(oids, ".")
}

func indexesToLabels(indexOids []int, metric *Metric, oidToPdu map[string]gosnmp.SnmpPDU, metrics Metrics) map[string]string {
	labels := make(map[string]string)
	labelOids := make(map[string][]int)

	// 处理索引转换
	processIndexes(indexOids, metric, labels, labelOids)

	// 处理查找转换
	processLookups(metric, oidToPdu, labels, labelOids, metrics)

	return labels
}

func processIndexes(indexOids []int, metric *Metric, labels map[string]string, labelOids map[string][]int) []int {
	for _, index := range metric.Indexes {
		str, subOid, remaining := indexOidsAsString(
			indexOids,
			index.Type,
			index.FixedSize,
			index.Implied,
			index.EnumValues,
		)
		labels[index.Labelname] = str
		labelOids[index.Labelname] = subOid
		indexOids = remaining
	}
	return indexOids
}

func processLookups(metric *Metric, oidToPdu map[string]gosnmp.SnmpPDU,
	labels map[string]string, labelOids map[string][]int, metrics Metrics) {

	for _, lookup := range metric.Lookups {
		if len(lookup.Labels) == 0 {
			delete(labels, lookup.Labelname)
			continue
		}

		oid := buildLookupOID(lookup, labelOids)
		if pdu, exists := oidToPdu[oid]; exists {
			processValidLookup(lookup, oidToPdu, labelOids, labels, pdu, metrics)
		} else {
			labels[lookup.Labelname] = ""
		}
	}
}

func buildLookupOID(lookup *Lookup, labelOids map[string][]int) string {
	oid := lookup.Oid
	for _, label := range lookup.Labels {
		oid = fmt.Sprintf("%s.%s", oid, listToOid(labelOids[label]))
	}
	return oid
}

func processValidLookup(lookup *Lookup, oidToPdu map[string]gosnmp.SnmpPDU,
	labelOids map[string][]int, labels map[string]string,
	pdu gosnmp.SnmpPDU, metrics Metrics) {

	lookupType := determineLookupType(lookup, oidToPdu, labelOids)
	labels[lookup.Labelname] = pduValueAsString(&pdu, lookupType, metrics)
	labelOids[lookup.Labelname] = []int{int(gosnmp.ToBigInt(pdu.Value).Int64())}
}

func determineLookupType(lookup *Lookup, oidToPdu map[string]gosnmp.SnmpPDU,
	labelOids map[string][]int) string {

	if typeMapping, exists := combinedTypeMapping[lookup.Type]; exists {
		prevOid := buildPrevOID(lookup, labelOids)
		if prevPdu, exists := oidToPdu[prevOid]; exists {
			val := int(getPduValue(&prevPdu, wrapCounters))
			if resolvedType, exists := typeMapping[val]; exists {
				return resolvedType
			}
		}
	}
	return lookup.Type
}

func buildPrevOID(lookup *Lookup, labelOids map[string][]int) string {
	prevOid := getPrevOid(lookup.Oid)
	for _, label := range lookup.Labels {
		prevOid = fmt.Sprintf("%s.%s", prevOid, listToOid(labelOids[label]))
	}
	return prevOid
}
