package monitor

import (
	"context"
	"errors"
	"os"
)

func OSRelease(ctx context.Context) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	file, err := os.Open("/etc/os-release")
	if errors.Is(err, os.ErrNotExist) {
		file, err = os.Open("/usr/lib/os-release")
	}
	if errors.Is(err, os.ErrNotExist) {
		return "Linux", nil
	}
	if err != nil {
		return "", err
	}
	name, parseErr := parseOSRelease(file)
	return name, errors.Join(parseErr, file.Close())
}
