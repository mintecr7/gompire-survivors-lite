package world

import (
	"testing"

	"horde-lab/internal/jobs"
)

func TestConsumeAIIntentsForTickFallsBackToPendingRequest(t *testing.T) {
	w := NewWorld(2000, 2000)
	defer w.Close()

	req := jobs.IntentRequest{
		Tick:    9,
		PlayerX: 120,
		PlayerY: 20,
		Enemies: []jobs.EnemySnapshot{
			{EnemyID: 7, Role: jobs.EnemyRoleRunner, X: 120, Y: 120, Radius: 7},
		},
	}

	w.aiPendingRequests[req.Tick] = req

	got := w.consumeAIIntentsForTick(req.Tick)
	want := intentsFromResult(jobs.ComputeIntents(req))

	if len(got) != len(want) {
		t.Fatalf("intent count mismatch: got %d want %d", len(got), len(want))
	}

	for enemyID, wi := range want {
		gi, ok := got[enemyID]
		if !ok {
			t.Fatalf("missing intent for enemy %d", enemyID)
		}
		if !approxEqual(gi.Dir.X, wi.Dir.X) || !approxEqual(gi.Dir.Y, wi.Dir.Y) {
			t.Fatalf("direction mismatch for enemy %d: got (%.4f, %.4f) want (%.4f, %.4f)",
				enemyID, gi.Dir.X, gi.Dir.Y, wi.Dir.X, wi.Dir.Y)
		}
		if !approxEqual(gi.SpeedScale, wi.SpeedScale) {
			t.Fatalf("speed scale mismatch for enemy %d: got %.4f want %.4f",
				enemyID, gi.SpeedScale, wi.SpeedScale)
		}
		if gi.Mode != wi.Mode {
			t.Fatalf("mode mismatch for enemy %d: got %d want %d", enemyID, gi.Mode, wi.Mode)
		}
		if !approxEqual(gi.PreferredRange, wi.PreferredRange) {
			t.Fatalf("preferred range mismatch for enemy %d: got %.4f want %.4f",
				enemyID, gi.PreferredRange, wi.PreferredRange)
		}
	}

	if _, ok := w.aiPendingRequests[req.Tick]; ok {
		t.Fatalf("pending request for tick %d should be removed after fallback", req.Tick)
	}
}

func TestTickUsesReadyIntentsOnNextTickWindow(t *testing.T) {
	w := NewWorld(2000, 2000)
	defer w.Close()

	// Keep this test focused on movement behavior.
	w.aiPool.Close()
	w.aiPool = nil
	w.Cfg.PlayerAttackRange = 0
	w.spawnEvery = 9999
	w.Player.Pos = Vec2{X: 100, Y: 0}
	w.Enemies = []Enemy{
		{
			ID:    11,
			Pos:   Vec2{X: 100, Y: 100},
			Speed: 10,
			R:     8,
			HP:    1000,
			MaxHP: 1000,
		},
	}

	w.aiReadyResults[0] = jobs.IntentResult{
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
	}

	before := w.Enemies[0].Pos
	w.Tick(1)
	after := w.Enemies[0].Pos

	if !(after.X > before.X) {
		t.Fatalf("expected enemy to move right using ready intent: beforeX=%.3f afterX=%.3f", before.X, after.X)
	}
}

func TestTickFallsBackToPendingIntentWhenWorkerIsLate(t *testing.T) {
	w := NewWorld(2000, 2000)
	defer w.Close()

	// Keep this test focused on movement behavior.
	w.aiPool.Close()
	w.aiPool = nil
	w.Cfg.PlayerAttackRange = 0
	w.spawnEvery = 9999
	w.Player.Pos = Vec2{X: 100, Y: 0}
	w.Enemies = []Enemy{
		{
			ID:    2,
			Kind:  EnemyRunner,
			Pos:   Vec2{X: 100, Y: 100},
			Speed: 10,
			R:     7,
			HP:    1000,
			MaxHP: 1000,
		},
	}

	w.aiPendingRequests[0] = jobs.IntentRequest{
		Tick:    0,
		PlayerX: w.Player.Pos.X,
		PlayerY: w.Player.Pos.Y,
		Enemies: []jobs.EnemySnapshot{
			{EnemyID: 2, Role: jobs.EnemyRoleRunner, X: 100, Y: 100, Radius: 7},
		},
	}

	before := w.Enemies[0].Pos
	w.Tick(1)
	after := w.Enemies[0].Pos

	// Runner fallback intent at this range should include a strafe component.
	if !(after.X > before.X) {
		t.Fatalf("expected pending fallback intent to produce rightward strafe: beforeX=%.3f afterX=%.3f", before.X, after.X)
	}
}
