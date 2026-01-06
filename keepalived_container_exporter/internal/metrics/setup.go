package metrics

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// 状态设置方法
func (v *VRRPData) setState(state string) error {
	sc := StateConverter{}
	stateCode, valid := sc.VRRPStateToInt(state)
	v.State = stateCode
	if !valid {
		// err := fmt.Errorf("invalid VRRP state: %s for instance: %s", state, v.IName)
		logrus.Error("VRRP state validation failed")
		return fmt.Errorf("invalid VRRP state: %s for instance: %s", state, v.IName)
	}
	// logrus.Infof("set state %d", stateCode)
	// v.State = stateCode
	return nil
}

func (v *VRRPData) setWantState(state string) error {
	sc := StateConverter{}
	stateCode, valid := sc.VRRPStateToInt(state)
	v.WantState = stateCode
	if !valid {
		err := fmt.Errorf("invalid wantstate: %s", state)
		logrus.WithError(err).Error("Wantstate validation failed")
		return err
	}
	// v.WantState = stateCode
	return nil
}

// 数值转换方法
func (v *VRRPData) setGArpDelay(delay string) error {
	delayInt, err := strconv.Atoi(delay)

// TODO: implement functions
