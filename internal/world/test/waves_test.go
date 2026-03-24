package world_test

import (
	"reflect"
	"testing"

	"horde-lab/internal/world"
)

func TestWaveDirectorAdvancesAndPersistsAcrossSnapshots(t *testing.T) {
	w1 := world.NewWorld(2000, 2000)
	defer w1.Close()
	neutralizeEnemyPressure(w1)

	const (
		steps = 2700
		dt    = float32(1.0 / 60.0)
	)

	for range steps {
		w1.Enqueue(world.MsgInput{})
		w1.Tick(dt)
	}

	snap := w1.BuildSnapshot()
	if snap.Wave.Index < 3 {
		t.Fatalf("expected wave progression, got wave %d", snap.Wave.Index)
	}
	if snap.Wave.Label == "" {
		t.Fatal("expected generated wave label")
	}
	if snap.Wave.TankWeight <= 0 {
		t.Fatalf("expected tank weight in later waves, got %+v", snap.Wave)
	}
	if snap.SpawnEvery >= snap.Cfg.BaseSpawnEvery {
		t.Fatalf("expected wave pacing to accelerate spawning: got %.3f want < %.3f", snap.SpawnEvery, snap.Cfg.BaseSpawnEvery)
	}

	w2 := world.NewWorld(1, 1)
	defer w2.Close()
	neutralizeEnemyPressure(w2)

	if err := w2.ApplySnapshot(snap); err != nil {
		t.Fatalf("ApplySnapshot failed: %v", err)
	}

	got := w2.BuildSnapshot()
	if !reflect.DeepEqual(got.Wave, snap.Wave) {
		t.Fatalf("wave state mismatch after apply\n got: %#v\nwant: %#v", got.Wave, snap.Wave)
	}

	w1.Enqueue(world.MsgInput{})
	w2.Enqueue(world.MsgInput{})
	w1.Tick(dt)
	w2.Tick(dt)

	next1 := w1.BuildSnapshot()
	next2 := w2.BuildSnapshot()
	if !reflect.DeepEqual(next2, next1) {
		t.Fatalf("worlds diverged after wave snapshot round-trip\n got: %#v\nwant: %#v", next2, next1)
	}
}

func neutralizeEnemyPressure(w *world.World) {
	w.Cfg.EnemySpeed = 0
	w.Cfg.EnemyRunnerSpeed = 0
	w.Cfg.EnemyTankSpeed = 0
	w.Cfg.EnemyTouchDamage = 0
	w.Cfg.EnemyRunnerTouchDamage = 0
	w.Cfg.EnemyTankTouchDamage = 0
}
