package telemetry

import (
	"testing"
	"time"
)

func TestSinkBatchesEvents(t *testing.T) {
	out := make(chan Batch, 8)
	s := newSink(10*time.Millisecond, func(b Batch) {
		out <- b
	})
	defer s.Close()

	s.In <- Event{Kind: "kill", I: 2, At: time.Now()}
	s.In <- Event{Kind: "damage", F: 3.5, At: time.Now()}
	s.In <- Event{Kind: "frame", F: 0.016, At: time.Now()}
	s.In <- Event{Kind: "frame", F: 0.018, At: time.Now()}

	deadline := time.After(700 * time.Millisecond)
	for {
		select {
		case b := <-out:
			// Ignore empty periodic flushes; validate the first non-empty batch.
			if b.Kills == 0 && b.Dmg == 0 && b.Frames == 0 {
				continue
			}

			if b.Kills != 2 {
				t.Fatalf("kills mismatch: got %d want %d", b.Kills, 2)
			}
			if !approxEqual(b.Dmg, 3.5) {
				t.Fatalf("damage mismatch: got %.6f want %.6f", b.Dmg, 3.5)
			}
			if b.Frames != 2 {
				t.Fatalf("frames mismatch: got %d want %d", b.Frames, 2)
			}
			if !approxEqual(b.AvgDt, 0.017) {
				t.Fatalf("avg dt mismatch: got %.6f want %.6f", b.AvgDt, 0.017)
			}
			return

		case <-deadline:
			t.Fatal("timed out waiting for telemetry batch")
		}
	}
}

func TestSinkCloseIsIdempotent(t *testing.T) {
	s := newSink(10*time.Millisecond, nil)

	done := make(chan struct{})
	go func() {
		s.Close()
		s.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("sink close blocked")
	}
}

func approxEqual(a, b float32) bool {
	const eps = 1e-4
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}
