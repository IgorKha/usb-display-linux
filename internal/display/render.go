package display

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"unicode"

	"github.com/IgorKha/usb-display-linux/internal/monitor"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type Page struct {
	Key          string
	Title        []string
	Lines        []string
	Status       string
	Address      string
	HardwareAddr string
	Up           bool
}

type Renderer struct {
	small  font.Face
	normal font.Face
	medium font.Face
	large  font.Face
}

func NewRenderer() (*Renderer, error) {
	parsed, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}
	renderer := &Renderer{}
	for _, item := range []struct {
		size   float64
		target *font.Face
	}{
		{size: 9, target: &renderer.small},
		{size: 12, target: &renderer.normal},
		{size: 14, target: &renderer.medium},
		{size: 16, target: &renderer.large},
	} {
		face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: item.size, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			return nil, errors.Join(err, renderer.Close())
		}
		*item.target = face
	}
	return renderer, nil
}

func (renderer *Renderer) Close() error {
	errorsFound := []error{}
	for _, face := range []font.Face{renderer.small, renderer.normal, renderer.medium, renderer.large} {
		if face != nil {
			errorsFound = append(errorsFound, face.Close())
		}
	}
	return errors.Join(errorsFound...)
}

func (renderer *Renderer) Pages(osName string, interfaces []monitor.Interface) []Page {
	pages := []Page{}
	lines := wrap(osName, renderer.medium)
	for start := 0; start < len(lines); start += 3 {
		pages = append(pages, Page{Key: fmt.Sprintf("os:%d", start), Title: []string{"OPERATING SYSTEM"}, Lines: lines[start:min(start+3, len(lines))]})
	}
	for _, iface := range interfaces {
		addresses := []string{}
		for _, address := range iface.Addresses {
			addresses = append(addresses, address.String())
		}
		if len(addresses) == 0 {
			addresses = append(addresses, "NO IPv4")
		}
		if iface.AddressError {
			addresses = []string{"DATA ERROR"}
		}
		state := "DOWN"
		if iface.Up {
			state = "UP"
		}
		titles := wrap(iface.Name, renderer.normal)
		hardwareAddr := iface.HardwareAddr
		if hardwareAddr == "" {
			hardwareAddr = "MAC ?"
		}
		hardwareLines := wrap(hardwareAddr, renderer.medium)
		for _, address := range addresses {
			for start := 0; start < len(titles); start += 2 {
				for part, hardwareLine := range hardwareLines {
					key := fmt.Sprintf("if:%s:%s:%d", iface.Name, address, start)
					if part > 0 {
						key += fmt.Sprintf(":mac:%d", part)
					}
					pages = append(pages, Page{
						Key:          key,
						Title:        titles[start:min(start+2, len(titles))],
						Status:       state + "   LINK " + iface.Link.String(),
						Address:      address,
						HardwareAddr: hardwareLine,
						Up:           iface.Up,
					})
				}
			}
		}
	}
	if len(interfaces) == 0 {
		pages = append(pages, Page{Key: "empty", Title: []string{"NETWORK"}, Lines: []string{"NO INTERFACES"}})
	}
	return pages
}

func (renderer *Renderer) Render(page Page, position, total int) *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, Width, Height))
	white := color.RGBA{R: 235, G: 239, B: 240, A: 255}
	accent := color.RGBA{R: 77, G: 208, B: 190, A: 255}
	text := func(face font.Face, baseline int, value string, ink color.RGBA) {
		drawer := font.Drawer{Dst: frame, Src: image.NewUniform(ink), Face: face, Dot: fixed.P(4, baseline)}
		drawer.DrawString(value)
	}
	for index, title := range page.Title {
		text(renderer.normal, 13+index*13, title, accent)
	}
	if page.Address != "" {
		stateColor := color.RGBA{R: 245, G: 148, B: 87, A: 255}
		if page.Up {
			stateColor = color.RGBA{R: 128, G: 220, B: 137, A: 255}
		}
		text(renderer.normal, 41, page.Status, stateColor)
		text(renderer.large, 60, page.Address, white)
	}
	for index, line := range page.Lines {
		text(renderer.medium, 34+index*16, line, white)
	}
	counter := fmt.Sprintf("%d / %d", position+1, total)
	if page.HardwareAddr != "" {
		text(renderer.medium, 78, page.HardwareAddr, white)
		counter = fmt.Sprintf("%d/%d", position+1, total)
		drawer := font.Drawer{
			Dst: frame, Src: image.NewUniform(accent), Face: renderer.small,
			Dot: fixed.P(Width-4-font.MeasureString(renderer.small, counter).Ceil(), 41),
		}
		drawer.DrawString(counter)
	} else {
		text(renderer.small, 77, counter, accent)
	}
	return frame
}

func (renderer *Renderer) TestPattern() *image.RGBA {
	frame := renderer.Render(Page{Title: []string{"WeAct 160 x 80"}, Lines: []string{"RGB TEST"}}, 0, 1)
	colors := []color.RGBA{{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255}}
	for index, ink := range colors {
		area := image.Rect(index*Width/3, 46, (index+1)*Width/3, 69)
		draw.Draw(frame, area, image.NewUniform(ink), image.Point{}, draw.Src)
	}
	return frame
}

func wrap(value string, face font.Face) []string {
	return wrapWidth(value, face, Width-8)
}

func wrapWidth(value string, face font.Face, width int) []string {
	value = strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, value)
	lines := []string{}
	line := ""
	for _, character := range value {
		candidate := line + string(character)
		if line != "" && font.MeasureString(face, candidate) > fixed.I(width) {
			lines = append(lines, line)
			line = ""
		}
		line += string(character)
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return []string{"?"}
	}
	return lines
}
