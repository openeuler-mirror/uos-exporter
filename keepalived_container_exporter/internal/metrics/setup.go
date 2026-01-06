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
	v.GArpDelay = delayInt
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"delay": delay,
			"error": err,
		}).Error("Gratuitous ARP delay conversion failed")
		return fmt.Errorf("invalid GArpDelay value: %w", err)
	}
	// v.GArpDelay = delayInt
	return nil
}

func (v *VRRPData) setVRID(vrid string) error {
	vridInt, err := strconv.Atoi(vrid)
	v.VRID = vridInt
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"vrid":  vrid,
			"error": err,
		}).Error("VRID conversion failed")
		return fmt.Errorf("invalid VRID value: %w", err)
	}
	// v.VRID = vridInt
	return nil
}

// VIP管理方法
func (v *VRRPData) addVIP(vip string) {
	cleanVIP := strings.TrimSpace(vip)
	if cleanVIP != "" {
		v.VIPs = append(v.VIPs, cleanVIP)
	}
}

func (v *VRRPData) addExcludedVIP(vip string) {
	cleanVIP := strings.TrimSpace(vip)
	if cleanVIP != "" {
		v.ExcludedVIPs = append(v.ExcludedVIPs, cleanVIP)
	}
}
