package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/IgorKha/usb-display-linux/internal/app"
	"github.com/IgorKha/usb-display-linux/internal/config"
	"github.com/IgorKha/usb-display-linux/internal/display"
	"github.com/IgorKha/usb-display-linux/internal/monitor"
	"github.com/IgorKha/usb-display-linux/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	settings, err := config.Parse(args, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if settings.Version {
		if err := json.NewEncoder(stdout).Encode(version.Info()); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := execute(ctx, settings, stderr); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func execute(ctx context.Context, settings config.Settings, stderr io.Writer) (result error) {
	renderer, err := display.NewRenderer()
	if err != nil {
		return fmt.Errorf("load embedded font: %w", err)
	}
	defer func() { result = errors.Join(result, renderer.Close()) }()
	name, err := display.ResolvePort(settings.Port)
	if err != nil {
		return err
	}
	device, err := display.Open(ctx, name, settings.Brightness)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, device.Close()) }()
	device.Rotation = settings.Rotation
	fmt.Fprintf(stderr, "display %s, firmware %q, interval %s\n", name, device.Firmware, settings.Interval)
	if settings.Test {
		if err := device.Show(ctx, renderer.TestPattern()); err != nil {
			return err
		}
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	osName, err := monitor.OSRelease(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		fmt.Fprintf(stderr, "warning: OS release unavailable: %v\n", err)
		osName = runtime.GOOS + " (OS DATA ERROR)"
	}
	collect := func(ctx context.Context) ([]display.Page, error) {
		interfaces, err := monitor.CollectNetwork(ctx)
		if err != nil {
			return nil, err
		}
		return renderer.Pages(osName, interfaces), nil
	}
	return app.Cycle(ctx, device, renderer, settings.Interval, collect)
}
