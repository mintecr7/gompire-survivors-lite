package world

import "testing"

func TestWorldTickDeterministicSmoke(t *testing.T) {
	w1 := NewWorld(2000, 2000)
	w2 := NewWorld(2000, 2000)

	const (
		steps = 300
		dt    = float32(1.0 / 60.0)
	)

	for range steps {
		w1.Enqueue(MsgInput{})
		w2.Enqueue(MsgInput{})
		w1.Tick(dt)
		w2.Tick(dt)
	}

	wantTime := float32(steps) * dt
	if !approxEqual(w1.TimeSurvived, wantTime) {
		t.Fatalf("world did not advance expected time: got %.6f want %.6f", w1.TimeSurvived, wantTime)
	}

	assertWorldEquivalent(t, w1, w2)
	if len(w1.Enemies) == 0 {
		t.Fatal("smoke check failed: expected spawned enemies after ticking")
	}
}

func assertWorldEquivalent(t *testing.T, a, b *World) {
	t.Helper()

	if !approxEqual(a.TimeSurvived, b.TimeSurvived) {
		t.Fatalf("time mismatch: a=%.6f b=%.6f", a.TimeSurvived, b.TimeSurvived)
	}
	if a.GameOver != b.GameOver {
		t.Fatalf("game over mismatch: a=%v b=%v", a.GameOver, b.GameOver)
	}
	if a.Paused != b.Paused {
		t.Fatalf("paused mismatch: a=%v b=%v", a.Paused, b.Paused)
	}

	if !approxEqual(a.Player.Pos.X, b.Player.Pos.X) || !approxEqual(a.Player.Pos.Y, b.Player.Pos.Y) {
		t.Fatalf("player position mismatch: a=(%.6f, %.6f) b=(%.6f, %.6f)",
			a.Player.Pos.X, a.Player.Pos.Y, b.Player.Pos.X, b.Player.Pos.Y)
	}
	if !approxEqual(a.Player.HP, b.Player.HP) {
		t.Fatalf("player hp mismatch: a=%.6f b=%.6f", a.Player.HP, b.Player.HP)
	}
	if a.Player.Level != b.Player.Level {
		t.Fatalf("player level mismatch: a=%d b=%d", a.Player.Level, b.Player.Level)
	}
	if !approxEqual(a.Player.XP, b.Player.XP) {
		t.Fatalf("player xp mismatch: a=%.6f b=%.6f", a.Player.XP, b.Player.XP)
	}

	if a.Stats.EnemiesKilled != b.Stats.EnemiesKilled {
		t.Fatalf("kills mismatch: a=%d b=%d", a.Stats.EnemiesKilled, b.Stats.EnemiesKilled)
	}
	if a.Stats.EnemiesSpawned != b.Stats.EnemiesSpawned {
		t.Fatalf("spawned mismatch: a=%d b=%d", a.Stats.EnemiesSpawned, b.Stats.EnemiesSpawned)
	}
	if !approxEqual(a.Stats.DamageTaken, b.Stats.DamageTaken) {
		t.Fatalf("damage mismatch: a=%.6f b=%.6f", a.Stats.DamageTaken, b.Stats.DamageTaken)
	}

	if len(a.Enemies) != len(b.Enemies) {
		t.Fatalf("enemy count mismatch: a=%d b=%d", len(a.Enemies), len(b.Enemies))
	}
	for i := range a.Enemies {
		ea := a.Enemies[i]
		eb := b.Enemies[i]
		if ea.Kind != eb.Kind {
			t.Fatalf("enemy[%d] kind mismatch: a=%d b=%d", i, ea.Kind, eb.Kind)
		}
		if !approxEqual(ea.Pos.X, eb.Pos.X) || !approxEqual(ea.Pos.Y, eb.Pos.Y) {
			t.Fatalf("enemy[%d] pos mismatch: a=(%.6f, %.6f) b=(%.6f, %.6f)",
				i, ea.Pos.X, ea.Pos.Y, eb.Pos.X, eb.Pos.Y)
		}
		if !approxEqual(ea.HP, eb.HP) {
			t.Fatalf("enemy[%d] hp mismatch: a=%.6f b=%.6f", i, ea.HP, eb.HP)
		}
	}

	if len(a.Orbs) != len(b.Orbs) {
		t.Fatalf("orb count mismatch: a=%d b=%d", len(a.Orbs), len(b.Orbs))
	}
	for i := range a.Orbs {
		oa := a.Orbs[i]
		ob := b.Orbs[i]
		if !approxEqual(oa.Pos.X, ob.Pos.X) || !approxEqual(oa.Pos.Y, ob.Pos.Y) {
			t.Fatalf("orb[%d] pos mismatch: a=(%.6f, %.6f) b=(%.6f, %.6f)",
				i, oa.Pos.X, oa.Pos.Y, ob.Pos.X, ob.Pos.Y)
		}
		if !approxEqual(oa.Value, ob.Value) {
			t.Fatalf("orb[%d] value mismatch: a=%.6f b=%.6f", i, oa.Value, ob.Value)
		}
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
