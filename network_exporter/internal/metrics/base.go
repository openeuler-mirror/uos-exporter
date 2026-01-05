package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// metricInfo 存储单个metric的信息
type metricInfo struct {
	desc       *prometheus.Desc
	labelNames []string
	values     map[string]float64 // key是标签值的组合
}

// baseMetrics 基础metrics结构
type baseMetrics struct {
	prefix  string
	metrics map[string]*metricInfo
	mutex   sync.RWMutex
}

// newBaseMetrics 创建新的baseMetrics实例
func newBaseMetrics(prefix string) *baseMetrics {
	return &baseMetrics{
		prefix:  prefix,
		metrics: make(map[string]*metricInfo),
	}
}

// addMetric 添加一个metric定义
func (b *baseMetrics) addMetric(name, help string, labels []string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	fqName := b.prefix + "_" + name
	desc := prometheus.NewDesc(fqName, help, labels, nil)
	
	b.metrics[name] = &metricInfo{
		desc:       desc,
		labelNames: labels,
		values:     make(map[string]float64),
	}
}

// setMetric 设置metric的值（无标签）
func (b *baseMetrics) setMetric(name string, value float64) {
	b.setMetricWithLabels(name, value, nil)
}

// setMetricWithLabels 设置metric的值（带标签）
func (b *baseMetrics) setMetricWithLabels(name string, value float64, labelValues map[string]string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	
	metric, exists := b.metrics[name]
	if !exists {
		return
	}
	
	// 构建标签值数组和key
	labelArray := make([]string, len(metric.labelNames))
	for i, labelName := range metric.labelNames {
		if labelValues != nil {
			labelArray[i] = labelValues[labelName]
		}
	}
	
	key := b.labelsToKey(labelArray)
	metric.values[key] = value
}

// labelsToKey 将标签数组转换为key
func (b *baseMetrics) labelsToKey(labels []string) string {
	key := ""
	for i, label := range labels {
		if i > 0 {
			key += "|"
		}
		key += label
	}
	return key
}

// Describe 实现prometheus.Collector接口
func (b *baseMetrics) Describe(ch chan<- *prometheus.Desc) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	for _, metric := range b.metrics {
		ch <- metric.desc
	}
}

// Collect 实现prometheus.Collector接口
func (b *baseMetrics) Collect(ch chan<- prometheus.Metric) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	
	for _, metric := range b.metrics {
		for labelKey, value := range metric.values {
			var labels []string
			if labelKey != "" {
				labels = b.keyToLabels(labelKey)
			}
			
			ch <- prometheus.MustNewConstMetric(
				metric.desc,
				prometheus.GaugeValue,
				value,
				labels...,
			)
		}
	}
}

// keyToLabels 将key转换回标签数组
func (b *baseMetrics) keyToLabels(key string) []string {
	if key == "" {
		return []string{}
	}
	// 按分隔符拆分
	labels := []string{}
	parts := []rune(key)
	current := ""
	for _, char := range parts {
		if char == '|' {
			labels = append(labels, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		labels = append(labels, current)
	}
	return labels
} 