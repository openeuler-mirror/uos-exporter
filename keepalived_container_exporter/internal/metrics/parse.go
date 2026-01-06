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


// TODO: implement functions
