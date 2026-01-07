package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// The Prefix for labels of this prometheus exporter
const EXPORTER_LABEL_PREFIX = "samba"

// SambaExporter - The class that implements the Prometheus Exporter Interface
type SambaExporter struct {
	RequestHandler *PipeHandler
	ResponseHander *PipeHandler
	Version        string
	RequestTimeOut int
	parms          Parmeters

	// Used to ensure that every metric is only added once
	descriptions map[string]prometheus.Desc

	// Used to ensure that the order of labels is always the same for a given metric
	metricsLabelList map[string][]string
}

// Get a new instance of the SambaExporter
func NewSambaExporter(requestHandler *PipeHandler, responseHander *PipeHandler, version string, requestTimeOut int, parms Parmeters) *SambaExporter {
	var ret SambaExporter
	ret.RequestHandler = requestHandler
	ret.ResponseHander = responseHander
	ret.Version = version
	ret.RequestTimeOut = requestTimeOut
	ret.descriptions = make(map[string]prometheus.Desc)
	ret.parms = parms
	ret.metricsLabelList = make(map[string][]string)

	return &ret
}

// Describe function for the Prometheus Exporter Interface
func (smbExporter *SambaExporter) Describe(ch chan<- *prometheus.Desc) {
	logrus.Info("Request samba_statusd to get prometheus descriptions")
	locks, processes, shares, psData, errGet := GetSambaStatus(smbExporter.RequestHandler, smbExporter.ResponseHander, smbExporter.RequestTimeOut)
	if errGet != nil {
		logrus.Error(errGet)

		// Exit with panic, since this means there are no descriptions setup for further operation
		panic(errGet)
	}
	smbExporter.setDescriptionsFromResponse(locks, processes, shares, psData, ch)
}

// Collect function for the Prometheus Exporter Interface
func (smbExporter *SambaExporter) Collect(ch chan<- prometheus.Metric) {
	logrus.Info("Request samba_statusd to get prometheus metrics")
	smbStatusUp := 1
	smbServerUp := 1
	start := time.Now()
	locks, processes, shares, psData, errGet := GetSambaStatus(smbExporter.RequestHandler, smbExporter.ResponseHander, smbExporter.RequestTimeOut)
	if errGet != nil {
		logrus.Error(errGet)
		switch errGet.(type) {
		case *SmbStatusTimeOutError:
			smbStatusUp = 0
			smbServerUp = 0
		case *SmbStatusUnexpectedResponseError:
			smbServerUp = 0
		default:
			return
		}
	}
	elapsed := time.Since(start)
	elapsedFloat := float64(elapsed.Milliseconds())
	smbExporter.setMetricsFromResponse(locks, processes, shares, psData, smbStatusUp, smbServerUp, elapsedFloat, ch)

	// return
}

func (smbExporter *SambaExporter) setMetricsFromResponse(locks []LockData, processes []ProcessData, shares []ShareData, psData []PsUtilPidData, smbStatusUp int, smbServerUp int, requestTime float64, ch chan<- prometheus.Metric) {
	logrus.Info("Handle samba_statusd response and set prometheus metrics")
	smbExporter.setGaugeIntMetricNoLabel("server_up", float64(smbServerUp), ch)
	smbExporter.setGaugeIntMetricNoLabel("satutsd_up", float64(smbStatusUp), ch)
	smbExporter.setGaugeIntMetricWithLabel("exporter_information", 1, map[string]string{"version": smbExporter.Version}, ch)

	stats := GetSmbStatistics(locks, processes, shares, smbExporter.parms)
	if stats == nil {
		logrus.Error(NewSmbStatusUnexpectedResponseError("Empty response from samba_statusd"))
		return
	}
	stats = append(stats, GetSmbdMetrics(psData, smbExporter.parms.Do_not_expose_pid)...)
	for _, stat := range stats {
		// logrus.Infof("start create metrics %v", stat.Value)
		if stat.Labels == nil {
			smbExporter.setGaugeIntMetricNoLabel(stat.Name, stat.Value, ch)
		} else {
			smbExporter.setGaugeIntMetricWithLabel(stat.Name, stat.Value, stat.Labels, ch)
		}
	}

	smbExporter.setGaugeIntMetricNoLabel("request_time", requestTime, ch)
}

func (smbExporter *SambaExporter) setDescriptionsFromResponse(locks []LockData, processes []ProcessData, shares []ShareData, psData []PsUtilPidData, ch chan<- *prometheus.Desc) {
	logrus.Info("Handle samba_statusd response and set prometheus descriptions")
	stats := GetSmbStatistics(locks, processes, shares, smbExporter.parms)
	if stats == nil {
		err := NewSmbStatusUnexpectedResponseError("Empty response from samba_statusd")
		logrus.Error(err)

		// Exit with panic, since this means there are no descriptions setup for further operation
		panic(err)
	}
	stats = append(stats, GetSmbdMetrics(psData, smbExporter.parms.Do_not_expose_pid)...)

	smbExporter.setGaugeDescriptionNoLabel("server_up", "1 if the samba server seems to be running", ch)
	smbExporter.setGaugeDescriptionNoLabel("satutsd_up", "1 if the samba_statusd seems to be running", ch)
	smbExporter.setGaugeDescriptionWithLabel("exporter_information", "Information of the samba_exporter", map[string]string{"version": smbExporter.Version}, ch)

	for _, stat := range stats {
		if stat.Labels == nil {
			smbExporter.setGaugeDescriptionNoLabel(stat.Name, stat.Help, ch)
		} else {
			smbExporter.setGaugeDescriptionWithLabel(stat.Name, stat.Help, stat.Labels, ch)
		}
	}

	smbExporter.setGaugeDescriptionNoLabel("request_time", "Time it took to reqest the samba status from samba_statusd [ms]", ch)
}

func (smbExporter *SambaExporter) setGaugeIntMetricNoLabel(name string, value float64, ch chan<- prometheus.Metric) {
	desc, found := smbExporter.descriptions[name]
	if !found {
		logrus.Error(fmt.Sprintf("No description found for %s", name))
		return
	}

	met := prometheus.MustNewConstMetric(&desc, prometheus.GaugeValue, value)
	ch <- met
}

func (smbExporter *SambaExporter) setGaugeIntMetricWithLabel(name string, value float64, labels map[string]string, ch chan<- prometheus.Metric) {
	desc, found := smbExporter.descriptions[name]
	if !found {
		logrus.Error(fmt.Sprintf("No description found for metric '%s'", name))
		return
	}

	// Ensure the expected order of labels is known for this metric (see bug #79)
	labelKeys, foundInLabelList := smbExporter.metricsLabelList[name]
	if !foundInLabelList {
		logrus.Error(fmt.Sprintf("No label keys found for metric '%s'", name))
		return
	}

	// Validate that the given labels list is the same length as the expected label list for this metric (see bug #79)
	if len(labelKeys) != len(labels) {
		logrus.Error(fmt.Sprintf(
			"The number of labels given with metric '%s' ('%d') does not match the expected number '%d'",
			name, len(labels), len(labelKeys)))
		return
	}

	labelValues := make([]string, len(labelKeys))
	// The loop over the expected label is done explicit form 0 to end, so the order can not be lost (see bug #79)
	for i := 0; i < len(labelKeys); i++ {
		key := labelKeys[i]
		value, foundValue := labels[key]
		if !foundValue {
			logrus.Error(fmt.Sprintf("No label with key '%s' found for metric '%s'", key, name))
			return
		}
		if value != "" {
			// The set is done to a explicit field to ensure the order of labels is not lost (see bug #79)
			labelValues[i] = value
		} else {
			// if a labels value is "", we don't add the value at all
			return
		}
	}
	// logrus.Infof("create metrics %v", value)
	met := prometheus.MustNewConstMetric(&desc, prometheus.GaugeValue, value, labelValues...)
	ch <- met
}

func (smbExporter *SambaExporter) setGaugeDescriptionNoLabel(name string, help string, ch chan<- *prometheus.Desc) {
	desc := prometheus.NewDesc(prometheus.BuildFQName(EXPORTER_LABEL_PREFIX, "", name), help, []string{}, nil)
	smbExporter.descriptions[name] = *desc
	ch <- desc
}

func (smbExporter *SambaExporter) setGaugeDescriptionWithLabel(name string, help string, labels map[string]string, ch chan<- *prometheus.Desc) {
	// Since the a the same label can have multiple values, we need only one description
	_, found := smbExporter.descriptions[name]

	if !found {
		var labelKeys []string
		for key, _ := range labels {
			labelKeys = append(labelKeys, key)
		}

		smbExporter.metricsLabelList[name] = labelKeys
		desc := prometheus.NewDesc(prometheus.BuildFQName(EXPORTER_LABEL_PREFIX, "", name), help, labelKeys, nil)
		smbExporter.descriptions[name] = *desc
		ch <- desc
	}
}
