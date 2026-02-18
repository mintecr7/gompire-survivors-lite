package world

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"horde-lab/internal/jobs"
)

const SnapshotVersion = 1

type Snapshot struct {
	Version int `json:"version"`

	W float32 `json:"w"`
	H float32 `json:"h"`

	Cfg Config `json:"cfg"`

	Player  Player  `json:"player"`
	Enemies []Enemy `json:"enemies"`
	Orbs    []XPOrb `json:"orbs"`

	SpawnTimer float32 `json:"spawn_timer"`
	SpawnEvery float32 `json:"spawn_every"`

	LastAttackPos Vec2    `json:"last_attack_pos"`
	LastAttackT   float32 `json:"last_attack_t"`

	TimeSurvived float32     `json:"time_survived"`
	GameOver     bool        `json:"game_over"`
	Paused       bool        `json:"paused"`
	Upgrade      UpgradeMenu `json:"upgrade"`
	Stats        Stats       `json:"stats"`

	ShakeT     float32 `json:"shake_t"`
	ShakePhase float32 `json:"shake_phase"`
	ShakeOff   Vec2    `json:"shake_off"`

	NextEnemyID int    `json:"next_enemy_id"`
	AITick      uint64 `json:"ai_tick"`

	RNGSeed  int64  `json:"rng_seed"`
	RNGCalls uint64 `json:"rng_calls"`
}

func (w *World) BuildSnapshot() Snapshot {
	enemies := make([]Enemy, len(w.Enemies))
	copy(enemies, w.Enemies)

	orbs := make([]XPOrb, len(w.Orbs))
	copy(orbs, w.Orbs)

	return Snapshot{
		Version: SnapshotVersion,
		W:       w.W,
		H:       w.H,
		Cfg:     w.Cfg,

		Player:  w.Player,
		Enemies: enemies,
		Orbs:    orbs,

		SpawnTimer: w.spawnTimer,
		SpawnEvery: w.spawnEvery,

		LastAttackPos: w.LastAttackPos,
		LastAttackT:   w.LastAttackT,

		TimeSurvived: w.TimeSurvived,
		GameOver:     w.GameOver,
		Paused:       w.Paused,
		Upgrade:      w.Upgrade,
		Stats:        w.Stats,

		ShakeT:     w.ShakeT,
		ShakePhase: w.ShakePhase,
		ShakeOff:   w.ShakeOff,

		NextEnemyID: w.nextEnemyID,
		AITick:      w.aiTick,

		RNGSeed:  w.rngSeed,
		RNGCalls: w.rngCalls,
	}
}

func (w *World) ApplySnapshot(s Snapshot) error {
	if s.Version != SnapshotVersion {
		return fmt.Errorf("unsupported snapshot version: got %d want %d", s.Version, SnapshotVersion)
	}
	if s.W <= 0 || s.H <= 0 {
		return fmt.Errorf("invalid world size in snapshot: w=%.3f h=%.3f", s.W, s.H)
	}

	w.W = s.W
	w.H = s.H
	w.Cfg = s.Cfg

	w.Player = s.Player
	w.Enemies = make([]Enemy, len(s.Enemies))
	copy(w.Enemies, s.Enemies)
	w.Orbs = make([]XPOrb, len(s.Orbs))
	copy(w.Orbs, s.Orbs)

	w.spawnTimer = s.SpawnTimer
	w.spawnEvery = s.SpawnEvery

	w.LastAttackPos = s.LastAttackPos
	w.LastAttackT = s.LastAttackT

	w.TimeSurvived = s.TimeSurvived
	w.GameOver = s.GameOver
	w.Paused = s.Paused
	w.Upgrade = s.Upgrade
	w.Stats = s.Stats

	w.ShakeT = s.ShakeT
	w.ShakePhase = s.ShakePhase
	w.ShakeOff = s.ShakeOff

	w.nextEnemyID = s.NextEnemyID
	w.aiTick = s.AITick

	w.rngSeed = s.RNGSeed
	if w.rngSeed == 0 {
		w.rngSeed = 1
	}
	w.rng = nil
	w.rngCalls = 0
	w.ensureRNG()
	for range s.RNGCalls {
		_ = w.randFloat32()
	}

	if w.aiPendingRequests == nil {
		w.aiPendingRequests = make(map[uint64]jobs.IntentRequest, 8)
	} else {
		clear(w.aiPendingRequests)
	}
	if w.aiReadyResults == nil {
		w.aiReadyResults = make(map[uint64]jobs.IntentResult, 8)
	} else {
		clear(w.aiReadyResults)
	}
	if w.aiPool == nil {
		w.aiPool = newAIPool()
	}

	return nil
}

func (w *World) SaveSnapshot(path string) error {
	if path == "" {
		return fmt.Errorf("snapshot path is empty")
	}

	s := w.BuildSnapshot()
	blob, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensure snapshot dir: %w", err)
		}
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, blob, 0o644); err != nil {
		return fmt.Errorf("write snapshot temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename snapshot temp file: %w", err)
	}

	return nil
}

func (w *World) LoadSnapshot(path string) error {
	if path == "" {
		return fmt.Errorf("snapshot path is empty")
	}

	blob, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read snapshot file: %w", err)
	}

	var s Snapshot
	if err := json.Unmarshal(blob, &s); err != nil {
		return fmt.Errorf("decode snapshot file: %w", err)
	}

	if err := w.ApplySnapshot(s); err != nil {
		return fmt.Errorf("apply snapshot: %w", err)
	}
	return nil
}
