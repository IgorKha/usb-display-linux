package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func OSRelease(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	name, err := exec.CommandContext(ctx, "/usr/bin/sw_vers", "-productName").Output()
	if err != nil {
		return "", fmt.Errorf("read macOS name: %w", err)
	}
	version, err := exec.CommandContext(ctx, "/usr/bin/sw_vers", "-productVersion").Output()
	if err != nil {
		return "", fmt.Errorf("read macOS version: %w", err)
	}
	return strings.TrimSpace(string(name)) + " " + strings.TrimSpace(string(version)), nil
}
