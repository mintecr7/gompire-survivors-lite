package world

import (
	"runtime"

	"horde-lab/internal/jobs"
)

type enemyMoveIntent struct {
	Dir            Vec2
	SpeedScale     float32
	PreferredRange float32
	Mode           jobs.IntentMode
}

func newAIPool() *jobs.IntentPool {
	workers := runtime.NumCPU() / 2
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}

	return jobs.NewIntentPool(workers, 16)
}

func (w *World) drainAIResults() {
	if w.aiPool == nil {
		return
	}

	for {
		select {
		case res := <-w.aiPool.Res:
			// Drop stale results that are older than the previous tick window.
			if res.Tick+1 < w.aiTick {
				continue
			}
			w.aiReadyResults[res.Tick] = res
		default:
			return
		}
	}
}

func (w *World) consumeAIIntentsForTick(tick uint64) map[int]enemyMoveIntent {
	if res, ok := w.aiReadyResults[tick]; ok {
		delete(w.aiReadyResults, tick)
		delete(w.aiPendingRequests, tick)
		return intentsFromResult(res)
	}

	// Deterministic fallback: compute synchronously from the exact snapshot
	// that was submitted for this tick if workers were late.
	if req, ok := w.aiPendingRequests[tick]; ok {
		delete(w.aiPendingRequests, tick)
		return intentsFromResult(jobs.ComputeIntents(req))
	}

	return nil
}

func (w *World) submitAIJob(tick uint64) {
	if w.aiPool == nil || len(w.Enemies) == 0 {
		return
	}

	req := jobs.IntentRequest{
		Tick:    tick,
		PlayerX: w.Player.Pos.X,
		PlayerY: w.Player.Pos.Y,
		Enemies: make([]jobs.EnemySnapshot, len(w.Enemies)),
	}

	for i, e := range w.Enemies {
		req.Enemies[i] = jobs.EnemySnapshot{
			EnemyID: e.ID,
			Role:    roleFromEnemyKind(e.Kind),
			X:       e.Pos.X,
			Y:       e.Pos.Y,
			Radius:  e.R,
		}
	}

	w.aiPendingRequests[tick] = req

	select {
	case w.aiPool.Req <- req:
	default:
		// Queue full: synchronous fallback at consume time will handle it.
	}

	w.pruneAIState(tick)
}

func (w *World) pruneAIState(currentTick uint64) {
	if currentTick <= 8 {
		return
	}

	cutoff := currentTick - 8
	for tick := range w.aiPendingRequests {
		if tick < cutoff {
			delete(w.aiPendingRequests, tick)
		}
	}
	for tick := range w.aiReadyResults {
		if tick < cutoff {
			delete(w.aiReadyResults, tick)
		}
	}
}

func intentsFromResult(res jobs.IntentResult) map[int]enemyMoveIntent {
	if len(res.Intents) == 0 {
		return nil
	}

	out := make(map[int]enemyMoveIntent, len(res.Intents))
	for _, in := range res.Intents {
		out[in.EnemyID] = enemyMoveIntent{
			Dir: Vec2{
				X: in.MoveX,
				Y: in.MoveY,
			},
			SpeedScale:     in.SpeedScale,
			PreferredRange: in.PreferredRange,
			Mode:           in.Mode,
		}
	}
	return out
}

func roleFromEnemyKind(kind EnemyKind) jobs.EnemyRole {
	switch kind {
	case EnemyRunner:
		return jobs.EnemyRoleRunner
	case EnemyTank:
		return jobs.EnemyRoleTank
	default:
		return jobs.EnemyRoleNormal
	}
}
