package monitor

import (
	"net/netip"
	"strconv"
	"strings"
)

type AddressSource uint8

const (
	SourceUnknown AddressSource = iota
	SourceDHCP
	SourceStatic
	SourceLikelyDHCP
	SourceLikelyStatic
)

func (source AddressSource) String() string {
	switch source {
	case SourceDHCP:
		return "DHCP"
	case SourceStatic:
		return "STATIC"
	case SourceLikelyDHCP:
		return "DHCP?"
	case SourceLikelyStatic:
		return "STATIC?"
	default:
		return "?"
	}
}

func inferAddressSource(address netip.Addr, permanent bool, preferred, valid uint32) AddressSource {
	if !address.Is4() || address.IsLinkLocalUnicast() || address.IsLoopback() || valid == 0 {
		return SourceUnknown
	}
	const forever = ^uint32(0)
	if valid != forever || preferred != forever {
		return SourceLikelyDHCP
	}
	if permanent {
		return SourceLikelyStatic
	}
	return SourceUnknown
}

func parseIPConfigSummary(summary string) map[netip.Addr]AddressSource {
	result := map[netip.Addr]AddressSource{}
	addresses := []netip.Addr{}
	method := SourceUnknown
	published, ipv4, inAddresses := false, false, false
	flush := func() {
		if published && method != SourceUnknown {
			for _, address := range addresses {
				result[address] = method
			}
		}
		addresses = []netip.Addr{}
		method, published, inAddresses = SourceUnknown, false, false
	}
	for _, line := range strings.Split(summary, "\n") {
		if line == "  IPv4 : <array> {" {
			ipv4 = true
			continue
		}
		if !ipv4 {
			continue
		}
		if line == "  }" {
			flush()
			break
		}
		if strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "     ") {
			key, value, found := strings.Cut(strings.TrimSpace(line), " : ")
			if _, err := strconv.Atoi(key); found && err == nil && value == "<dictionary> {" {
				flush()
			}
		}
		if line == "      Addresses : <array> {" {
			inAddresses = true
			continue
		}
		if inAddresses {
			if line == "      }" {
				inAddresses = false
				continue
			}
			_, value, found := strings.Cut(strings.TrimSpace(line), " : ")
			if address, err := netip.ParseAddr(value); found && err == nil && address.Is4() {
				addresses = append(addresses, address)
			}
			continue
		}
		if strings.HasPrefix(line, "      ConfigMethod : ") {
			switch strings.TrimPrefix(line, "      ConfigMethod : ") {
			case "DHCP":
				method = SourceDHCP
			case "Manual", "INFORM":
				method = SourceStatic
			default:
				method = SourceUnknown
			}
		}
		if line == "      IsPublished : TRUE" {
			published = true
		}
	}
	return result
}
