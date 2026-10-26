package display

import (
	"bytes"
	"net/netip"
	"strings"
	"testing"

	"github.com/IgorKha/usb-display-linux/internal/monitor"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	renderer, err := NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := renderer.Close(); err != nil {
			t.Error(err)
		}
	})
	return renderer
}

func TestPages(t *testing.T) {
	renderer := testRenderer(t)
	pages := renderer.Pages("macOS 15.7", []monitor.Interface{
		{Name: "en0", HardwareAddr: "02:1a:2b:3c:4d:5e", Up: true, Link: monitor.LinkYes, Addresses: []netip.Addr{netip.MustParseAddr("169.254.1.2"), netip.MustParseAddr("255.255.255.255")}},
		{Name: "vpn0"},
		{Name: "gone0", AddressError: true},
	})
	if len(pages) != 5 || pages[1].Address != "169.254.1.2" || pages[2].Address != "255.255.255.255" || pages[3].Address != "NO IPv4" || pages[4].Address != "DATA ERROR" {
		t.Fatalf("pages = %+v", pages)
	}
	if pages[3].Status != "DOWN   LINK ?" {
		t.Fatalf("status = %q", pages[3].Status)
	}
	if pages[1].HardwareAddr != "02:1a:2b:3c:4d:5e" || pages[2].HardwareAddr != pages[1].HardwareAddr || pages[3].HardwareAddr != "MAC ?" {
		t.Fatalf("hardware address pages = %+v", pages)
	}
	frame := renderer.Render(pages[1], 1, len(pages))
	macPixels := 0
	for row := 69; row < Height; row++ {
		for column := 4; column < Width-4; column++ {
			pixel := frame.RGBAAt(column, row)
			if pixel.R > 50 && pixel.G >= pixel.R && pixel.G-pixel.R < 8 && pixel.B >= pixel.R && pixel.B-pixel.R < 8 {
				macPixels++
			}
		}
	}
	if macPixels == 0 {
		t.Fatal("MAC is missing from rendered footer")
	}
	if len(renderer.Pages("Linux", nil)) != 2 {
		t.Fatal("missing empty network page")
	}
}

func TestRenderBounds(t *testing.T) {
	renderer := testRenderer(t)
	longName := strings.Repeat("very-long-interface-", 4)
	pages := renderer.Pages(strings.Repeat("Linux release ", 12), []monitor.Interface{{Name: longName}})
	if len(pages) < 4 {
		t.Fatal("long content was not paginated")
	}
	for position, page := range pages {
		for _, line := range page.Title {
			if font.MeasureString(renderer.normal, line) > fixed.I(Width-8) {
				t.Fatalf("title overflows: %q", line)
			}
		}
		for _, line := range page.Lines {
			if font.MeasureString(renderer.medium, line) > fixed.I(Width-8) {
				t.Fatalf("OS line overflows: %q", line)
			}
		}
		frame := renderer.Render(page, position, len(pages))
		if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
			t.Fatal("wrong bounds")
		}
		if bytes.Count(frame.Pix, []byte{0}) == len(frame.Pix) {
			t.Fatal("blank image")
		}
	}
	if font.MeasureString(renderer.large, "255.255.255.255") > fixed.I(Width-8) {
		t.Fatal("IPv4 overflows")
	}
	macBounds, _ := font.BoundString(renderer.medium, "02:1a:2b:3c:4d:5e")
	if font.MeasureString(renderer.medium, "02:1a:2b:3c:4d:5e") > fixed.I(Width-8) {
		t.Fatal("MAC overflows footer")
	}
	if macBounds.Max.Y+fixed.I(78) > fixed.I(Height) {
		t.Fatal("MAC overflows bottom edge")
	}
	addressBounds, _ := font.BoundString(renderer.large, "255.255.255.255")
	if addressBounds.Max.Y+fixed.I(60) >= macBounds.Min.Y+fixed.I(78) {
		t.Fatal("MAC overlaps IPv4")
	}
	statusWidth := font.MeasureString(renderer.normal, "IPv4 STATIC?")
	counterWidth := font.MeasureString(renderer.small, "999/999")
	if statusWidth+counterWidth > fixed.I(Width-12) {
		t.Fatal("address source overlaps page counter")
	}
	joined := strings.Join(wrap(longName, renderer.normal), "")
	if joined != longName {
		t.Fatal("long interface name lost characters")
	}
}

func TestAddressSourcePages(t *testing.T) {
	renderer := testRenderer(t)
	dhcp := netip.MustParseAddr("192.0.2.1")
	static := netip.MustParseAddr("192.0.2.2")
	unknown := netip.MustParseAddr("169.254.1.2")
	name := strings.Repeat("long-interface-name", 3)
	pages := renderer.Pages("Linux", []monitor.Interface{{
		Name:      name,
		Addresses: []netip.Addr{dhcp, static, unknown},
		Sources: map[netip.Addr]monitor.AddressSource{
			dhcp: monitor.SourceDHCP, static: monitor.SourceLikelyStatic,
		},
	}})
	want := map[string]string{dhcp.String(): "DHCP", static.String(): "STATIC?", unknown.String(): "?"}
	title := ""
	for _, page := range pages[1:] {
		if len(page.Title) != 1 || page.Source != want[page.Address] {
			t.Fatalf("source page = %+v", page)
		}
		if page.Address == dhcp.String() {
			title += page.Title[0]
		}
	}
	if title != name {
		t.Fatal("interface name lost during pagination")
	}
	frame := renderer.Render(pages[1], 1, len(pages))
	without := pages[1]
	without.Source = ""
	if bytes.Equal(frame.Pix, renderer.Render(without, 1, len(pages)).Pix) {
		t.Fatal("address source was not rendered")
	}
	for _, rows := range []struct {
		name, upper, lower           string
		upperBaseline, lowerBaseline int
	}{
		{name: "title and source", upper: "en0", lower: "IPv4 STATIC?", upperBaseline: 13, lowerBaseline: 27},
		{name: "source and status", upper: "IPv4 STATIC?", lower: "DOWN   LINK YES", upperBaseline: 27, lowerBaseline: 41},
	} {
		t.Run(rows.name, func(t *testing.T) {
			upper, _ := font.BoundString(renderer.normal, rows.upper)
			lower, _ := font.BoundString(renderer.normal, rows.lower)
			if upper.Max.Y+fixed.I(rows.upperBaseline) >= lower.Min.Y+fixed.I(rows.lowerBaseline) {
				t.Fatal("rows overlap")
			}
		})
	}
}

func TestHardwareAddressContinuation(t *testing.T) {
	renderer := testRenderer(t)
	address := strings.Repeat("ab:", 19) + "ab"
	pages := renderer.Pages("Linux", []monitor.Interface{{Name: "ib0", HardwareAddr: address}})
	parts := []string{}
	keys := map[string]bool{}
	for _, page := range pages[1:] {
		parts = append(parts, page.HardwareAddr)
		if keys[page.Key] {
			t.Fatalf("duplicate page key: %q", page.Key)
		}
		keys[page.Key] = true
		if font.MeasureString(renderer.medium, page.HardwareAddr) > fixed.I(Width-8) {
			t.Fatalf("hardware address overflows: %q", page.HardwareAddr)
		}
	}
	if len(parts) < 2 || strings.Join(parts, "") != address {
		t.Fatalf("hardware address lost: %v", parts)
	}
}
