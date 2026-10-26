package display

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"testing"
)

func TestRGB565(t *testing.T) {
	frame := image.NewRGBA(image.Rect(3, 4, 6, 5))
	frame.Set(3, 4, color.RGBA{R: 255, A: 255})
	frame.Set(4, 4, color.RGBA{G: 255, A: 255})
	frame.Set(5, 4, color.RGBA{B: 255, A: 255})
	if got := RGB565(frame); !bytes.Equal(got, []byte{0, 0xf8, 0xe0, 7, 0x1f, 0}) {
		t.Fatalf("RGB565 = %x", got)
	}
}

type shortWriter struct{ bytes.Buffer }

func (writer *shortWriter) Write(data []byte) (int, error) {
	return writer.Buffer.Write(data[:min(3, len(data))])
}

func TestWriteBitmap(t *testing.T) {
	writer := &shortWriter{}
	frame := image.NewRGBA(image.Rect(0, 0, Width, Height))
	if err := WriteFrame(writer, frame); err != nil {
		t.Fatal(err)
	}
	if writer.Len() != 25610 {
		t.Fatalf("packet length = %d", writer.Len())
	}
	want := []byte{5, 0, 0, 0, 0, 159, 0, 79, 0, 10}
	if !bytes.Equal(writer.Bytes()[:10], want) {
		t.Fatalf("header = %x", writer.Bytes()[:10])
	}
	if err := WriteFrame(writer, image.NewRGBA(image.Rect(0, 0, 1, 1))); err == nil {
		t.Fatal("accepted invalid dimensions")
	}
}

type stalledWriter struct{}

func (stalledWriter) Write([]byte) (int, error) { return 0, nil }

func TestWriteStalled(t *testing.T) {
	if err := writeAll(stalledWriter{}, []byte{1}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("error = %v", err)
	}
}
