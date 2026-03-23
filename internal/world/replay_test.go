package world

import (
	"path/filepath"
	"reflect"
	"testing"

	"horde-lab/internal/shared/input"
)

func TestBuildReplayHeaderStableForSameSnapshot(t *testing.T) {
	w := newSnapshotFixtureWorld()
	defer w.Close()

	snap := w.BuildSnapshot()

	h1, err := BuildReplayHeader(snap, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed: %v", err)
	}
	h2, err := BuildReplayHeader(snap, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed on second call: %v", err)
	}

	if !reflect.DeepEqual(h1, h2) {
		t.Fatalf("replay headers should match\n got: %#v\nwant: %#v", h2, h1)
	}
	if h1.Version != ReplayVersion {
		t.Fatalf("unexpected replay version: got %d want %d", h1.Version, ReplayVersion)
	}
	if h1.ConfigHash == "" {
		t.Fatal("config hash should not be empty")
	}
}

func TestSaveLoadReplayFileRoundTrip(t *testing.T) {
	w := newSnapshotFixtureWorld()
	defer w.Close()

	initial := w.BuildSnapshot()
	header, err := BuildReplayHeader(initial, 1.0/60.0)
	if err != nil {
		t.Fatalf("BuildReplayHeader failed: %v", err)
	}

	want := ReplayFile{
		Header:  header,
		Initial: initial,
		Frames: []ReplayFrame{
			{Tick: 0, Input: input.State{Right: true}},
			{Tick: 1, Input: input.State{Up: true}, Choose: 1},
			{Tick: 2, TogglePause: true},
			{Tick: 3, Restart: true},
		},
	}

	path := filepath.Join(t.TempDir(), "replay.json")
	if err := SaveReplayFile(path, want); err != nil {
		t.Fatalf("SaveReplayFile failed: %v", err)
	}

	got, err := LoadReplayFile(path)
	if err != nil {
		t.Fatalf("LoadReplayFile failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("replay mismatch after save/load\n got: %#v\nwant: %#v", got, want)
	}
}
