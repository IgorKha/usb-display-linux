package display

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

const (
	Width  = 160
	Height = 80
)

func WriteFrame(writer io.Writer, frame image.Image) error {
	if frame.Bounds().Dx() != Width || frame.Bounds().Dy() != Height {
		return fmt.Errorf("frame must be %dx%d", Width, Height)
	}
	header := []byte{0x05, 0, 0, 0, 0, Width - 1, 0, Height - 1, 0, 0x0a}
	if err := writeAll(writer, header); err != nil {
		return fmt.Errorf("write bitmap header: %w", err)
	}
	pixels := RGB565(frame)
	for offset := 0; offset < len(pixels); offset += Width * 4 {
		if err := writeAll(writer, pixels[offset:offset+Width*4]); err != nil {
			return fmt.Errorf("write bitmap pixels: %w", err)
		}
	}
	return nil
}

func RGB565(frame image.Image) []byte {
	bounds := frame.Bounds()
	pixels := make([]byte, 0, bounds.Dx()*bounds.Dy()*2)
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			red, green, blue, _ := frame.At(column, row).RGBA()
			value := uint16((red>>11)<<11 | (green>>10)<<5 | blue>>11)
			pixels = binary.LittleEndian.AppendUint16(pixels, value)
		}
	}
	return pixels
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		written, err := writer.Write(data)
		if err != nil {
			return err
		}
		if written <= 0 || written > len(data) {
			return io.ErrShortWrite
		}
		data = data[written:]
	}
	return nil
}
