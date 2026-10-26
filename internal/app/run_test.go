package app

import (
	"context"
	"errors"
	"image"
	"testing"
	"testing/synctest"
	"time"

	"github.com/IgorKha/usb-display-linux/internal/display"
)

type recordingScreen struct {
	times  []time.Time
	cancel context.CancelFunc
}

func (screen *recordingScreen) Show(ctx context.Context, frame image.Image) error {
	screen.times = append(screen.times, time.Now())
	time.Sleep(time.Second)
	if len(screen.times) == 3 {
		screen.cancel()
	}
	return nil
}

func TestCycleInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		renderer, err := display.NewRenderer()
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			if err := renderer.Close(); err != nil {
				t.Error(err)
			}
		}()
		screen := &recordingScreen{cancel: cancel, times: []time.Time{}}
		started := time.Now()
		collect := func(context.Context) ([]display.Page, error) {
			return []display.Page{{Key: "os", Title: []string{"OS"}}, {Key: "en0", Title: []string{"en0"}}}, nil
		}
		err = Cycle(ctx, screen, renderer, 3*time.Second, collect)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v", err)
		}
		if len(screen.times) != 3 || !screen.times[0].Equal(started) {
			t.Fatalf("times = %v", screen.times)
		}
		for index := 1; index < len(screen.times); index++ {
			if delta := screen.times[index].Sub(screen.times[index-1]); delta != 4*time.Second {
				t.Fatalf("delta = %v", delta)
			}
		}
	})
}

func TestNextPosition(t *testing.T) {
	previous := []display.Page{{Key: "os"}, {Key: "a"}, {Key: "b"}, {Key: "c"}}
	current := []display.Page{{Key: "os"}, {Key: "a"}, {Key: "c"}}
	if got := nextPosition(previous, current, 1); got != 2 {
		t.Fatalf("next = %d", got)
	}
	if got := nextPosition(previous, current, 3); got != 0 {
		t.Fatalf("wrap = %d", got)
	}
	if got := nextPosition(previous, current, 2); got != 2 {
		t.Fatalf("removed current next = %d", got)
	}
}

type failedScreen struct{ err error }

func (screen failedScreen) Show(context.Context, image.Image) error { return screen.err }

func TestCycleErrors(t *testing.T) {
	renderer, err := display.NewRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := renderer.Close(); err != nil {
			t.Error(err)
		}
	}()
	failure := errors.New("USB disconnected")
	collect := func(context.Context) ([]display.Page, error) { return []display.Page{{Key: "os"}}, nil }
	err = Cycle(context.Background(), failedScreen{err: failure}, renderer, time.Second, collect)
	if !errors.Is(err, failure) {
		t.Fatalf("lost transport error: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = Cycle(ctx, failedScreen{err: failure}, renderer, time.Second, collect)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation = %v", err)
	}
}
