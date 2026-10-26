package monitor

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func collectAddressSources(ctx context.Context, interfaces []Interface) {
	if ctx.Err() != nil {
		return
	}
	handle, err := netlink.NewHandle(unix.NETLINK_ROUTE)
	if err != nil {
		slog.Warn("address origins unavailable", "error", err)
		return
	}
	defer handle.Close()
	timeout := 500 * time.Millisecond
	if deadline, ok := ctx.Deadline(); ok {
		timeout = min(timeout, time.Until(deadline))
	}
	if timeout <= 0 {
		return
	}
	if err := handle.SetSocketTimeout(timeout); err != nil {
		slog.Warn("address origin timeout unavailable", "error", err)
		return
	}
	addresses, err := handle.AddrList(nil, netlink.FAMILY_V4)
	if err != nil || ctx.Err() != nil {
		return
	}
	byIndex := map[int]*Interface{}
	for index := range interfaces {
		iface, err := net.InterfaceByName(interfaces[index].Name)
		if err == nil {
			byIndex[iface.Index] = &interfaces[index]
		}
	}
	for _, item := range addresses {
		iface := byIndex[item.LinkIndex]
		if iface == nil || item.IPNet == nil {
			continue
		}
		address, ok := netip.AddrFromSlice(item.IP)
		if !ok {
			continue
		}
		address = address.Unmap()
		iface.Sources[address] = inferAddressSource(
			address,
			item.Flags&unix.IFA_F_PERMANENT != 0,
			uint32(item.PreferedLft),
			uint32(item.ValidLft),
		)
	}
}

func collectLinks(ctx context.Context, interfaces []net.Interface) map[string]Link {
	result := map[string]Link{}
	for _, iface := range interfaces {
		if ctx.Err() != nil {
			break
		}
		result[iface.Name] = readCarrier("/sys/class/net", iface.Name)
	}
	return result
}

func readCarrier(root, name string) Link {
	directory := filepath.Join(root, name)
	if _, err := os.Stat(filepath.Join(directory, "device")); err != nil {
		return LinkUnknown
	}
	data, err := os.ReadFile(filepath.Join(directory, "carrier"))
	if err != nil {
		return LinkUnknown
	}
	switch strings.TrimSpace(string(data)) {
	case "1":
		return LinkYes
	case "0":
		return LinkNo
	default:
		return LinkUnknown
	}
}
