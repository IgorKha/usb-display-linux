package monitor

import (
	"net/netip"
	"testing"
)

func TestInferAddressSource(t *testing.T) {
	const forever = ^uint32(0)
	for _, test := range []struct {
		name      string
		address   string
		permanent bool
		preferred uint32
		valid     uint32
		want      AddressSource
	}{
		{name: "finite lease", address: "192.0.2.1", preferred: 100, valid: 100, want: SourceLikelyDHCP},
		{name: "permanent", address: "192.0.2.1", permanent: true, preferred: forever, valid: forever, want: SourceLikelyStatic},
		{name: "unmarked infinite", address: "192.0.2.1", preferred: forever, valid: forever},
		{name: "link local", address: "169.254.1.2", preferred: 100, valid: 100},
		{name: "missing lifetime", address: "192.0.2.1", permanent: true},
		{name: "expired", address: "192.0.2.1", preferred: 0, valid: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := inferAddressSource(netip.MustParseAddr(test.address), test.permanent, test.preferred, test.valid)
			if got != test.want {
				t.Fatalf("source = %s, want %s", got, test.want)
			}
		})
	}
}

func TestParseIPConfigSummary(t *testing.T) {
	for _, test := range []struct {
		name, method string
		published    bool
		want         AddressSource
	}{
		{name: "dhcp", method: "DHCP", published: true, want: SourceDHCP},
		{name: "manual", method: "Manual", published: true, want: SourceStatic},
		{name: "inform is static", method: "INFORM", published: true, want: SourceStatic},
		{name: "unpublished", method: "DHCP"},
		{name: "link local", method: "LinkLocal", published: true},
		{name: "unknown method", method: "Other", published: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			published := "FALSE"
			if test.published {
				published = "TRUE"
			}
			summary := "<dictionary> {\n  IPv4 : <array> {\n    0 : <dictionary> {\n" +
				"      Addresses : <array> {\n        0 : 192.0.2.1\n        1 : 192.0.2.2\n      }\n" +
				"      ConfigMethod : " + test.method + "\n      IsPublished : " + published + "\n    }\n" +
				"    1 : <dictionary> {\n      ConfigMethod : LinkLocal\n      IsPublished : TRUE\n    }\n  }\n" +
				"  IPv6 : <array> {\n      ConfigMethod : Manual\n  }\n}\n"
			got := parseIPConfigSummary(summary)
			for _, text := range []string{"192.0.2.1", "192.0.2.2"} {
				if got[netip.MustParseAddr(text)] != test.want {
					t.Fatalf("sources = %v, want %s", got, test.want)
				}
			}
		})
	}
	if len(parseIPConfigSummary("unsupported or malformed output")) != 0 {
		t.Fatal("invented an origin from unsupported output")
	}
}
