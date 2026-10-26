package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version", "--config=/missing/file"}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d: %s", code, stderr.String())
	}
	info := map[string]string{}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if info["version"] == "" || info["os"] == "" || stderr.Len() != 0 {
		t.Fatalf("version = %v; stderr = %s", info, stderr.String())
	}
}

func TestRunHelpAndInvalidConfig(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{name: "help", args: []string{"--help"}},
		{name: "missing config", args: []string{"--config=/missing/file"}, code: 2},
		{name: "positional argument", args: []string{"unexpected"}, code: 2},
		{name: "bad flag", args: []string{"--unknown"}, code: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(test.args, &stdout, &stderr); code != test.code {
				t.Fatalf("exit = %d, want %d", code, test.code)
			}
			if stdout.Len() != 0 {
				t.Fatal("diagnostics leaked to stdout")
			}
		})
	}
}
