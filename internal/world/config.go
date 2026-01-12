package world

import "math"

type Config struct {
	// World / pacing
	BaseSpawnEvery float32
	MinSpawnEvery  float32
	RampEvery      float32
	RampFactor     float32
	SoftEnemyCap   int
	SpawnRadius    float32

	// Player
	PlayerRadius         float32
	PlayerSpeed          float32
	PlayerMaxHP          float32
	PlayerHurtCooldown   float32
	PlayerLevelUpHeal    float32
	PlayerAttackCooldown float32
	PlayerAttackRange    float32
	PlayerDamage         float32

	// Knockback feel
	PlayerKnockbackSpeed   float32
	PlayerKnockbackDamping float32

	// Enemy
	EnemyRadius      float32
	EnemySpeed       float32
	EnemyHP          float32
	EnemyTouchDamage float32

	// XP
	XPOrbRadius     float32
	XPPickupPadding float32
	XPPerKill       float32
	XPBaseToNext    float32
	XPGrowthToNext  float64

	// Visual timers
	LastAttackMax float32

	// Screen shake (on taking damage)
	HitShakeDuration  float32
	HitShakeMagnitude float32
	HitShakeFreq1     float32
	HitShakeFreq2     float32

	// Enemy archetypes
	// 1. Runner
	EnemyRunnerRadius      float32
	EnemyRunnerSpeed       float32
	EnemyRunnerHP          float32
	EnemyRunnerTouchDamage float32
	EnemyRunnerXP          float32

	// 2. Tank
	EnemyTankRadius      float32
	EnemyTankSpeed       float32
	EnemyTankHP          float32
	EnemyTankTouchDamage float32
	EnemyTankXP          float32
}

func DefaultConfig() Config {
	return Config{
		BaseSpawnEvery: 0.75,
		MinSpawnEvery:  0.20,
		RampEvery:      15.0,
		RampFactor:     0.92,
		SoftEnemyCap:   140,
		SpawnRadius:    420,

		PlayerRadius:         10,
		PlayerSpeed:          260,
		PlayerMaxHP:          100,
		PlayerHurtCooldown:   0.35,
		PlayerLevelUpHeal:    15,
		PlayerAttackCooldown: 0.45,
		PlayerAttackRange:    180,
		PlayerDamage:         25,

		PlayerKnockbackSpeed:   520,
		PlayerKnockbackDamping: 18,

		EnemyRadius:      9,
		EnemySpeed:       120,
		EnemyHP:          50,
		EnemyTouchDamage: 10,

		XPOrbRadius:     6,
		XPPickupPadding: 10,
		XPPerKill:       5,

		XPBaseToNext:   25,
		XPGrowthToNext: 1.28,

		LastAttackMax: 0.08,

		HitShakeDuration:  0.12,
		HitShakeMagnitude: 6.0,
		HitShakeFreq1:     26.0,
		HitShakeFreq2:     33.0,

		EnemyRunnerRadius:      7,
		EnemyRunnerSpeed:       190,
		EnemyRunnerHP:          30,
		EnemyRunnerTouchDamage: 8,
		EnemyRunnerXP:          4,

		EnemyTankRadius:      14,
		EnemyTankSpeed:       75,
		EnemyTankHP:          140,
		EnemyTankTouchDamage: 18,
		EnemyTankXP:          12,
	}
}

func (c Config) XPToNext(level int) float32 {
	// level 1 -> base, multiplicative growth thereafter
	if level < 1 {
		level = 1
	}
	return float32(float64(c.XPBaseToNext) * math.Pow(c.XPGrowthToNext, float64(level-1)))
}
