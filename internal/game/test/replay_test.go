package game_test

import (
	"reflect"
	"testing"
	"time"

	"horde-lab/internal/game"
	"horde-lab/internal/shared/input"
	"horde-lab/internal/world"
)

func TestRecordAndPlayReplayRoundTrip(t *testing.T) {
	const fixedStep = time.Second / 60

	recordedWorld := world.NewWorld(2000, 2000)
	defer recordedWorld.Close()

	initial := recordedWorld.BuildSnapshot()
	frames := deterministicReplayFrames()

	rep, err := game.RecordReplay(recordedWorld, fixedStep, frames)
	if err != nil {
		t.Fatalf("RecordReplay failed: %v", err)
	}

	if !reflect.DeepEqual(rep.Initial, initial) {
		t.Fatalf("recorded replay initial snapshot mismatch\n got: %#v\nwant: %#v", rep.Initial, initial)
	}
	if !reflect.DeepEqual(rep.Frames, frames) {
		t.Fatalf("recorded replay frames mismatch\n got: %#v\nwant: %#v", rep.Frames, frames)
	}

	wantFinal := recordedWorld.BuildSnapshot()

	replayedWorld := world.NewWorld(1, 1)
	defer replayedWorld.Close()

	if err := game.PlayReplay(replayedWorld, rep); err != nil {
		t.Fatalf("PlayReplay failed: %v", err)
	}

	gotFinal := replayedWorld.BuildSnapshot()
	if !reflect.DeepEqual(gotFinal, wantFinal) {
		t.Fatalf("recorded replay did not reproduce final snapshot\n got: %#v\nwant: %#v", gotFinal, wantFinal)
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

		if tick == 60 || tick == 70 {
			frame.TogglePause = true
		}

		frames = append(frames, frame)
	}

	return frames
}
