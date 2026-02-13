package jobs

import (
	"testing"
	"time"
)

func TestComputeIntents(t *testing.T) {
	req := IntentRequest{
		Tick:    42,
		PlayerX: 10,
		PlayerY: 0,
		Enemies: []EnemySnapshot{
			{EnemyID: 1, Role: EnemyRoleNormal, X: 0, Y: 0, Radius: 9},
			{EnemyID: 2, Role: EnemyRoleRunner, X: 10, Y: 0, Radius: 7},
		},
	}

	got := ComputeIntents(req)

	if got.Tick != 42 {
		t.Fatalf("tick mismatch: got %d want %d", got.Tick, 42)
	}
	if len(got.Intents) != 2 {
		t.Fatalf("intent length mismatch: got %d want %d", len(got.Intents), 2)
	}

	i0 := got.Intents[0]
	if i0.EnemyID != 1 || !almostEq(i0.MoveX, 1) || !almostEq(i0.MoveY, 0) {
		t.Fatalf("unexpected first intent: %+v", i0)
	}
	if i0.Mode != IntentModePressure {
		t.Fatalf("unexpected mode for first intent: got %d want %d", i0.Mode, IntentModePressure)
	}
	if !almostEq(i0.PreferredRange, 65) {
		t.Fatalf("unexpected preferred range for first intent: got %.2f", i0.PreferredRange)
	}
	if !almostEq(i0.SpeedScale, 0.85) {
		t.Fatalf("unexpected speed scale for first intent: got %.2f", i0.SpeedScale)
	}

	i1 := got.Intents[1]
	if i1.EnemyID != 2 {
		t.Fatalf("unexpected second intent: %+v", i1)
	}
	if i1.Mode != IntentModeKite {
		t.Fatalf("unexpected mode for second intent: got %d want %d", i1.Mode, IntentModeKite)
	}
	if !almostEq(i1.SpeedScale, 1.28) {
		t.Fatalf("unexpected speed scale for second intent: got %.2f", i1.SpeedScale)
	}
	if i1.MoveX >= 0 || i1.MoveY <= 0 {
		t.Fatalf("runner kite direction should be up-left, got (%.3f, %.3f)", i1.MoveX, i1.MoveY)
	}
}

func TestIntentPoolDeliversResults(t *testing.T) {
	pool := NewIntentPool(2, 8)
	defer pool.Close()

	req := IntentRequest{
		Tick:    7,
		PlayerX: 8,
		PlayerY: -2,
		Enemies: []EnemySnapshot{
			{EnemyID: 5, Role: EnemyRoleTank, X: 1, Y: -2, Radius: 14},
		},
	}

	pool.Req <- req

	select {
	case res := <-pool.Res:
		if res.Tick != 7 {
			t.Fatalf("tick mismatch: got %d want %d", res.Tick, 7)
		}
		if len(res.Intents) != 1 {
			t.Fatalf("intent length mismatch: got %d want %d", len(res.Intents), 1)
		}
		if res.Intents[0].EnemyID != 5 {
			t.Fatalf("enemy id mismatch: got %d want %d", res.Intents[0].EnemyID, 5)
		}
		if res.Intents[0].Mode != IntentModeHold {
			t.Fatalf("unexpected mode: got %d want %d", res.Intents[0].Mode, IntentModeHold)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timed out waiting for intent result")
	}
}

func almostEq(a, b float32) bool {
	const eps = 1e-4
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}
