package world_test

import (
	"reflect"
	"testing"

	"horde-lab/internal/shared/input"
	"horde-lab/internal/world"
)

func TestArenaObstaclesGenerateDeterministically(t *testing.T) {
	w1 := world.NewWorld(2000, 2000)
	w2 := world.NewWorld(2000, 2000)
	defer w1.Close()
	defer w2.Close()

	if len(w1.Obstacles) == 0 {
		t.Fatal("expected generated obstacles")
	}
	if !reflect.DeepEqual(w1.Obstacles, w2.Obstacles) {
		t.Fatalf("obstacles should be deterministic\n got: %#v\nwant: %#v", w2.Obstacles, w1.Obstacles)
	}

	for _, obstacle := range w1.Obstacles {
		minSafe := obstacle.R + w1.Cfg.StartSafeRadius
		if dist2f(w1.Player.Pos, obstacle.Pos) < minSafe*minSafe {
			t.Fatalf("obstacle generated inside safe radius: %#v", obstacle)
		}
	}
}

func TestPlayerMovementRespectsObstacleCollision(t *testing.T) {
	w := world.NewWorld(2000, 2000)
	defer w.Close()

	if len(w.Obstacles) == 0 {
		t.Fatal("expected generated obstacles")
	}

	obstacle := w.Obstacles[0]
	w.Player.Pos = world.Vec2{
		X: obstacle.Pos.X - obstacle.R - w.Cfg.ObstaclePadding - w.Player.R - 40,
		Y: obstacle.Pos.Y,
	}

	const dt = float32(1.0 / 60.0)
	for range 120 {
		w.Enqueue(world.MsgInput{Input: input.State{Right: true}})
		w.Tick(dt)
	}

	minDist := obstacle.R + w.Cfg.ObstaclePadding + w.Player.R
	if dist2f(w.Player.Pos, obstacle.Pos) < minDist*minDist {
		t.Fatalf("player overlapped obstacle after movement: player=%#v obstacle=%#v", w.Player.Pos, obstacle)
	}
}

func dist2f(a, b world.Vec2) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}
