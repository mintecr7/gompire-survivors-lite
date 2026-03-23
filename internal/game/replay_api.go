package game

import (
	"fmt"
	"time"

	"horde-lab/internal/world"
)

func RecordReplay(w *world.World, fixedStep time.Duration, frames []world.ReplayFrame) (world.ReplayFile, error) {
	if w == nil {
		return world.ReplayFile{}, fmt.Errorf("world is nil")
	}
	if fixedStep <= 0 {
		return world.ReplayFile{}, fmt.Errorf("fixed step must be positive")
	}

	g := &Game{
		w:         w,
		fixedStep: fixedStep,
	}
	g.resetReplayRecording()

	for _, frame := range frames {
		g.enqueueReplayFrame(frame)
		g.replay.Frames = append(g.replay.Frames, frame)
		g.replayTick++
		g.w.Tick(float32(g.fixedStep.Seconds()))
	}

	return g.replay, nil
}

func PlayReplay(w *world.World, rep world.ReplayFile) error {
	if w == nil {
		return fmt.Errorf("world is nil")
	}
	if rep.Header.FixedStepSeconds <= 0 {
		return fmt.Errorf("replay fixed step must be positive")
	}
	if err := w.ApplySnapshot(rep.Initial); err != nil {
		return err
	}

	g := &Game{
		w:      w,
		replay: rep,
	}

	for g.replayFrameIdx < len(g.replay.Frames) {
		frame := g.replay.Frames[g.replayFrameIdx]
		g.enqueueReplayFrame(frame)
		g.replayFrameIdx++
		g.w.Tick(rep.Header.FixedStepSeconds)
	}

	return nil
}
