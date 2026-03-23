package world_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"horde-lab/internal/shared/input"
	"horde-lab/internal/world"
)

func TestApplySnapshotRoundTripPreservesWorldState(t *testing.T) {
	source := newSnapshotFixtureWorld()
	defer source.Close()

	snap := source.BuildSnapshot()

	target := world.NewWorld(10, 10)
	defer target.Close()

	if err := target.ApplySnapshot(snap); err != nil {
		t.Fatalf("ApplySnapshot failed: %v", err)
	}

	got := target.BuildSnapshot()
	if !reflect.DeepEqual(got, snap) {
		t.Fatalf("snapshot mismatch after apply\n got: %#v\nwant: %#v", got, snap)
	}

	source.Enqueue(world.MsgInput{Input: input.State{Left: true}})
	target.Enqueue(world.MsgInput{Input: input.State{Left: true}})
	source.Tick(1.0 / 60.0)
	target.Tick(1.0 / 60.0)

	nextSource := source.BuildSnapshot()
	nextTarget := target.BuildSnapshot()
	if !reflect.DeepEqual(nextTarget, nextSource) {
		t.Fatalf("worlds diverged after round-trip and next tick\n got: %#v\nwant: %#v", nextTarget, nextSource)
	}
}

func TestSaveLoadSnapshotFileRoundTrip(t *testing.T) {
	source := newSnapshotFixtureWorld()
	defer source.Close()

	path := filepath.Join(t.TempDir(), "snapshot.json")
	if err := source.SaveSnapshot(path); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	loaded := world.NewWorld(1, 1)
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

func newSnapshotFixtureWorld() *world.World {
	w := world.NewWorld(1234, 567)

	w.Player.Pos = world.Vec2{X: 321, Y: 222}
	w.Player.Speed = 280
	w.Player.AttackTimer = 0.13
	w.Player.AttackCooldown = 0.39
	w.Player.AttackRange = 0
	w.Player.Damage = 47
	w.Player.Weapon = world.WeaponNova
	w.Player.HP = 88
	w.Player.MaxHP = 120
	w.Player.Level = 4
	w.Player.XP = 19
	w.Player.XPToNext = 50
	w.Player.XPMagnet = 26
	w.Player.KnockVel = world.Vec2{X: -12, Y: 4}
	w.Player.Moving = true

	w.Enemies = []world.Enemy{
		{
			ID:          11,
			Pos:         world.Vec2{X: 20, Y: 30},
			Speed:       123,
			R:           9,
			HP:          41,
			MaxHP:       50,
			HitT:        0.4,
			TouchDamage: 10,
			Kind:        world.EnemyNormal,
			XPValue:     5,
		},
		{
			ID:          12,
			Pos:         world.Vec2{X: 70, Y: 90},
			Speed:       190,
			R:           7,
			HP:          21,
			MaxHP:       30,
			HitT:        0.2,
			TouchDamage: 8,
			Kind:        world.EnemyRunner,
			XPValue:     4,
			ShotTimer:   0.8,
		},
	}
	w.Orbs = []world.XPOrb{
		{Pos: world.Vec2{X: 12, Y: 14}, R: 6, Value: 5},
	}
	w.Drops = []world.WeaponDrop{
		{Pos: world.Vec2{X: 40, Y: 44}, R: 9, Kind: world.WeaponFang},
	}
	w.Shots = []world.EnemyProjectile{
		{Pos: world.Vec2{X: 50, Y: 60}, Vel: world.Vec2{X: 5, Y: -3}, R: 4, Damage: 6, Life: 0.7},
	}
	w.Upgrade = world.UpgradeMenu{
		Active: true,
		Options: [2]world.UpgradeOption{
			{Kind: world.UpDamage, Title: "1) +Damage", Desc: "Increase damage by +10"},
			{Kind: world.UpMagnet, Title: "2) Magnet", Desc: "Increase XP pickup radius by +15"},
		},
		Pending: 2,
	}
	w.Stats = world.Stats{
		EnemiesSpawned: 17,
		EnemiesKilled:  9,
		DamageTaken:    22,
		XPCollected:    35,
	}

	for range 60 {
		w.Enqueue(world.MsgInput{Input: input.State{Right: true}})
		w.Tick(1.0 / 60.0)
	}

	return w
}
