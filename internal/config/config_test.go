package config

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("interval = '5s'\nport = '/dev/test'\nbrightness = 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		args       []string
		interval   time.Duration
		brightness int
	}{
		{name: "file", interval: 5 * time.Second},
		{name: "explicit default", args: []string{"--interval=3s", "--brightness=30"}, interval: 3 * time.Second, brightness: 30},
		{name: "override", args: []string{"--interval=1s"}, interval: time.Second},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := append([]string{"--config", path}, test.args...)
			got, err := Parse(args, io.Discard)
			if err != nil {
				t.Fatal(err)
			}
			if got.Interval != test.interval || got.Brightness != test.brightness || got.Port != "/dev/test" {
				t.Fatalf("settings = %+v", got)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for _, test := range []struct{ name, content string }{
		{name: "unknown", content: "intervall = '1s'"},
		{name: "zero", content: "interval = '0s'"},
		{name: "negative", content: "interval = '-1s'"},
		{name: "invalid", content: "interval = 'tomorrow'"},
		{name: "brightness", content: "brightness = 101"},
		{name: "port", content: "port = ''"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(test.content), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Parse([]string{"--config", path}, io.Discard); err == nil {
				t.Fatal("accepted invalid config")
			}
		})
	}
	if _, err := Parse([]string{"--config", filepath.Join(t.TempDir(), "missing")}, io.Discard); err == nil {
		t.Fatal("accepted explicitly missing file")
	}
}

func TestParseDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	got, err := Parse(nil, io.Discard)
	if err != nil || got.Interval != 3*time.Second || got.Port != "auto" || got.Brightness != 30 {
		t.Fatalf("defaults = %+v, %v", got, err)
	}
	directory, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "usb-display", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("interval = '7s'\nbrightness = 60"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = Parse([]string{"--brightness=0"}, io.Discard)
	if err != nil || got.Interval != 7*time.Second || got.Brightness != 0 {
		t.Fatalf("default config and zero flag = %+v, %v", got, err)
	}
}
