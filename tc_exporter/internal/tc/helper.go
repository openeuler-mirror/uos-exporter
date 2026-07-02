package tc

import (
	"errors"
	"github.com/jsimonetti/rtnetlink"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	netnsDir = "/var/run/netns"
)

// GetNetNameSpaceList returns a list of all network namespaces in the system.
// default netns will not be included.
func GetNetNameSpaceList() ([]string, error) {
	files, err := os.ReadDir(netnsDir)
	if err != nil {
		logrus.Debugf("get netns list failed: %v", err)
		if errors.Is(err, os.ErrNotExist) {
			return []string{"default"}, nil
		}
		return nil, err
	}
	logrus.Debug("get netns list: %v", files)
	var names []string
	for _, file := range files {
		names = append(names, file.Name())
	}
	// 添加default netns
	logrus.Debug("add default netns")
	names = append(names, "default")
	return names, nil
}

func GetInterfaceInNetNS(ns string) ([]rtnetlink.LinkMessage, error) {
	conn, err := GetNetlinkConn(ns)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	links, err := conn.Link.List()
	if err != nil {
		return nil, err
	}
	// 忽略回环网卡
	selected := make([]rtnetlink.LinkMessage,
		len(links)-1)
	for i, link := range links {
		if i == 0 {
			continue
		}
		selected[i-1] = link
	}
	return selected, nil
}
