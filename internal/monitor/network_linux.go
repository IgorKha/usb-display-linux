package monitor

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
)

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
