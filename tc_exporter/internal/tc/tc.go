package tc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/netlink"
	"golang.org/x/sys/unix"
)

// HandleStr returns the major and minor parts of a TC handle
func HandleStr(handle uint32) (uint32, uint32) {
	return (handle & 0xffff0000) >> 16, handle & 0x0000ffff
}

// FmtHandleStr formats a TC handle as "major:minor"
func FmtHandleStr(handle uint32) string {
	major, minor := HandleStr(handle)
	return fmt.Sprintf("%d:%d", major, minor)
}

// getConnHelper is a helper function to get a connection in a network namespace
func getConnHelper(ns string, dialFunc func(*netlink.Config) (*rtnetlink.Conn, error)) (*rtnetlink.Conn, error) {
	if ns == "default" {
		return dialFunc(nil)
	}
	NetNsPath := "/var/run/netns/" + ns
	cleanNetNsPath := filepath.Clean(NetNsPath)
	if !strings.HasPrefix(cleanNetNsPath, "/var/run/netns/") {
		return nil, os.ErrPermission
	}
	f, err := os.Open(cleanNetNsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open netns: %w", err)
	}
	defer f.Close()

	return dialFunc(&netlink.Config{NetNS: int(f.Fd())})
}

// GetNetlinkConn gets a rtnetlink connection for the specified network namespace
func GetNetlinkConn(ns string) (*rtnetlink.Conn, error) {
	return getConnHelper(ns, rtnetlink.Dial)
}

func ValidateNsPath(ns string) (nsPath string, err error) {
	netNsPath := "/var/run/netns/" + ns
	cleanNetNsPath := filepath.Clean(netNsPath)
	if !strings.HasPrefix(cleanNetNsPath, "/var/run/netns/") {
		return "", os.ErrPermission
	}
	return cleanNetNsPath, nil
}

// GetTcConn gets a TC connection for the specified network namespace
func GetTcConn(ns string) (*tc.Tc, error) {
	var sock *tc.Tc
	var err error

	openFunc := func(cfg *netlink.Config) (*tc.Tc, error) {
		return tc.Open(&tc.Config{NetNS: cfg.NetNS})
	}

	if ns == "default" {
		sock, err = openFunc(&netlink.Config{})
	} else {
		netNsPath := "/var/run/netns/" + ns
		cleanNetNsPath := filepath.Clean(netNsPath)
		if !strings.HasPrefix(cleanNetNsPath, "/var/run/netns/") {
			return nil, os.ErrPermission
		}
		// cleanNsPath, err := ValidateNsPath(ns)
		// if err != nil {
		// 	return nil, err
		// }
		f, err := os.Open(cleanNetNsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open netns: %w", err)
		}
		defer f.Close()

		sock, err = openFunc(&netlink.Config{NetNS: int(f.Fd())})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open TC connection: %w", err)
	}
	return sock, nil
}

// getTcObjects is a helper function to get TC objects (qdiscs, classes or filters)
func getTcObjects(devid uint32, ns string, getFunc func(*tc.Tc) ([]tc.Object, error)) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()

	objects, err := getFunc(sock)
	if err != nil {
		return nil, err
	}

	// Filter objects by interface index
	var result []tc.Object
	for _, obj := range objects {
		if obj.Ifindex == devid {
			result = append(result, obj)
		}
	}

	return result, nil
}

// GetQdiscs fetches all qdiscs for a specified interface in the netns
func GetQdiscs(devid uint32, ns string) ([]tc.Object, error) {
	return getTcObjects(devid, ns, func(sock *tc.Tc) ([]tc.Object, error) {
		return sock.Qdisc().Get()
	})
}

// GetClasses fetches all classes for a specified interface in the netns
func GetClasses(devid uint32, ns string) ([]tc.Object, error) {
	return getTcObjects(devid, ns, func(sock *tc.Tc) ([]tc.Object, error) {
		return sock.Class().Get(&tc.Msg{
			Family:  unix.AF_UNSPEC,
			Info:    0,
			Handle:  tc.HandleRoot,
			Ifindex: devid,
		})
	})
}

// GetFilters fetches all filters for a specified interface in the netns
func GetFilters(devid uint32, ns string) ([]tc.Object, error) {
	return getTcObjects(devid, ns, func(sock *tc.Tc) ([]tc.Object, error) {
		return sock.Filter().Get(&tc.Msg{
			Family:  unix.AF_UNSPEC,
			Info:    0,
			Handle:  tc.HandleRoot,
			Ifindex: devid,
		})
	})
}
