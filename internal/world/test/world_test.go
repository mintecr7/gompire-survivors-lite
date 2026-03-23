package world_test

import (
	"reflect"
	"testing"

	"horde-lab/internal/world"
)

func TestWorldTickDeterministicSmoke(t *testing.T) {
	w1 := world.NewWorld(2000, 2000)
	w2 := world.NewWorld(2000, 2000)
	defer w1.Close()
	defer w2.Close()

	const (
		steps = 300
		dt    = float32(1.0 / 60.0)
	)

	for range steps {
		w1.Enqueue(world.MsgInput{})
		w2.Enqueue(world.MsgInput{})
		w1.Tick(dt)
		w2.Tick(dt)
	}

	wantTime := float32(steps) * dt
	if !approxEqual(w1.BuildSnapshot().TimeSurvived, wantTime) {
		t.Fatalf("world did not advance expected time: got %.6f want %.6f", w1.BuildSnapshot().TimeSurvived, wantTime)
	}

	s1 := w1.BuildSnapshot()
	s2 := w2.BuildSnapshot()
	if !reflect.DeepEqual(s1, s2) {
		t.Fatalf("world snapshots diverged\n got: %#v\nwant: %#v", s1, s2)
	}
	if len(s1.Enemies) == 0 {
		t.Fatal("smoke check failed: expected spawned enemies after ticking")
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
