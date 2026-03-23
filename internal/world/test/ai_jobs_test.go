package world_test

import (
	"testing"

	"horde-lab/internal/jobs"
	"horde-lab/internal/world"
)

func TestTickUsesReadyIntentsOnNextTickWindow(t *testing.T) {
	w := world.NewWorld(2000, 2000)
	defer w.Close()

	w.TestOnlyDisableAIPool()
	w.Cfg.PlayerAttackRange = 0
	w.Player.AttackRange = 0
	w.Player.Pos = world.Vec2{X: 100, Y: 0}
	w.Enemies = []world.Enemy{
		{
			ID:    11,
			Pos:   world.Vec2{X: 100, Y: 100},
			Speed: 10,
			R:     8,
			HP:    1000,
			MaxHP: 1000,
		},
	}

	w.TestOnlySetAIReadyResult(jobs.IntentResult{
		Tick: 0,
		Intents: []jobs.EnemyIntent{
			{
				EnemyID:        11,
				MoveX:          1,
				MoveY:          0,
				SpeedScale:     1,
				PreferredRange: 100,
				Mode:           jobs.IntentModeStrafe,
			},
		},
	})

	before := w.Enemies[0].Pos
	w.Tick(1)
	after := w.Enemies[0].Pos

	if !(after.X > before.X) {
		t.Fatalf("expected enemy to move right using ready intent: beforeX=%.3f afterX=%.3f", before.X, after.X)
	}
}

func TestTickFallsBackToPendingIntentWhenWorkerIsLate(t *testing.T) {
	w := world.NewWorld(2000, 2000)
	defer w.Close()

	w.TestOnlyDisableAIPool()
	w.Cfg.PlayerAttackRange = 0
	w.Player.AttackRange = 0
	w.Player.Pos = world.Vec2{X: 100, Y: 0}
	w.Enemies = []world.Enemy{
		{
			ID:    2,
			Kind:  world.EnemyRunner,
			Pos:   world.Vec2{X: 100, Y: 100},
			Speed: 10,
			R:     7,
			HP:    1000,
			MaxHP: 1000,
		},
	}

	w.TestOnlySetAIPendingRequest(jobs.IntentRequest{
		Tick:    0,
		PlayerX: w.Player.Pos.X,
		PlayerY: w.Player.Pos.Y,
		Enemies: []jobs.EnemySnapshot{
			{EnemyID: 2, Role: jobs.EnemyRoleRunner, X: 100, Y: 100, Radius: 7},
		},
	})

	before := w.Enemies[0].Pos
	w.Tick(1)
	after := w.Enemies[0].Pos

	if !(after.X > before.X) {
		t.Fatalf("expected pending fallback intent to produce rightward strafe: beforeX=%.3f afterX=%.3f", before.X, after.X)
	}
}
