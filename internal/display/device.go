package display

import (
	"context"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/jacobsa/go-serial/serial"
	"golang.org/x/sys/unix"
)

type Device struct {
	file     *os.File
	Firmware string
	Rotation int
}

func ResolvePort(name string) (string, error) {
	if name != "auto" {
		return name, nil
	}
	pattern := "/dev/cu.usbmodem*"
	if runtime.GOOS == "linux" {
		pattern = "/dev/ttyACM*"
	}
	ports, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	slices.Sort(ports)
	if len(ports) != 1 {
		return "", fmt.Errorf("found %d candidate display ports (%s); select one using --port", len(ports), strings.Join(ports, ", "))
	}
	return ports[0], nil
}

func Open(ctx context.Context, name string, brightness int) (*Device, error) {
	if brightness < 0 || brightness > 100 {
		return nil, errors.New("brightness must be between 0 and 100")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	connection, err := serial.Open(serial.OpenOptions{
		PortName:          name,
		BaudRate:          115200,
		DataBits:          8,
		StopBits:          1,
		RTSCTSFlowControl: true,
		MinimumReadSize:   1,
	})
	if err != nil {
		return nil, fmt.Errorf("open display %q (check port permissions and other applications): %w", name, err)
	}
	original, ok := connection.(*os.File)
	if !ok {
		return nil, errors.Join(errors.New("serial transport is not a Unix file"), connection.Close())
	}
	descriptor, err := unix.Dup(int(original.Fd()))
	err = errors.Join(err, original.Close())
	if err != nil {
		if descriptor >= 0 {
			err = errors.Join(err, unix.Close(descriptor))
		}
		return nil, err
	}
	if err := unix.SetNonblock(descriptor, true); err != nil {
		return nil, errors.Join(err, unix.Close(descriptor))
	}
	if err := unix.IoctlSetInt(descriptor, unix.TIOCEXCL, 0); err != nil {
		return nil, errors.Join(fmt.Errorf("lock serial port: %w", err), unix.Close(descriptor))
	}
	file := os.NewFile(uintptr(descriptor), name)
	device := &Device{file: file}
	if err := file.SetDeadline(time.Now().Add(time.Second)); err != nil {
		return nil, errors.Join(fmt.Errorf("serial deadlines unavailable: %w", err), file.Close())
	}
	if err := device.initialize(ctx, brightness); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return device, nil
}

func (device *Device) operation(ctx context.Context, action func() error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	deadline := time.Now().Add(3 * time.Second)
	if limit, ok := ctx.Deadline(); ok && limit.Before(deadline) {
		deadline = limit
	}
	if err := device.file.SetDeadline(deadline); err != nil {
		return err
	}
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		device.file.SetDeadline(time.Now())
		close(finished)
	})
	err := action()
	if !stop() {
		<-finished
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func (device *Device) initialize(ctx context.Context, brightness int) error {
	query, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	err := device.operation(query, func() error {
		if err := writeAll(device.file, []byte{0xc2, 0x0a}); err != nil {
			return err
		}
		response := make([]byte, 19)
		if _, err := io.ReadFull(device.file, response); err != nil {
			return err
		}
		device.Firmware = strings.TrimSpace(string(response[1:9]))
		return nil
	})
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, os.ErrDeadlineExceeded) {
		return fmt.Errorf("query display firmware: %w", err)
	}
	return device.operation(ctx, func() error {
		if err := writeAll(device.file, []byte{2, 2, 10}); err != nil {
			return err
		}
		return writeAll(device.file, []byte{3, byte(brightness * 255 / 100), 0xe8, 3, 10})
	})
}

func (device *Device) Show(ctx context.Context, frame image.Image) error {
	if device.Rotation == 180 {
		frame = rotate180(frame)
	}
	return device.operation(ctx, func() error { return WriteFrame(device.file, frame) })
}

func rotate180(frame image.Image) *image.RGBA {
	bounds := frame.Bounds()
	rotated := image.NewRGBA(bounds)
	for row := bounds.Min.Y; row < bounds.Max.Y; row++ {
		for column := bounds.Min.X; column < bounds.Max.X; column++ {
			rotated.Set(column, row, frame.At(bounds.Min.X+bounds.Max.X-1-column, bounds.Min.Y+bounds.Max.Y-1-row))
		}
	}
	return rotated
}

func (device *Device) Close() error {
	deadlineErr := device.file.SetWriteDeadline(time.Now().Add(300 * time.Millisecond))
	var offErr error
	if deadlineErr == nil {
		offErr = writeAll(device.file, []byte{3, 0, 0, 0, 10, 7, 10})
	}
	return errors.Join(deadlineErr, offErr, device.file.Close())
}
