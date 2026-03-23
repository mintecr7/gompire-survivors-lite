package assets_test

import (
	"strconv"
	"testing"
	"time"

	"horde-lab/internal/assets"
)

func TestLoaderCanDecodeWebPFromMultiplePaths(t *testing.T) {
	l := assets.NewLoader()
	defer l.Close()

	paths := []string{
		"player.webp",
		"assets/player.webp",
		"internal/assets/player.webp",
	}

	for i, p := range paths {
		l.Req <- assets.Request{
			Key:  strconv.Itoa(i),
			Path: p,
		}

		select {
		case res := <-l.Res:
			if res.Err != nil {
				t.Fatalf("load %q failed: %v", p, res.Err)
			}
			b := res.Image.Bounds()
			if b.Dx() <= 0 || b.Dy() <= 0 {
				t.Fatalf("load %q returned invalid bounds: %+v", p, b)
			}
		case <-time.After(500 * time.Millisecond):
			t.Fatalf("timed out waiting for load result for %q", p)
		}
	}
}

func TestLoaderCloseIsIdempotentUnderBackpressure(t *testing.T) {
	l := assets.NewLoader()
	defer l.Close()

	for i := range 256 {
		select {
		case l.Req <- assets.Request{
			Key:  strconv.Itoa(i),
			Path: "player.webp",
		}:
		default:
		}
	}

	time.Sleep(20 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		l.Close()
		l.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("loader close blocked under backpressure")
	}
}
