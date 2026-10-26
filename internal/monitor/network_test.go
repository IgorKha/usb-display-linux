package monitor

import (
	"errors"
	"net"
	"testing"
)

func TestCollectInterfaces(t *testing.T) {
	interfaces := []net.Interface{
		{Name: "lo0", Flags: net.FlagLoopback | net.FlagUp},
		{Name: "vpn0"},
		{Name: "en0", Flags: net.FlagUp},
		{Name: "gone0"},
	}
	addresses := func(iface net.Interface) ([]net.Addr, error) {
		if iface.Name == "gone0" {
			return nil, errors.New("interface disappeared")
		}
		if iface.Name != "en0" {
			return []net.Addr{}, nil
		}
		values := []net.Addr{}
		for _, text := range []string{"169.254.2.3/16", "10.0.0.10/24", "10.0.0.2/24", "10.0.0.2/24", "127.0.0.1/8", "fe80::1/64"} {
			address, prefix, err := net.ParseCIDR(text)
			if err != nil {
				t.Fatal(err)
			}
			prefix.IP = address
			values = append(values, prefix)
		}
		return values, nil
	}
	got := collectInterfaces(interfaces, addresses, map[string]Link{"en0": LinkYes})
	if len(got) != 3 || got[0].Name != "en0" || !got[0].Up || got[0].Link != LinkYes {
		t.Fatalf("interfaces = %+v", got)
	}
	want := []string{"10.0.0.2", "10.0.0.10", "169.254.2.3"}
	for index, address := range got[0].Addresses {
		if index >= len(want) || address.String() != want[index] {
			t.Fatalf("addresses = %v", got[0].Addresses)
		}
	}
	if len(got[0].Addresses) != len(want) || !got[1].AddressError || got[2].Up || got[2].Link != LinkUnknown {
		t.Fatalf("interfaces = %+v", got)
	}
}

func TestParseMediaLinks(t *testing.T) {
	text := "en0: flags=UP\n\tmedia: autoselect (1000baseT <full-duplex>)\n\tstatus: active\n" +
		"en1: flags=UP\n\tmedia: IEEE802.11 autoselect\n\tstatus: inactive\n" +
		"bridge0: flags=UP\n\tmedia: autoselect\n\tstatus: active\n" +
		"utun0: flags=UP\n"
	got := parseMediaLinks(text, map[string]bool{"en0": true, "en1": true})
	if got["en0"] != LinkYes || got["en1"] != LinkNo || got["bridge0"] != LinkUnknown || got["utun0"] != LinkUnknown {
		t.Fatalf("links = %v", got)
	}
}

func TestHardwarePorts(t *testing.T) {
	hardware := "Hardware Port: Wi-Fi\nDevice: en0\n\nHardware Port: Thunderbolt Bridge\nDevice: bridge0\n"
	links := parseMediaLinks("en0: flags=UP\n\tmedia: autoselect\n\tstatus: active\nfeth1: flags=UP\n\tstatus: active\n", parseHardwarePorts(hardware))
	if links["en0"] != LinkYes || links["feth1"] != LinkUnknown || links["bridge0"] != LinkUnknown {
		t.Fatalf("links = %v", links)
	}
}
