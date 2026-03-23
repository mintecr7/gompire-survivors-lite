package world_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"horde-lab/internal/shared/input"
	"horde-lab/internal/world"
)

func TestBuildReplayHeaderStableForSameSnapshot(t *testing.T) {
	w := newSnapshotFixtureWorld()
	defer w.Close()

	snap := w.BuildSnapshot()

	h1, err := world.BuildReplayHeader(snap, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed: %v", err)
	}
	h2, err := world.BuildReplayHeader(snap, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed on second call: %v", err)
	}

	if !reflect.DeepEqual(h1, h2) {
		t.Fatalf("replay headers should match\n got: %#v\nwant: %#v", h2, h1)
	}
	if h1.Version != world.ReplayVersion {
		t.Fatalf("unexpected replay version: got %d want %d", h1.Version, world.ReplayVersion)
	}
	if h1.ConfigHash == "" {
		t.Fatal("config hash should not be empty")
	}
}

func TestSaveLoadReplayFileRoundTrip(t *testing.T) {
	w := newSnapshotFixtureWorld()
	defer w.Close()

	initial := w.BuildSnapshot()
	header, err := world.BuildReplayHeader(initial, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed: %v", err)
	}

	want := world.ReplayFile{
		Header:  header,
		Initial: initial,
		Frames: []world.ReplayFrame{
			{Tick: 0, Input: input.State{Right: true}},
			{Tick: 1, Input: input.State{Up: true}, Choose: 1},
			{Tick: 2, TogglePause: true},
			{Tick: 3, Restart: true},
		},
	}

	path := filepath.Join(t.TempDir(), "replay.json")
	if err := world.SaveReplayFile(path, want); err != nil {
		t.Fatalf("SaveReplayFile failed: %v", err)
	}

	got, err := world.LoadReplayFile(path)
	if err != nil {
		t.Fatalf("LoadReplayFile failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replay mismatch after save/load\n got: %#v\nwant: %#v", got, want)
	}
}

func TestReplayFramesReproduceFinalSnapshot(t *testing.T) {
	const dt = float32(1.0 / 60.0)

	original := world.NewWorld(2000, 2000)
	defer original.Close()

	initial := original.BuildSnapshot()
	frames := deterministicReplayFrames()

	runReplayFrames(original, frames, dt)
	wantFinal := original.BuildSnapshot()

	replayed := world.NewWorld(1, 1)
	defer replayed.Close()

	if err := replayed.ApplySnapshot(initial); err != nil {
		t.Fatalf("ApplySnapshot failed: %v", err)
	}

	runReplayFrames(replayed, frames, dt)
	gotFinal := replayed.BuildSnapshot()

	if !reflect.DeepEqual(gotFinal, wantFinal) {
		t.Fatalf("replay did not reproduce final snapshot\n got: %#v\nwant: %#v", gotFinal, wantFinal)
	}
}

func runReplayFrames(w *world.World, frames []world.ReplayFrame, dt float32) {
	for _, frame := range frames {
		w.Enqueue(world.MsgInput{Input: frame.Input})
		if frame.Restart {
			w.Enqueue(world.MsgRestart{})
		}
		if frame.TogglePause {
			w.Enqueue(world.MsgTogglePause{})
		}
		if frame.Choose == 0 || frame.Choose == 1 {
			w.Enqueue(world.MsgChooseUpgrade{Choice: frame.Choose})
		}
		w.Tick(dt)
	}
}

func deterministicReplayFrames() []world.ReplayFrame {
	frames := make([]world.ReplayFrame, 0, 180)

	for tick := uint64(0); tick < 180; tick++ {
		frame := world.ReplayFrame{Tick: tick, Choose: -1}

		switch {
		case tick < 45:
			frame.Input = input.State{Right: true}
		case tick < 90:
			frame.Input = input.State{Down: true}
		case tick < 135:
			frame.Input = input.State{Left: true, Down: true}
		default:
			frame.Input = input.State{Up: true}
		}

		// Brief pause window exercises replay of control messages too.
		if tick == 60 || tick == 70 {
			frame.TogglePause = true
		}

		frames = append(frames, frame)
	}

	return frames
}
