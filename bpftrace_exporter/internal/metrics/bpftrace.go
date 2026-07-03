package metrics

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"bpftrace_exporter/internal/exporter"
	"bpftrace_exporter/pkg/bpftrace"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

func init() {
	exporter.Register(NewBpftraceExporter("bpftrace_exporter", "bpftrace exporter metrics", []string{}))
}

type BpftraceExporter struct {
	*baseMetrics
	mutex         sync.RWMutex
	numProbesDesc *prometheus.Desc
	process       *bpftrace.Process
	scriptName    string
	vars          map[string]*VarDef
}

type VarDef struct {
	VarType  int
	IsMap    bool
	Desc     *prometheus.Desc
	PromType prometheus.ValueType
}

func NewBpftraceExporter(fqname, help string, labels []string) *BpftraceExporter {
	return &BpftraceExporter{
		baseMetrics: NewMetrics(fqname, help, labels),
	}
}

func (be *BpftraceExporter) Initialize(bpftracePath string, scriptPath string, varDefs string) error {
	process, err := bpftrace.NewProcess(bpftracePath, scriptPath)
	if err != nil {
		return err
	}

	err = process.Start()
	if err != nil {
		return err
	}

	// 从路径提取脚本名
	parts := strings.Split(scriptPath, "/")
	scriptName := parts[len(parts)-1]
	scriptName = strings.Split(scriptName, ".")[0]
	
	numProbesDesc := prometheus.NewDesc(
		prometheus.BuildFQName("bpftrace", scriptName, "probes_total"),
		"number of attached probes",
		nil, nil,
	)

	vars, err := parseVarDefs(scriptName, varDefs)
	if err != nil {
		return err
	}

	be.numProbesDesc = numProbesDesc
	be.process = process
	be.scriptName = scriptName
	be.vars = vars

	return nil
}

func parseVarDefs(scriptName string, varDefs string) (map[string]*VarDef, error) {
	vars := map[string]*VarDef{}
	for _, varDef := range strings.Split(varDefs, ",") {
		if varDef == "" {
			continue
		}

		// example: `usecs:hist`
		s := strings.Split(varDef, ":")
		name := s[0]
		def := ""
		if len(s) > 1 {
			def = s[1]
		}

		switch def {
		case "":
			vars[name] = &VarDef{
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    false,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace variable @%s", name), nil, nil),
				PromType: prometheus.GaugeValue,
			}
		case "counter":
			vars[name] = &VarDef{
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    false,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace variable @%s", name), nil, nil),
				PromType: prometheus.CounterValue,
			}
		case "map":
			vars[name] = &VarDef{
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    true,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace map @%s", name), []string{"key"}, nil),
				PromType: prometheus.GaugeValue,
			}
		case "countermap":
			vars[name] = &VarDef{
				VarType:  bpftrace.VarTypeNumber,
				IsMap:    true,
				Desc:     prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace map @%s", name), []string{"key"}, nil),
				PromType: prometheus.CounterValue,
			}
		case "hist":
			vars[name] = &VarDef{
				VarType: bpftrace.VarTypeHistogram,
				IsMap:   false,
				Desc:    prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace histogram @%s", name), nil, nil),
			}
		case "histmap":
			vars[name] = &VarDef{
				VarType: bpftrace.VarTypeHistogram,
				IsMap:   true,
				Desc:    prometheus.NewDesc(prometheus.BuildFQName("bpftrace", scriptName, name), fmt.Sprintf("bpftrace histogram @%s", name), []string{"key"}, nil),
			}
		default:
			return nil, fmt.Errorf("unknown variable definition: \"%s\"", def)
		}
	}
	return vars, nil
}

func (be *BpftraceExporter) Describe(ch chan<- *prometheus.Desc) {
	if be.numProbesDesc != nil {
		ch <- be.numProbesDesc
	}
	if be.vars != nil {
		for _, metric := range be.vars {
			ch <- metric.Desc
		}
	}
}

func (be *BpftraceExporter) Collect(ch chan<- prometheus.Metric) {
	if be.process == nil || be.vars == nil {
		logrus.Warn("BpftraceExporter not initialized, skipping collection")
		return
	}

	be.mutex.Lock() // 保护并发收集
	defer be.mutex.Unlock()

	// 添加probes指标
	ch <- prometheus.MustNewConstMetric(be.numProbesDesc, prometheus.GaugeValue, float64(be.process.NumProbes))

	// 发送信号获取最新数据
	err := be.process.SendSigusr1()
	if err != nil {
		logrus.Errorf("Error sending signal: %v", err)
		return
	}

	var out bpftrace.Output
	// 一旦所有变量都处理完毕就结束循环
	for remVars := len(be.vars); remVars > 0 && be.process.StdoutScanner.Scan(); {
		line := be.process.StdoutScanner.Text()
		err := json.Unmarshal([]byte(line), &out)
		if err != nil {
			logrus.Errorf("Cannot parse JSON of line %s: %v", line, err)
			return
		}

		switch out.Type {
		case "printf":
			logrus.Infof("bpftrace output: %s", out.Data)
		case "map", "hist":
			var varData bpftrace.VarData
			err := json.Unmarshal(out.Data, &varData)
			if err != nil {
				logrus.Errorf("Cannot parse JSON of line %s: %v", line, err)
				return
			}

			for name, val := range varData {
				varDef, ok := be.vars[name[1:]]
				if ok {
					be.exportVar(ch, varDef, val)
					remVars--
				}
			}
		default:
			logrus.Warnf("Unknown output type: %s", out.Type)
		}
	}
}

func (be *BpftraceExporter) exportNumber(ch chan<- prometheus.Metric, varDef *VarDef, val json.RawMessage, labelValues ...string) {
	var bpfNum bpftrace.Number
	err := json.Unmarshal(val, &bpfNum)
	if err != nil {
		log.Printf("cannot unmarshal %s into a number: %v", val, err)
		return
	}

	ch <- prometheus.MustNewConstMetric(varDef.Desc, varDef.PromType, bpfNum, labelValues...)
}

func (be *BpftraceExporter) exportHist(ch chan<- prometheus.Metric, varDef *VarDef, val json.RawMessage, labelValues ...string) {
	var bpfHist bpftrace.Hist
	err := json.Unmarshal(val, &bpfHist)
	if err != nil {
		log.Printf("cannot unmarshal %s into a histogram: %v", val, err)
		return
	}

	buckets := map[float64]uint64{}
	count := uint64(0)

	// bpftrace buckets are already sorted
	for _, bucket := range bpfHist {
		upperBound := bucket.Max
		count += bucket.Count
		buckets[upperBound] = count
	}
	ch <- prometheus.MustNewConstHistogram(varDef.Desc, count, -1, buckets, labelValues...)
}

func (be *BpftraceExporter) exportScalarVar(ch chan<- prometheus.Metric, varDef *VarDef, val json.RawMessage, labelValues ...string) {
	switch varDef.VarType {
	case bpftrace.VarTypeNumber:
		be.exportNumber(ch, varDef, val, labelValues...)
	case bpftrace.VarTypeHistogram:
		be.exportHist(ch, varDef, val, labelValues...)
	}
}

func (be *BpftraceExporter) exportVar(ch chan<- prometheus.Metric, varDef *VarDef, val json.RawMessage) {
	if !varDef.IsMap {
		be.exportScalarVar(ch, varDef, val)
	} else {
		var mapData bpftrace.Map
		err := json.Unmarshal(val, &mapData)
		if err != nil {
			log.Printf("cannot unmarshal %s into a map: %v", val, err)
			return
		}

		for k, v := range mapData {
			be.exportScalarVar(ch, varDef, v, k)
		}
	}
}
// Final commit for bpftrace_exporter/internal/metrics/bpftrace.go
