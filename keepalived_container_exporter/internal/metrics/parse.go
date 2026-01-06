package metrics

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// 状态枚举常量
const (
	ScriptStatusGood = "GOOD"
	ScriptStatusBad  = "BAD"

	ScriptStateIdle          = "idle"
	ScriptStateRunning       = "running"
	ScriptStateRequestedTerm = "requested termination"
	ScriptStateForcingTerm   = "forcing termination"

	VRRPStateInit   = "INIT"
	VRRPStateBackup = "BACKUP"
	VRRPStateMaster = "MASTER"
	VRRPStateFault  = "FAULT"
)

// 状态转换器
type StateConverter struct{}

func (c *StateConverter) ScriptStatusToInt(status string) (int, bool) {
	switch status {
	case ScriptStatusGood:
		return 1, true
	case ScriptStatusBad:
		return 0, true
	default:
		return -1, false
	}
}

func (c *StateConverter) ScriptStateToInt(state string) (int, bool) {
	switch state {
	case ScriptStateIdle:
		return 0, true
	case ScriptStateRunning:
		return 1, true
	case ScriptStateRequestedTerm:
		return 2, true
	case ScriptStateForcingTerm:
		return 3, true
	default:
		return -1, false
	}
}

func (c *StateConverter) VRRPStateToInt(state string) (int, bool) {
	switch state {
	case VRRPStateInit:
		return 0, true
	case VRRPStateBackup:
		return 1, true
	case VRRPStateMaster:
		return 2, true
	case VRRPStateFault:
		return 3, true
	default:
		return -1, false
	}
}

// JSON解析器
type JSONParser struct{}

func (p *JSONParser) Parse(r io.Reader) ([]VRRP, error) {
	var stats []VRRP
	if err := json.NewDecoder(r).Decode(&stats); err != nil {
		return nil, err
	}
	return stats, nil
}

// 文本解析器
type TextParser struct {
	converter *StateConverter
}

func isKeyArray(key string) bool {
	supportedKeys := []string{"Virtual IP"}
	for _, supportedKey := range supportedKeys {
		if supportedKey == key {
			return true
		}
	}

	logrus.WithField("Key", key).Debug("Unsupported array key")

	return false
}
func ParseVRRPData(i io.Reader) (map[string]*VRRPData, error) {
	data := make(map[string]*VRRPData)

	sep := "VRRP Instance"
	prop := "="
	arrayProp := ":"

	var instance, key, val string

	scanner := bufio.NewScanner(bufio.NewReader(i))

	for scanner.Scan() {
		l := scanner.Text()

		switch {
		case strings.HasPrefix(l, " "+sep) && strings.Contains(l, prop):
			s := strings.Split(strings.TrimSpace(l), prop)
			instance = strings.TrimSpace(s[1])
			// logrus.Infof("instance %s", instance)
			data[instance] = &VRRPData{IName: instance}
		case strings.HasPrefix(l, "   ") && instance != "":
			if strings.HasPrefix(l, "     ") {
				val = strings.TrimSpace(l)
			} else {
				var args []string
				switch {
				case strings.Contains(l, prop):
					args = strings.Split(strings.TrimSpace(l), prop)
				case strings.Contains(l, arrayProp):
					args = strings.Split(strings.TrimSpace(l), arrayProp)
				default:
					continue
				}

				key = strings.TrimSpace(args[0])
				if isKeyArray(key) {
					continue
				}
				val = strings.TrimSpace(args[1])
			}

			if (strings.HasPrefix(key, "Virtual IP (") || key == "Virtual IP") && val != "" {
				data[instance].addVIP(val)
			}

			if (strings.HasPrefix(key, "Virtual IP Excluded")) && val != "" {
				data[instance].addExcludedVIP(val)
			}

			// if val != "" {
			// 	logrus.Infof("set state %s", val)
			// }

			switch key {
			case "State":
				if err := data[instance].setState(val); err != nil {
					return data, err
				}
			case "Wantstate":
				if err := data[instance].setWantState(val); err != nil {
					return data, err
				}
			case "Interface", "Listening device":
				data[instance].Intf = val
			case "Gratuitous ARP delay":
				if err := data[instance].setGArpDelay(val); err != nil {
					return data, err
				}
			case "Virtual Router ID":
				if err := data[instance].setVRID(val); err != nil {
					return data, err
				}
			}

			// logrus.Infof("instance state %d", data[instance].State)
		case strings.HasPrefix(l, " VRRP Version") || strings.HasPrefix(l, " VRRP Script"):
			// Seen in version <= 1.3.5
			continue
		default:
			instance = ""
		}
	}

	return data, nil
}

func (p *TextParser) parseInstance(line string) string {
	if strings.HasPrefix(line, "VRRP Instance = ") {
		return strings.TrimPrefix(line, "VRRP Instance = ")
	}
	return ""
}

func (p *TextParser) parseKeyValue(line string) (string, string) {
	if !strings.Contains(line, "=") && !strings.Contains(line, ":") {
		return "", ""
	}

	sep := "="
	if strings.Contains(line, ":") {
		sep = ":"
	}

	parts := strings.SplitN(line, sep, 2)
	if len(parts) != 2 {
		return "", ""
	}

	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// func (p *TextParser) processVRRPData(data *VRRPData, key, val string) error {
// 	switch {
// 	case strings.HasPrefix(key, "Virtual IP"):
// 		data.addVIP(val)
// 	case strings.HasPrefix(key, "Virtual IP Excluded"):
// 		data.addExcludedVIP(val)
// 	case key == "State":
// 		return data.setState(val)
// 	case key == "Wantstate":
// 		return data.setWantState(val)
// 	case key == "Interface" || key == "Listening device":
// 		data.Intf = val
// 	case key == "Gratuitous ARP delay":
// 		return data.setGArpDelay(val)
// 	case key == "Virtual Router ID":
// 		return data.setVRID(val)
// 	}
// 	return nil
// }

func (p *TextParser) ParseVRRPScript(r io.Reader) []VRRPScript {
	var scripts []VRRPScript
	scanner := bufio.NewScanner(r)
	var currentScript VRRPScript

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "VRRP Script = ") {
			if currentScript.Name != "" {
				scripts = append(scripts, currentScript)
			}
			currentScript = VRRPScript{Name: strings.TrimPrefix(line, "VRRP Script = ")}
			continue
		}

		if currentScript.Name == "" {
			continue
		}

		key, val := p.parseKeyValue(line)
		switch key {
		case "Status":
			currentScript.Status = val
		case "State":
			currentScript.State = val
		}
	}

	if currentScript.Name != "" {
		scripts = append(scripts, currentScript)
	}

	return scripts
}

func ParseStats(i io.Reader) (map[string]*VRRPStats, error) {
	stats := make(map[string]*VRRPStats)

	sep := "VRRP Instance"
	prop := ":"

	scanner := bufio.NewScanner(bufio.NewReader(i))

	var instance, section string

	for scanner.Scan() {
		l := scanner.Text()

		switch {
		case strings.HasPrefix(l, sep) && strings.Contains(l, prop):
			sp := strings.Split(strings.TrimSpace(l), prop)
			// logrus.Infof("stats sp %s", strings.Split(sp[1], " ")[1])
			instance = strings.TrimSpace(strings.Split(sp[1], " ")[1])
			// logrus.Infof("stats instance %s", instance)
			stats[instance] = &VRRPStats{}
		case strings.HasPrefix(l, "  ") && strings.HasSuffix(l, prop):
			sp := strings.Split(strings.TrimSpace(l), prop)
			section = strings.TrimSpace(sp[0])
		case strings.HasPrefix(l, "    ") && section != "":
			sp := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(sp[0])
			val := strings.TrimSpace(sp[1])

			value, err := strconv.Atoi(val)
			if err != nil {
				logrus.WithFields(logrus.Fields{"key": key, "val": val}).WithError(err).Error("Unknown metric value from keepalived.stats")

				return stats, err
			}

			switch section {
			case "Advertisements":
				switch key {
				case "Received":
					stats[instance].AdvertRcvd = value
				case "Sent":
					stats[instance].AdvertSent = value
				}
			case "Packet Errors":
				switch key {
				case "Length":
					stats[instance].PacketLenErr = value
				case "TTL":
					stats[instance].IPTTLErr = value
				case "Invalid Type":
					stats[instance].InvalidTypeRcvd = value
				case "Advertisement Interval":
					stats[instance].AdvertIntervalErr = value
				case "Address List":
					stats[instance].AddrListErr = value
				}
			case "Authentication Errors":
				switch key {
				case "Invalid Type":
					stats[instance].InvalidAuthType = value
				case "Type Mismatch":
					stats[instance].AuthTypeMismatch = value
				case "Failure":
					stats[instance].AuthFailure = value
				}
			case "Priority Zero":
				switch key {
				case "Received":
					stats[instance].PRIZeroRcvd = value
				case "Sent":
					stats[instance].PRIZeroSent = value
				}
			}
		case strings.HasPrefix(l, "  ") && !strings.HasSuffix(l, prop) && !strings.HasPrefix(l, "    "):
			sp := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(sp[0])
			val := strings.TrimSpace(sp[1])
			section = ""

			value, err := strconv.Atoi(val)
			if err != nil {
				logrus.WithFields(logrus.Fields{"key": key, "val": val}).WithError(err).Error("Unknown metric value from keepalived.stats")

				return stats, err
			}

			switch key {
			case "Became master":
				stats[instance].BecomeMaster = value
			case "Released master":
				stats[instance].ReleaseMaster = value
			}
		}
	}

	return stats, nil
}

// func (p *TextParser) ParseStats(r io.Reader) (map[string]*VRRPStats, error) {
// 	stats := make(map[string]*VRRPStats)
// 	scanner := bufio.NewScanner(r)
// 	var currentInstance, currentSection string

// 	for scanner.Scan() {
// 		line := strings.TrimSpace(scanner.Text())

// 		if instance := p.parseStatsInstance(line); instance != "" {
// 			currentInstance = instance
// 			stats[currentInstance] = &VRRPStats{}
// 			continue
// 		}

// 		if section := p.parseStatsSection(line); section != "" {
// 			currentSection = section
// 			continue
// 		}

// 		if currentInstance == "" {
// 			continue
// 		}

// 		key, val := p.parseKeyValue(line)
// 		if key == "" {
// 			continue
// 		}

// 		if err := p.processStats(stats[currentInstance], currentSection, key, val); err != nil {
// 			return stats, err
// 		}
// 	}

// 	return stats, nil
// }

func (p *TextParser) parseStatsInstance(line string) string {
	if strings.HasPrefix(line, "VRRP Instance: ") {
		return strings.TrimPrefix(line, "VRRP Instance: ")
	}
	return ""
}

func (p *TextParser) parseStatsSection(line string) string {
	if strings.HasSuffix(line, ":") {
		return strings.TrimSuffix(line, ":")
	}
	return ""
}

func (p *TextParser) processStats(stats *VRRPStats, section, key, val string) error {
	value, err := strconv.Atoi(val)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"key": key,
			"val": val,
		}).Error("Invalid metric value")
		return err
	}

	switch section {
	case "Advertisements":
		switch key {
		case "Received":
			stats.AdvertRcvd = value
		case "Sent":
			stats.AdvertSent = value
		}
	case "Packet Errors":
		// 处理包错误统计...
		switch key {
		case "Length":
			stats.PacketLenErr = value
		case "TTL":
			stats.IPTTLErr = value
		case "Invalid Type":
			stats.InvalidTypeRcvd = value
		case "Advertisement Interval":
			stats.AdvertIntervalErr = value
		case "Address List":
			stats.AddrListErr = value
		}
	case "Authentication Errors":
		// 处理认证错误统计...
		switch key {
		case "Invalid Type":
			stats.InvalidAuthType = value
		case "Type Mismatch":
			stats.AuthTypeMismatch = value
		case "Failure":
			stats.AuthFailure = value
		}
	case "Priority Zero":
		// 处理优先级统计...
		switch key {
		case "Received":
			stats.PRIZeroRcvd = value
		case "Sent":
			stats.PRIZeroSent = value
		}
	default:
		switch key {
		case "Became master":
			stats.BecomeMaster = value
		case "Released master":
			stats.ReleaseMaster = value
		}
	}
	return nil
}

func ParseVIP(vip string) (string, string, bool) {
	parts := strings.Fields(vip)
	if len(parts) < 3 {
		logrus.WithField("VIP", vip).Error("Invalid VIP format")
		return "", "", false
	}
	return parts[0], parts[2], true
}
