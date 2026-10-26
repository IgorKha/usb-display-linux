package monitor

import (
	"context"
	"log/slog"
	"net"
	"os/exec"
	"time"
)

func collectAddressSources(ctx context.Context, interfaces []Interface) {
	for index := range interfaces {
		if ctx.Err() != nil {
			return
		}
		iface := &interfaces[index]
		if len(iface.Addresses) == 0 {
			continue
		}
		query, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
		data, err := exec.CommandContext(query, "/usr/sbin/ipconfig", "getsummary", iface.Name).Output()
		cancel()
		if err == nil {
			iface.Sources = parseIPConfigSummary(string(data))
		}
	}
}

func collectLinks(ctx context.Context, interfaces []net.Interface) map[string]Link {
	hardware, err := exec.CommandContext(ctx, "/usr/sbin/networksetup", "-listallhardwareports").Output()
	if err != nil {
		slog.Warn("hardware port list unavailable", "error", err)
		return map[string]Link{}
	}
	data, err := exec.CommandContext(ctx, "/sbin/ifconfig", "-a").Output()
	if err != nil {
		slog.Warn("link status unavailable", "error", err)
		return map[string]Link{}
	}
	return parseMediaLinks(string(data), parseHardwarePorts(string(hardware)))
}
