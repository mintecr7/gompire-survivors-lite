package world

import (
	"path/filepath"
	"reflect"
	"testing"

	"horde-lab/internal/jobs"
)

func TestApplySnapshotRoundTripPreservesWorldState(t *testing.T) {
	source := newSnapshotFixtureWorld()
	defer source.Close()

	snap := source.BuildSnapshot()

	target := NewWorld(10, 10)
	defer target.Close()

	target.aiPendingRequests[99] = jobs.IntentRequest{Tick: 99}
	target.aiReadyResults[98] = jobs.IntentResult{Tick: 98}
	target.aiPool.Close()
	target.aiPool = nil

	if err := target.ApplySnapshot(snap); err != nil {
		t.Fatalf("ApplySnapshot failed: %v", err)
	}

	got := target.BuildSnapshot()
	if !reflect.DeepEqual(got, snap) {
		t.Fatalf("snapshot mismatch after apply\n got: %#v\nwant: %#v", got, snap)
	}

	if len(target.aiPendingRequests) != 0 {
		t.Fatalf("pending AI requests should be cleared, got %d", len(target.aiPendingRequests))
	}
	if len(target.aiReadyResults) != 0 {
		t.Fatalf("ready AI results should be cleared, got %d", len(target.aiReadyResults))
	}
	if target.aiPool == nil {
		t.Fatal("ai pool should be reinitialized on ApplySnapshot")
	}

	wantNextRand := source.randFloat32()
	gotNextRand := target.randFloat32()
	if wantNextRand != gotNextRand {
		t.Fatalf("restored RNG state mismatch: got %.8f want %.8f", gotNextRand, wantNextRand)
	}
}

func TestSaveLoadSnapshotFileRoundTrip(t *testing.T) {
	source := newSnapshotFixtureWorld()
	defer source.Close()

	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := source.SaveSnapshot(path); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	loaded := NewWorld(1, 1)
	defer loaded.Close()

	if err := loaded.LoadSnapshot(path); err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	want := source.BuildSnapshot()
	got := loaded.BuildSnapshot()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshot mismatch after save/load\n got: %#v\nwant: %#v", got, want)
	}
}

func newSnapshotFixtureWorld() *World {
	w := NewWorld(1234, 567)

	w.Player.Pos = Vec2{X: 321, Y: 222}
	w.Player.Speed = 280
	w.Player.AttackTimer = 0.13
	w.Player.AttackCooldown = 0.39
	w.Player.AttackRange = 205
	w.Player.Damage = 47
	w.Player.Weapon = WeaponNova
	w.Player.HP = 88
	w.Player.MaxHP = 120
	w.Player.Level = 4
	w.Player.XP = 19
	w.Player.XPToNext = 50
	w.Player.XPMagnet = 26
	w.Player.KnockVel = Vec2{X: -12, Y: 4}
	w.Player.Moving = true

	w.Enemies = []Enemy{
		{
			ID:          11,
			Pos:         Vec2{X: 20, Y: 30},
			Speed:       123,
			R:           9,
			HP:          41,
			MaxHP:       50,
			HitT:        0.4,
			TouchDamage: 10,
			Kind:        EnemyNormal,
			XPValue:     5,
		},
		{
			ID:          12,
			Pos:         Vec2{X: 70, Y: 90},
			Speed:       190,
			R:           7,
			HP:          21,
			MaxHP:       30,
			HitT:        0.2,
			TouchDamage: 8,
			Kind:        EnemyRunner,
			XPValue:     4,
			ShotTimer:   0.8,
		},
	}
	w.Orbs = []XPOrb{
		{Pos: Vec2{X: 12, Y: 14}, R: 6, Value: 5},
	}
	w.Drops = []WeaponDrop{
		{Pos: Vec2{X: 40, Y: 44}, R: 9, Kind: WeaponFang},
	}
	w.Shots = []EnemyProjectile{
		{Pos: Vec2{X: 50, Y: 60}, Vel: Vec2{X: 5, Y: -3}, R: 4, Damage: 6, Life: 0.7},
	}

	w.spawnTimer = 0.25
	w.spawnEvery = 0.61
	w.LastAttackPos = Vec2{X: 45, Y: 47}
	w.LastAttackT = 0.05
	w.LastAttackRadius = 108
	w.LastAttackWeapon = WeaponNova
	w.TimeSurvived = 33.5
	w.GameOver = false
	w.Paused = true
	w.Upgrade = UpgradeMenu{
		Active: true,
		Options: [2]UpgradeOption{
			{Kind: UpDamage, Title: "1) +Damage", Desc: "Increase damage by +10"},
			{Kind: UpMagnet, Title: "2) Magnet", Desc: "Increase XP pickup radius by +15"},
		},
		Pending: 2,
	}
	w.Stats = Stats{
		EnemiesSpawned: 17,
		EnemiesKilled:  9,
		DamageTaken:    22,
		XPCollected:    35,
	}
	w.ShakeT = 0.1
	w.ShakePhase = 2.3
	w.ShakeOff = Vec2{X: 1.5, Y: -0.5}
	w.nextEnemyID = 77
	w.aiTick = 12

	_ = w.randFloat32()
	_ = w.randIntn(100)

	return w
}
