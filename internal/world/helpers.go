package world

import (
	"math"

	"horde-lab/internal/jobs"
)

var base float32 = float32(0.75)

// ============================================================================
// SPAWNING & DIFFICULTY
// ============================================================================

func (w *World) updateSpawning(dt float32) {

	// soft cap: if too many enemies, slow spawning instead pf hard stopping
	// Example: above cap, effective spawn interval increase linearly.

	effectiveEvery := w.spawnEvery
	cap := w.Cfg.SoftEnemyCap
	if cap > 0 && len(w.Enemies) > cap {
		over := float32(len(w.Enemies)-cap) / float32(cap)

		effectiveEvery *= (1 + over)
	}

	w.spawnTimer += dt
	for w.spawnTimer >= effectiveEvery {
		w.spawnTimer -= effectiveEvery
		w.spawnEnemyNearPlayer()
	}
}

func (w *World) spawnEnemyNearPlayer() {

	cfg := w.Cfg

	spawnRadius := cfg.SpawnRadius
	// spawn position in a ring around player
	ang := w.randFloat32() * 2 * math.Pi
	off := Vec2{
		X: float32(math.Cos(float64(ang))) * spawnRadius,
		Y: float32(math.Sin(float64(ang))) * spawnRadius,
	}

	pos := w.Player.Pos.Add(off)

	// Clamp to world bounds so enemies always exist in-world
	pos.X = clamp(pos.X, 0, w.W)
	pos.Y = clamp(pos.Y, 0, w.H)

	kind := w.chooseEnemyKind()

	e := Enemy{
		ID:   w.nextEnemyID,
		Pos:  pos,
		Kind: kind,
	}
	w.nextEnemyID++

	switch kind {
	case EnemyTank:
		e.R = cfg.EnemyTankRadius
		e.Speed = cfg.EnemyTankSpeed
		e.MaxHP = cfg.EnemyTankHP
		e.HP = cfg.EnemyTankHP
		e.TouchDamage = cfg.EnemyTankTouchDamage
		e.XPValue = cfg.EnemyTankXP

	case EnemyRunner:
		e.R = cfg.EnemyRunnerRadius
		e.Speed = cfg.EnemyRunnerSpeed
		e.MaxHP = cfg.EnemyRunnerHP
		e.HP = cfg.EnemyRunnerHP
		e.TouchDamage = cfg.EnemyRunnerTouchDamage
		e.XPValue = cfg.EnemyRunnerXP

	default: // normal
		e.R = cfg.EnemyRadius
		e.Speed = cfg.EnemySpeed
		e.MaxHP = cfg.EnemyHP
		e.HP = cfg.EnemyHP
		e.TouchDamage = cfg.EnemyTouchDamage
		e.XPValue = cfg.XPPerKill
	}
	w.Enemies = append(w.Enemies, e)
	w.Stats.EnemiesSpawned++
}

func (w *World) updateDifficulty() {
	// Ramp based on time survived: every RampEvery seconds reduce spawnEvery by RampFactor
	cfg := w.Cfg
	if cfg.RampEvery <= 0 {
		return
	}

	steps := int(w.TimeSurvived / cfg.RampEvery)

	target := cfg.BaseSpawnEvery
	for range steps {
		target *= cfg.RampFactor
	}
	if target < cfg.MinSpawnEvery {
		target = cfg.MinSpawnEvery
	}

	w.spawnEvery = target // update spawn interval

}

// ============================================================================
// ENEMY MOVEMENT & AI
// ============================================================================

func (w *World) updateEnemies(dt float32, intents map[int]enemyMoveIntent) {
	p := w.Player.Pos
	for i := range w.Enemies {
		e := &w.Enemies[i]
		// hit flash decay
		if e.HitT > 0 {
			e.HitT -= dt
			if e.HitT < 0 {
				e.HitT = 0
			}
		}
		speedScale := float32(1)
		dir, ok := Vec2{}, false
		if in, has := intents[e.ID]; has {
			dir = in.Dir
			speedScale = clamp(in.SpeedScale, 0.2, 1.5)
			dir = resolveIntentDirection(e.ID, p.Sub(e.Pos), in, dir)
			ok = true
		}
		if !ok {
			toP := p.Sub(e.Pos)
			if toP.X == 0 && toP.Y == 0 {
				continue
			}
			dir = toP.Norm()
		}

		if dir.X == 0 && dir.Y == 0 {
			continue
		}
		e.Pos = e.Pos.Add(dir.Mul(e.Speed * speedScale * dt))

		// Clamp
		e.Pos.X = clamp(e.Pos.X, 0, w.W)
		e.Pos.Y = clamp(e.Pos.Y, 0, w.H)
	}
}

func resolveIntentDirection(enemyID int, toPlayer Vec2, in enemyMoveIntent, baseDir Vec2) Vec2 {
	dir := baseDir
	dist := toPlayer.Len()
	if dist <= 0 {
		return dir
	}

	toPlayer = toPlayer.Mul(1 / dist)
	preferred := in.PreferredRange
	if preferred <= 0 {
		preferred = dist
	}

	// Keep intent motion around a target range so modes remain readable in-game.
	rangeErr := clamp((dist-preferred)/maxf(preferred, 1), -1, 1)
	rangeGain := modeRangeGain(in.Mode)
	dir = dir.Add(toPlayer.Mul(rangeErr * rangeGain))

	strafeGain := modeStrafeGain(in.Mode)
	if strafeGain > 0 {
		tangent := Vec2{X: -toPlayer.Y, Y: toPlayer.X}
		if enemyID%2 != 0 {
			tangent = Vec2{X: toPlayer.Y, Y: -toPlayer.X}
		}
		dir = dir.Add(tangent.Mul(strafeGain))
	}

	if dir.X == 0 && dir.Y == 0 {
		return toPlayer
	}
	return dir.Norm()
}

func modeRangeGain(mode jobs.IntentMode) float32 {
	switch mode {
	case jobs.IntentModeKite:
		return 1.05
	case jobs.IntentModeHold:
		return 0.85
	case jobs.IntentModeStrafe:
		return 0.55
	case jobs.IntentModePressure:
		return 0.30
	default:
		return 0.45
	}
}

func modeStrafeGain(mode jobs.IntentMode) float32 {
	switch mode {
	case jobs.IntentModeStrafe:
		return 0.35
	case jobs.IntentModeKite:
		return 0.20
	case jobs.IntentModeHold:
		return 0.10
	default:
		return 0
	}
}

// ============================================================================
// COMBAT SYSTEM
// ============================================================================

func (w *World) updateCombat(dt float32) {
	// cooldown timer
	if w.Player.AttackTimer > 0 {
		w.Player.AttackTimer -= dt
		if w.Player.AttackTimer > 0 {
			return
		}
	}

	wd := weaponDef(w.Player.Weapon)
	attackRange := w.Player.AttackRange * wd.RangeMul
	damage := w.Player.Damage * wd.DamageMul
	nextCooldown := maxf(0.08, w.Player.AttackCooldown*wd.CooldownMul)
	fired := false

	switch wd.AttackStyle {
	case AttackPierce:
		idxs := w.nearestEnemiesInRange(w.Player.Pos, attackRange, 2)
		if len(idxs) == 0 {
			return
		}
		fired = true
		w.LastAttackPos = w.Enemies[idxs[0]].Pos
		sortIdxDesc(idxs)
		for _, idx := range idxs {
			w.damageEnemyAt(idx, damage)
		}
	case AttackRadial:
		rad := wd.AttackRadius
		if rad <= 0 {
			rad = attackRange
		}
		idxs := w.nearestEnemiesInRange(w.Player.Pos, rad, 64)
		if len(idxs) == 0 {
			return
		}
		fired = true
		w.LastAttackPos = w.Player.Pos
		w.LastAttackRadius = rad
		sortIdxDesc(idxs)
		for _, idx := range idxs {
			w.damageEnemyAt(idx, damage)
		}
	default:
		idx := w.nearestEnemyInRange(w.Player.Pos, attackRange)
		if idx < 0 {
			return
		}
		fired = true
		w.LastAttackPos = w.Enemies[idx].Pos
		w.damageEnemyAt(idx, damage)
	}
	if fired {
		w.Player.AttackTimer = nextCooldown
		w.LastAttackT = 0.08
		w.LastAttackWeapon = w.Player.Weapon
	}
}

func (w *World) updateContactDamage(dt float32) {

	// invulnerability timer
	if w.Player.HurtTimer > 0 {
		w.Player.HurtTimer -= dt
		if w.Player.HurtTimer > 0 {
			return
		}

		w.Player.HurtTimer = 0
	}

	pr := w.Player.R
	p := w.Player.Pos

	// if touching any enemy, take damage once per HurtCooldown.
	for i := range w.Enemies {
		e := &w.Enemies[i]
		rr := pr + e.R
		if dist2(p, e.Pos) < rr*rr {
			w.Player.HP -= e.TouchDamage
			w.Stats.DamageTaken += e.TouchDamage
			w.Player.HurtTimer = w.Cfg.PlayerHurtCooldown

			// Knockback
			dir := w.Player.Pos.Sub(e.Pos).Norm()
			if dir.X == 0 && dir.Y == 0 {
				// rare overlap: pick a deterministic-ish random direction
				ang := w.randFloat32() * 2 * math.Pi
				dir = Vec2{
					X: float32(math.Cos(float64(ang))),
					Y: float32(math.Sin(float64(ang))),
				}
			} else {
				dir = dir.Norm()
			}

			w.Player.KnockVel = dir.Mul(w.Cfg.PlayerKnockbackSpeed)

			// trigger/refresh shake
			w.ShakeT = w.Cfg.HitShakeDuration

			if w.Player.HP <= 0 {
				w.Player.HP = 0
				w.GameOver = true
			}

			return
		}
	}

}

func (w *World) updateRunnerRangedShots(dt float32) {
	const (
		minRange  = float32(42)
		maxRange  = float32(170)
		shotSpeed = float32(360)
		shotLife  = float32(1.05)
		shotDmg   = float32(7)
	)

	hasNova := w.Player.Weapon == WeaponNova
	p := w.Player.Pos
	for i := range w.Enemies {
		e := &w.Enemies[i]
		if e.Kind != EnemyRunner {
			continue
		}
		if e.ShotTimer > 0 {
			e.ShotTimer -= dt
			if e.ShotTimer < 0 {
				e.ShotTimer = 0
			}
		}
		if !hasNova || e.ShotTimer > 0 {
			continue
		}

		toPlayer := p.Sub(e.Pos)
		dist := toPlayer.Len()
		if dist < minRange || dist > maxRange {
			continue
		}

		dir := toPlayer.Norm()
		if dir.X == 0 && dir.Y == 0 {
			continue
		}

		w.Shots = append(w.Shots, EnemyProjectile{
			Pos:    e.Pos,
			Vel:    dir.Mul(shotSpeed),
			R:      4,
			Damage: shotDmg,
			Life:   shotLife,
		})
		e.ShotTimer = 1.15
	}
}

func (w *World) updateEnemyProjectiles(dt float32) {
	p := w.Player.Pos
	for i := 0; i < len(w.Shots); {
		s := w.Shots[i]
		s.Pos = s.Pos.Add(s.Vel.Mul(dt))
		s.Life -= dt

		if s.Life <= 0 || s.Pos.X < 0 || s.Pos.X > w.W || s.Pos.Y < 0 || s.Pos.Y > w.H {
			w.removeShotAt(i)
			continue
		}

		rr := w.Player.R + s.R
		if dist2(p, s.Pos) <= rr*rr {
			if w.Player.HurtTimer <= 0 {
				w.Player.HP -= s.Damage
				w.Stats.DamageTaken += s.Damage
				w.Player.HurtTimer = w.Cfg.PlayerHurtCooldown
				if w.Player.HP <= 0 {
					w.Player.HP = 0
					w.GameOver = true
				}
			}
			w.removeShotAt(i)
			continue
		}

		w.Shots[i] = s
		i++
	}
}

// ============================================================================
// PLAYER MOVEMENT & PHYSICS
// ============================================================================

func (w *World) updateKnockback(dt float32) {
	kv := w.Player.KnockVel
	if kv.X == 0 && kv.Y == 0 {
		return
	}
	// integrate
	w.Player.Pos = w.Player.Pos.Add(kv.Mul(dt))

	// damping (euler integration)
	d := w.Cfg.PlayerKnockbackDamping
	f := 1 - d*dt
	if f < 0 {
		f = 0
	}
	w.Player.KnockVel = w.Player.KnockVel.Mul(f)

	// clamp + stop tiny velocity
	w.Player.Pos.X = clamp(w.Player.Pos.X, 0, w.W)
	w.Player.Pos.Y = clamp(w.Player.Pos.Y, 0, w.H)

	if absf(w.Player.KnockVel.X)+absf(w.Player.KnockVel.Y) < 1 {
		w.Player.KnockVel = Vec2{}
	}
}

func (w *World) updateShake(dt float32) {
	if w.ShakeT <= 0 {
		w.ShakeOff = Vec2{}
		return
	}

	w.ShakeT -= dt
	if w.ShakeT <= 0 {
		w.ShakeT = 0
		w.ShakeOff = Vec2{}
		return
	}

	// Fade out as timer decreases
	t := w.ShakeT / w.Cfg.HitShakeDuration

	if t < 0 {
		t = 0
	}

	if t > 1 {
		t = 1
	}

	amp := w.Cfg.HitShakeMagnitude * t

	w.ShakeOff = Vec2{
		X: float32(math.Sin(float64(w.ShakePhase*w.Cfg.HitShakeFreq1))) * amp,
		Y: float32(math.Cos(float64(w.ShakePhase)*float64(w.Cfg.HitShakeFreq2))) * amp,
	}
}

// ============================================================================
// XP & LEVELING SYSTEM
// ============================================================================

func (w *World) spawnXPOrb(pos Vec2, value float32) {
	w.Orbs = append(w.Orbs, XPOrb{
		Pos:   pos,
		R:     w.Cfg.XPOrbRadius,
		Value: value,
	})
}

func (w *World) updateXPOrbs(dt float32) {
	_ = dt // reserved for future motion/magnetism

	p := w.Player.Pos
	pickupR := w.Player.R + w.Cfg.XPPickupPadding + w.Player.XPMagnet
	// pickup padding

	for i := 0; i < len(w.Orbs); {
		o := w.Orbs[i]
		rr := pickupR + o.R

		if dist2(p, o.Pos) <= rr*rr {
			w.Player.XP += o.Value
			w.Stats.XPCollected += o.Value
			w.removeOrbAt(i)
			continue
		}
		i++
	}
}

func (w *World) updateWeaponDrops() {
	p := w.Player.Pos
	pickupR := w.Player.R + 14

	for i := 0; i < len(w.Drops); {
		d := w.Drops[i]
		rr := pickupR + d.R
		if dist2(p, d.Pos) <= rr*rr {
			w.Player.Weapon = d.Kind
			w.removeDropAt(i)
			continue
		}
		i++
	}
}

func (w *World) updateLevelUp() {
	// If menu is already active, don't process more levels rights now.
	if w.Upgrade.Active {
		return
	}

	leveled := false

	for w.Player.XP >= w.Player.XPToNext {
		w.Player.XP -= w.Player.XPToNext
		w.Player.Level++
		w.Player.XPToNext = w.Cfg.XPToNext(w.Player.Level)

		// queue one upgrade choice per level
		w.Upgrade.Pending++
		leveled = true

		// v0.1 simple reward: small heal on level up
		w.Player.HP = minf(w.Player.MaxHP, w.Player.HP+w.Cfg.PlayerLevelUpHeal)
		w.Player.MaxHP += w.Cfg.PlayerLevelUpHeal
	}

	if leveled {
		w.openUpgradeMenuIfNeeded()
	}
}

// ============================================================================
// UTILITY & HELPER FUNCTIONS
// ============================================================================

func (w *World) chooseEnemyKind() EnemyKind {
	// deterministic pattern based on spawn count
	// every 12th is a tank, every 4th is a runner, otherwise normal

	n := w.Stats.EnemiesSpawned + 1

	if n%12 == 0 {
		return EnemyTank
	}
	if n%4 == 0 {
		return EnemyRunner
	}
	return EnemyNormal
}

func (w *World) nearestEnemyInRange(p Vec2, rng float32) int {
	if len(w.Enemies) == 0 {
		return -1
	}

	r2 := rng * rng
	best := -1
	bestD2 := float32(0)

	for i := range w.Enemies {
		d := w.Enemies[i].Pos.Sub(p)
		d2 := d.X*d.X + d.Y*d.Y

		if d2 > r2 {
			continue
		}

		if best == -1 || d2 < bestD2 {
			best = i
			bestD2 = d2
		}
	}

	return best
}

func (w *World) removeEnemyAt(idx int) {
	last := len(w.Enemies) - 1

	if idx != last {
		w.Enemies[idx] = w.Enemies[last]
	}

	w.Enemies = w.Enemies[:last]
}

func (w *World) removeOrbAt(i int) {
	last := len(w.Orbs) - 1
	if i != last {
		w.Orbs[i] = w.Orbs[last]
	}
	w.Orbs = w.Orbs[:last]
}

func (w *World) removeDropAt(i int) {
	last := len(w.Drops) - 1
	if i != last {
		w.Drops[i] = w.Drops[last]
	}
	w.Drops = w.Drops[:last]
}

func (w *World) removeShotAt(i int) {
	last := len(w.Shots) - 1
	if i != last {
		w.Shots[i] = w.Shots[last]
	}
	w.Shots = w.Shots[:last]
}

func (w *World) maybeSpawnWeaponDrop(pos Vec2, kind EnemyKind) {
	if w.randFloat32() > weaponDropChance(kind) {
		return
	}

	weapon := w.randomWeaponKind()
	def := weaponDef(weapon)
	w.Drops = append(w.Drops, WeaponDrop{
		Pos:  pos,
		R:    def.DropRadius,
		Kind: weapon,
	})
}

func (w *World) nearestEnemiesInRange(p Vec2, rng float32, limit int) []int {
	if len(w.Enemies) == 0 || limit <= 0 {
		return nil
	}

	type cand struct {
		idx int
		d2  float32
	}
	r2 := rng * rng
	cands := make([]cand, 0, len(w.Enemies))
	for i := range w.Enemies {
		d := w.Enemies[i].Pos.Sub(p)
		d2 := d.X*d.X + d.Y*d.Y
		if d2 <= r2 {
			cands = append(cands, cand{idx: i, d2: d2})
		}
	}
	if len(cands) == 0 {
		return nil
	}

	for i := 0; i < len(cands)-1; i++ {
		best := i
		for j := i + 1; j < len(cands); j++ {
			if cands[j].d2 < cands[best].d2 {
				best = j
			}
		}
		cands[i], cands[best] = cands[best], cands[i]
	}

	if len(cands) > limit {
		cands = cands[:limit]
	}

	out := make([]int, len(cands))
	for i := range cands {
		out[i] = cands[i].idx
	}
	return out
}

func (w *World) damageEnemyAt(idx int, dmg float32) {
	if idx < 0 || idx >= len(w.Enemies) {
		return
	}
	e := &w.Enemies[idx]
	e.HP -= dmg
	e.HitT = 1.10 // flash duration
	if e.HP > 0 {
		return
	}

	deathPos := e.Pos
	xp := e.XPValue
	kind := e.Kind
	w.removeEnemyAt(idx)
	w.spawnXPOrb(deathPos, xp)
	w.maybeSpawnWeaponDrop(deathPos, kind)
	w.Stats.EnemiesKilled++
}

func sortIdxDesc(idxs []int) {
	for i := 0; i < len(idxs)-1; i++ {
		for j := i + 1; j < len(idxs); j++ {
			if idxs[j] > idxs[i] {
				idxs[i], idxs[j] = idxs[j], idxs[i]
			}
		}
	}
}

func dist2(a, b Vec2) float32 {
	d := a.Sub(b)

	return d.X*d.X + d.Y*d.Y
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func clamp(v, lo, hi float32) float32 {
	return float32(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

func absf(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
