package display

import (
	"image"
	"image/color"
	"testing"
)

func TestRotate180(t *testing.T) {
	frame := image.NewRGBA(image.Rect(3, 5, 6, 7))
	frame.SetRGBA(3, 5, color.RGBA{R: 255, A: 255})
	frame.SetRGBA(5, 6, color.RGBA{B: 255, A: 255})
	rotated := rotate180(frame)
	if rotated.Bounds() != frame.Bounds() || rotated.RGBAAt(5, 6) != frame.RGBAAt(3, 5) || rotated.RGBAAt(3, 5) != frame.RGBAAt(5, 6) {
		t.Fatal("rotation did not reverse both axes")
	}
	restored := rotate180(rotated)
	for row := 5; row < 7; row++ {
		for column := 3; column < 6; column++ {
			if restored.RGBAAt(column, row) != frame.RGBAAt(column, row) {
				t.Fatal("double rotation changed image")
			}
		}
	}
}
