package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Settings struct {
	Interval   time.Duration
	Port       string
	Brightness int
	Test       bool
	Version    bool
}

type fileSettings struct {
	Interval   *string `toml:"interval"`
	Port       *string `toml:"port"`
	Brightness *int    `toml:"brightness"`
}

func Parse(args []string, output io.Writer) (Settings, error) {
	settings := Settings{Interval: 3 * time.Second, Port: "auto", Brightness: 30}
	flags := flag.NewFlagSet("usb-display", flag.ContinueOnError)
	flags.SetOutput(output)
	configPath := flags.String("config", "", "TOML configuration file (default: user config directory/usb-display/config.toml)")
	interval := flags.String("interval", "3s", "page interval, for example 3s or 500ms")
	port := flags.String("port", "auto", "serial port or auto (only if one USB CDC candidate exists)")
	brightness := flags.Int("brightness", 30, "display brightness, 0 to 100")
	flags.BoolVar(&settings.Test, "test", false, "show a diagnostic frame for five seconds, then exit")
	flags.BoolVar(&settings.Version, "version", false, "print build information and exit")
	if err := flags.Parse(args); err != nil {
		return settings, err
	}
	if flags.NArg() != 0 {
		return settings, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if settings.Version {
		return settings, nil
	}
	explicit := map[string]bool{}
	flags.Visit(func(option *flag.Flag) { explicit[option.Name] = true })
	path := *configPath
	if !explicit["config"] {
		directory, err := os.UserConfigDir()
		if err != nil {
			return settings, fmt.Errorf("find user config directory: %w", err)
		}
		path = filepath.Join(directory, "usb-display", "config.toml")
	}
	raw, err := readConfig(path)
	if err != nil && (!errors.Is(err, os.ErrNotExist) || explicit["config"]) {
		return settings, fmt.Errorf("read config %q: %w", path, err)
	}
	resolvedInterval := "3s"
	if raw.Interval != nil {
		resolvedInterval = *raw.Interval
	}
	if raw.Port != nil {
		settings.Port = *raw.Port
	}
	if raw.Brightness != nil {
		settings.Brightness = *raw.Brightness
	}
	flags.Visit(func(option *flag.Flag) {
		switch option.Name {
		case "interval":
			resolvedInterval = *interval
		case "port":
			settings.Port = *port
		case "brightness":
			settings.Brightness = *brightness
		}
	})
	settings.Interval, err = time.ParseDuration(resolvedInterval)
	if err != nil || settings.Interval <= 0 {
		return settings, fmt.Errorf("interval must be a positive duration, got %q", resolvedInterval)
	}
	if strings.TrimSpace(settings.Port) == "" {
		return settings, errors.New("port must not be empty")
	}
	if settings.Brightness < 0 || settings.Brightness > 100 {
		return settings, errors.New("brightness must be between 0 and 100")
	}
	return settings, nil
}

func readConfig(path string) (fileSettings, error) {
	var settings fileSettings
	file, err := os.Open(path)
	if err != nil {
		return settings, err
	}
	info, err := file.Stat()
	if err != nil {
		return settings, errors.Join(err, file.Close())
	}
	if info.Size() > 1<<20 {
		return settings, errors.Join(errors.New("config exceeds 1 MiB"), file.Close())
	}
	decodeErr := toml.NewDecoder(io.LimitReader(file, 1<<20)).DisallowUnknownFields().Decode(&settings)
	return settings, errors.Join(decodeErr, file.Close())
}
