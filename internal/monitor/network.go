package monitor

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strings"
	"time"
)

type Link uint8

const (
	LinkUnknown Link = iota
	LinkNo
	LinkYes
)

func (link Link) String() string {
	switch link {
	case LinkNo:
		return "NO"
	case LinkYes:
		return "YES"
	default:
		return "?"
	}
}

type Interface struct {
	Name         string
	HardwareAddr string
	Up           bool
	Link         Link
	Addresses    []netip.Addr
	AddressError bool
}

func CollectNetwork(ctx context.Context) ([]Interface, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	links := collectLinks(ctx, interfaces)
	return collectInterfaces(interfaces, func(iface net.Interface) ([]net.Addr, error) { return iface.Addrs() }, links), nil
}

func collectInterfaces(interfaces []net.Interface, addresses func(net.Interface) ([]net.Addr, error), links map[string]Link) []Interface {
	result := make([]Interface, 0, len(interfaces))
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		item := Interface{
			Name:         iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			Up:           iface.Flags&net.FlagUp != 0,
			Link:         links[iface.Name],
			Addresses:    []netip.Addr{},
		}
		values, err := addresses(iface)
		item.AddressError = err != nil
		for _, value := range values {
			prefix, parseErr := netip.ParsePrefix(value.String())
			if parseErr != nil {
				continue
			}
			address := prefix.Addr().Unmap()
			if address.Is4() && !address.IsLoopback() && !address.IsUnspecified() {
				item.Addresses = append(item.Addresses, address)
			}
		}
		slices.SortFunc(item.Addresses, func(first, second netip.Addr) int { return first.Compare(second) })
		item.Addresses = slices.Compact(item.Addresses)
		result = append(result, item)
	}
	slices.SortFunc(result, func(first, second Interface) int { return strings.Compare(first.Name, second.Name) })
	return result
}

func parseMediaLinks(text string, physicalPorts map[string]bool) map[string]Link {
	result := map[string]Link{}
	name, state := "", LinkUnknown
	flush := func() {
		if name != "" && physicalPorts[name] {
			result[name] = state
		}
	}
	for _, line := range strings.Split(text, "\n") {
		if line != "" && line[0] != '\t' && line[0] != ' ' {
			flush()
			name, _, _ = strings.Cut(line, ":")
			state = LinkUnknown
			continue
		}
		field := strings.TrimSpace(line)
		switch field {
		case "status: active":
			state = LinkYes
		case "status: inactive":
			state = LinkNo
		}
	}
	flush()
	return result
}

func parseHardwarePorts(text string) map[string]bool {
	ports := map[string]bool{}
	label := ""
	for _, line := range strings.Split(text, "\n") {
		field := strings.TrimSpace(line)
		if value, found := strings.CutPrefix(field, "Hardware Port:"); found {
			label = strings.ToLower(strings.TrimSpace(value))
		}
		if value, found := strings.CutPrefix(field, "Device:"); found {
			virtual := strings.Contains(label, "bridge") || strings.Contains(label, "vlan") || strings.Contains(label, "bond")
			if label != "" && !virtual {
				ports[strings.TrimSpace(value)] = true
			}
		}
	}
	return ports
}
