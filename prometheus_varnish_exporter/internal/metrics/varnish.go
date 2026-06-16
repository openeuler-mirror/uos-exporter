package metrics

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"prometheus_varnish_exporter/pkg/utils"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	vbeReloadPrefix = "VBE.reload_"
)

var (
	DescCache = &descCache{
		descs: make(map[string]*prometheus.Desc),
	}
)

type descCache struct {
	sync.RWMutex

	descs map[string]*prometheus.Desc
}

func (dc *descCache) Desc(key string) *prometheus.Desc {
	dc.RLock()
	desc := dc.descs[key]
	dc.RUnlock()
	return desc
}

func (dc *descCache) Set(key string, desc *prometheus.Desc) *prometheus.Desc {
	dc.Lock()
	dc.descs[key] = desc
	dc.Unlock()
	return desc
}

func ScrapeVarnish(ch chan<- prometheus.Metric) ([]byte, error) {
	params := []string{"-j"}
	if VarnishVersion.EqualsOrGreater(4, 1) {
		params = append(params, "-t", "0")
	}
	if !StartParams.Params.isEmpty() {
		params = append(params, StartParams.Params.make()...)
	}
	buf, errExec := executeVarnishstat(StartParams.VarnishstatExe, params...)
	if errExec != nil {
		return buf.Bytes(), fmt.Errorf("%s scrape failed: %s", StartParams.VarnishstatExe, errExec)
	}
	return ScrapeVarnishFrom(buf.Bytes(), ch)
}

func ScrapeVarnishFrom(buf []byte, ch chan<- prometheus.Metric) ([]byte, error) {
	metricsJSON := make(map[string]interface{})
	dec := json.NewDecoder(bytes.NewBuffer(buf))
	dec.UseNumber()
	if err := dec.Decode(&metricsJSON); err != nil {
		return buf, err
	}

	countersJSON := make(map[string]interface{})
	if metricsJSON["version"] != nil {
		versionRaw, ok := metricsJSON["version"].(json.Number)
		if !ok {
			return nil, fmt.Errorf("unhandled json stats version type: %T %#v", metricsJSON["version"], metricsJSON["version"])
		}
		version, err := versionRaw.Int64()
		if err != nil {
			return nil, fmt.Errorf("unhandled json stats version type: %s", err)
		}
		switch version {
		case 1:
			countersJSON = metricsJSON["counters"].(map[string]interface{})
		default:
			return nil, fmt.Errorf("unimplemented json stats version %d", version)
		}
	} else {
		countersJSON = metricsJSON
	}

	mostRecentVbeReloadPrefix := findMostRecentVbeReloadPrefix(countersJSON)

	for vName, raw := range countersJSON {
		if isOutdatedVbe(vName, mostRecentVbeReloadPrefix) {
			continue
		}
		if vName == "timestamp" {
			continue
		}
		if dt := reflect.TypeOf(raw); dt.Kind() != reflect.Map {
			if StartParams.Verbose {
				logWarn("found unexpected data from json: %s: %#v", vName, raw)
			}
			continue
		}
		data, ok := raw.(map[string]interface{})
		if !ok {
			if StartParams.Verbose {
				logWarn("failed to cast to map[string]interface{}: %s: %#v", vName, raw)
			}
			continue
		}
		processMetric(vName, data, ch)
	}
	return buf, nil
}

func processMetric(vName string, data map[string]interface{}, ch chan<- prometheus.Metric) {
	var (
		vGroup       = prometheusGroup(vName)
		vDescription string
		vIdentifier  string
		vValue       float64
		iValue       uint64
		vErr         error
	)
	flag, _ := stringProperty(data, "flag")

	if value, ok := data["description"]; ok && vErr == nil {
		if vDescription, ok = value.(string); !ok {
			vErr = fmt.Errorf("%s description is not a string", vName)
		}
	}
	if value, ok := data["ident"]; ok && vErr == nil {
		if vIdentifier, ok = value.(string); !ok {
			vErr = fmt.Errorf("%s ident is not a string", vName)
		}
	}
	if value, ok := data["value"]; ok && vErr == nil {
		if number, ok := value.(json.Number); ok {
			if vValue, vErr = number.Float64(); vErr != nil {
				vErr = fmt.Errorf("%s value float64 error: %s", vName, vErr)
			}
			if flag == "b" {
				if iValue, vErr = strconv.ParseUint(number.String(), 10, 64); vErr != nil {
					vErr = fmt.Errorf("%s value uint64 error: %s", vName, vErr)
				}
			}
		} else {
			vErr = fmt.Errorf("%s value is not a float64", vName)
		}
	}
	if vErr != nil {
		if StartParams.Verbose {
			logWarn(vErr.Error())
		}
		return
	}

	pName, pDescription, pLabelKeys, pLabelValues := computePrometheusInfo(vName, vGroup, vIdentifier, vDescription)

	descKey := pName + "_" + strings.Join(pLabelKeys, "_")
	pDesc := DescCache.Desc(descKey)
	if pDesc == nil {
		pDesc = DescCache.Set(descKey, prometheus.NewDesc(
			pName,
			pDescription,
			pLabelKeys,
			nil,
		))
	}

	var metricType prometheus.ValueType
	switch flag {
	case "c", "a":
		metricType = prometheus.CounterValue
	case "g":
		metricType = prometheus.GaugeValue
	default:
		metricType = prometheus.GaugeValue
	}

	ch <- prometheus.MustNewConstMetric(pDesc, metricType, vValue, pLabelValues...)

	if pName == "varnish_backend_happy" {
		processBackendUpMetric(pLabelKeys, pLabelValues, iValue, ch)
	}
}

func processBackendUpMetric(pLabelKeys, pLabelValues []string, iValue uint64, ch chan<- prometheus.Metric) {
	upName := "varnish_backend_up"
	upDesc := "Backend up as per the latest health probe"
	upValue := 0.0
	if iValue > 0 && (iValue&uint64(1)) > 0 {
		upValue = 1.0
	}

	descKey := upName + "_" + strings.Join(pLabelKeys, "_")
	pDesc := DescCache.Desc(descKey)
	if pDesc == nil {
		pDesc = DescCache.Set(descKey, prometheus.NewDesc(
			upName,
			upDesc,
			pLabelKeys,
			nil,
		))
	}
	ch <- prometheus.MustNewConstMetric(pDesc, prometheus.GaugeValue, upValue, pLabelValues...)
}

func executeVarnishstat(varnishstatExe string, params ...string) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	var cmd *exec.Cmd
	if len(StartParams.VarnishDockerContainer) == 0 {
		cmd = utils.GetCommand(varnishstatExe, params...)
	} else {
		cmd = utils.GetCommand("docker", append([]string{"exec", "-t", StartParams.VarnishDockerContainer, varnishstatExe}, params...)...)
	}
	cmd.Stdout = buf
	cmd.Stderr = buf
	return buf, cmd.Run()
}

func findMostRecentVbeReloadPrefix(countersJSON map[string]interface{}) string {
	var mostRecentVbeReloadPrefix string
	for vName := range countersJSON {
		if strings.HasPrefix(vName, vbeReloadPrefix) && strings.HasSuffix(vName, ".happy") {
			dotAfterPrefixIndex := len(vbeReloadPrefix) + strings.Index(vName[len(vbeReloadPrefix):], ".")
			vbeReloadPrefix := vName[:dotAfterPrefixIndex]
			if strings.Compare(vbeReloadPrefix, mostRecentVbeReloadPrefix) > 0 {
				mostRecentVbeReloadPrefix = vbeReloadPrefix
			}
		}
	}
	return mostRecentVbeReloadPrefix
}

func isOutdatedVbe(vName string, mostRecentVbeReloadPrefix string) bool {
	return len(mostRecentVbeReloadPrefix) > 0 &&
		strings.HasPrefix(vName, "VBE.") &&
		!strings.HasPrefix(vName, mostRecentVbeReloadPrefix)
}

type varnishVersion struct {
	Major    int
	Minor    int
	Patch    int
	Revision string
}

func NewVarnishVersion() *varnishVersion {
	return &varnishVersion{
		Major: -1, Minor: -1, Patch: -1,
	}
}

func (v *varnishVersion) EqualsOrGreater(major, minor int) bool {
	if v.Major > major {
		return true
	} else if v.Major == major && v.Minor >= minor {
		return true
	}
	return false
}

func (v *varnishVersion) Valid() bool {
	return v.Major != -1
}

func (v *varnishVersion) Initialize() error {
	return v.queryVersion()
}

func (v *varnishVersion) queryVersion() error {
	buf, err := executeVarnishstat(StartParams.VarnishstatExe, "-V")
	if err != nil {
		return err
	}
	if scanner := bufio.NewScanner(buf); scanner.Scan() {
		return v.parseVersion(scanner.Text())
	}
	return fmt.Errorf("failed to get varnishstat -V output")
}

func (v *varnishVersion) parseVersion(version string) error {
	r := regexp.MustCompile(`(?P<major>\d+)(\.(?P<minor>\d+))?(\.(?P<patch>\d+))?(.*revision\s(?P<revision>[0-9a-f]*)\))?`)
	match := r.FindStringSubmatch(version)

	parts := make(map[string]string)
	for i, name := range r.SubexpNames() {
		if i != 0 && name != "" {
			parts[name] = match[i]
		}
	}

	if len(parts) > 1 {
		if err := v.set(parts); err != nil {
			return err
		}
	}
	if !v.Valid() {
		return fmt.Errorf("failed to resolve version from %q", version)
	}
	return nil
}

func (v *varnishVersion) Labels() map[string]string {
	mark := make(map[string]string)
	if v.Major != -1 {
		mark["major"] = strconv.Itoa(v.Major)
	}
	if v.Minor != -1 {
		mark["minor"] = strconv.Itoa(v.Minor)
	}
	if v.Patch != -1 {
		mark["patch"] = strconv.Itoa(v.Patch)
	}
	if v.Revision != "" {
		mark["revision"] = v.Revision
	}
	mark["version"] = v.VersionString()
	return mark
}

func (v *varnishVersion) set(parts map[string]string) error {
	for name, value := range parts {
		if len(value) == 0 {
			continue
		}
		if name == "revision" {
			v.Revision = value
			continue
		}
		num, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		switch name {
		case "major":
			v.Major = num
		case "minor":
			v.Minor = num
		case "patch":
			v.Patch = num
		}
	}
	return nil
}

func (v *varnishVersion) VersionString() string {
	parts := []string{}
	for _, num := range []int{v.Major, v.Minor, v.Patch} {
		if num != -1 {
			parts = append(parts, strconv.Itoa(num))
		}
	}
	return strings.Join(parts, ".")
}

func (v *varnishVersion) String() string {
	version := v.VersionString()
	if v.Revision != "" {
		version += " " + v.Revision
	}
	return version
}
