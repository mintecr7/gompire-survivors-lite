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
			{EnemyID: 1, X: 0, Y: 0},
			{EnemyID: 2, X: 10, Y: 0},
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
	if i0.EnemyID != 1 || !almostEq(i0.DirX, 1) || !almostEq(i0.DirY, 0) {
		t.Fatalf("unexpected first intent: %+v", i0)
	}

	i1 := got.Intents[1]
	if i1.EnemyID != 2 || !almostEq(i1.DirX, 0) || !almostEq(i1.DirY, 0) {
		t.Fatalf("unexpected second intent: %+v", i1)
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
			{EnemyID: 5, X: 1, Y: -2},
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
