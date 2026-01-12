package world

import (
	"math/rand"

	"horde-lab/internal/shared/input"
)

type XPOrb struct {
	Pos   Vec2
	R     float32
	Value float32
}

type MsgInput struct{ Input input.State }

type World struct {
	W, H float32

	inbox []Msg // TODO: use channel

	Cfg     Config
	Orbs    []XPOrb
	Player  Player
	Enemies []Enemy

	// spawning
	spawnTimer float32
	spawnEvery float32
	rng        *rand.Rand

	// attack visualization
	LastAttackPos Vec2
	LastAttackT   float32

	// run state
	TimeSurvived float32
	GameOver     bool
	Paused       bool
	Upgrade      UpgradeMenu

	// stats
	Stats Stats

	// difficulty
	MinSpawnEvery float32
	RampEvery     float32
	RampFactor    float32
	SoftEnemyCap  int

	// screen shake state
	ShakeT     float32
	ShakePhase float32
	ShakeOff   Vec2
}

type Player struct {
	Pos   Vec2
	Speed float32
	R     float32

	// combat (auto attack)
	AttackCooldown float32 // seconds
	AttackTimer    float32 // counts down to 0
	AttackRange    float32
	Damage         float32

	// health / damage taken
	HP           float32
	MaxHP        float32
	HurtCooldown float32 // invulnerable window after taking damage
	HurtTimer    float32

	// progression
	Level    int
	XP       float32
	XPToNext float32
	XPMagnet float32

	// knockback
	KnockVel Vec2
}

type Enemy struct {
	Pos   Vec2
	Speed float32
	R     float32

	// combat
	HP    float32
	MaxHP float32
	HitT  float32 // hit flash timer (seconds)

	TouchDamage float32 // damage when colliding with player

	// archetype
	Kind    EnemyKind
	XPValue float32
}

type Stats struct {
	EnemiesSpawned int
	EnemiesKilled  int
	DamageTaken    float32
	XPCollected    float32
}

type EnemyKind int
