package app

import (
	"context"
	"errors"
	"fmt"
	"image"
	"time"

	"github.com/IgorKha/usb-display-linux/internal/display"
)

type Screen interface {
	Show(context.Context, image.Image) error
}

func Cycle(ctx context.Context, screen Screen, renderer *display.Renderer, interval time.Duration, collect func(context.Context) ([]display.Page, error)) error {
	if interval <= 0 {
		return errors.New("page interval must be positive")
	}
	pages, err := collect(ctx)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return errors.New("no pages to display")
	}
	position := 0
	timer := time.NewTimer(interval)
	defer timer.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := screen.Show(ctx, renderer.Render(pages[position], position, len(pages))); err != nil {
			return fmt.Errorf("send display frame: %w", err)
		}
		timer.Reset(interval)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
		updated, err := collect(ctx)
		if err != nil {
			return err
		}
		if len(updated) == 0 {
			return errors.New("no pages to display")
		}
		position = nextPosition(pages, updated, position)
		pages = updated
	}
}

func nextPosition(previous, current []display.Page, position int) int {
	for step := 1; step <= len(previous); step++ {
		candidate := previous[(position+step)%len(previous)].Key
		for index, page := range current {
			if page.Key == candidate {
				return index
			}
		}
	}
	return 0
}
