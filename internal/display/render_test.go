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
		{Name: "en0", Up: true, Link: monitor.LinkYes, Addresses: []netip.Addr{netip.MustParseAddr("169.254.1.2"), netip.MustParseAddr("255.255.255.255")}},
		{Name: "vpn0"},
		{Name: "gone0", AddressError: true},
	})
	if len(pages) != 5 || pages[1].Address != "169.254.1.2" || pages[2].Address != "255.255.255.255" || pages[3].Address != "NO IPv4" || pages[4].Address != "DATA ERROR" {
		t.Fatalf("pages = %+v", pages)
	}
	if pages[3].Status != "DOWN   LINK ?" {
		t.Fatalf("status = %q", pages[3].Status)
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
	joined := strings.Join(wrap(longName, renderer.normal), "")
	if joined != longName {
		t.Fatal("long interface name lost characters")
	}
}
