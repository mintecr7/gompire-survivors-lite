package assets

import (
	"strconv"
	"testing"
	"time"
)

func TestLoadImageWebPFromMultiplePaths(t *testing.T) {
	paths := []string{
		"player.webp",
		"assets/player.webp",
		"internal/assets/player.webp",
	}

	for _, p := range paths {
		img, err := loadImage(p)
		if err != nil {
			t.Fatalf("loadImage(%q) failed: %v", p, err)
		}

		b := img.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 {
			t.Fatalf("loadImage(%q) returned invalid bounds: %+v", p, b)
		}
	}
}

func TestLoaderCloseIsIdempotentUnderBackpressure(t *testing.T) {
	l := NewLoader()
	defer l.Close()

	for i := range 256 {
		select {
		case l.Req <- Request{
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
