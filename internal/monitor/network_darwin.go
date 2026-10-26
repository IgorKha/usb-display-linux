package monitor

import (
	"context"
	"log/slog"
	"net"
	"os/exec"
)

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
