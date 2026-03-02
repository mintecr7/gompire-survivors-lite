package world

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"horde-lab/internal/jobs"

	"horde-lab/internal/shared/input"
)

const (
	EnemyNormal EnemyKind = iota
	EnemyRunner
	EnemyTank
)

type AssetProvider interface {
	Get(key string) *ebiten.Image
}

func NewWorld(w, h float32) *World {
	cfg := DefaultConfig()
	const seed int64 = 1
	pl := Player{
		Pos:   Vec2{X: w / 2, Y: h / 2},
		Speed: cfg.PlayerSpeed,
		R:     cfg.PlayerRadius,

		AttackCooldown: cfg.PlayerAttackCooldown,
		AttackRange:    cfg.PlayerAttackRange,
		Damage:         cfg.PlayerDamage,

		MaxHP:        cfg.PlayerMaxHP,
		HP:           cfg.PlayerMaxHP,
		HurtCooldown: cfg.PlayerHurtCooldown,

		Level:    1,
		XP:       0,
		XPToNext: cfg.XPToNext(1),
		XPMagnet: 10,
		Weapon:   WeaponWhip,
	}
	return &World{
		W: w, H: h,
		Cfg: cfg,

		Player:     pl,
		Enemies:    make([]Enemy, 0, 256),
		Orbs:       make([]XPOrb, 0, 256),
		Drops:      make([]WeaponDrop, 0, 32),
		Shots:      make([]EnemyProjectile, 0, 128),
		spawnEvery: cfg.BaseSpawnEvery,

		rng:      rand.New(rand.NewSource(seed)),
		rngSeed:  seed,
		rngCalls: 0,

		aiPool:            newAIPool(),
		aiPendingRequests: make(map[uint64]jobs.IntentRequest, 8),
		aiReadyResults:    make(map[uint64]jobs.IntentResult, 8),
	}
}

func (w *World) Reset() {
	// keep constants/config; reset mutable state
	oldPool := w.aiPool
	*w = *NewWorld(w.W, w.H)
	if oldPool != nil {
		oldPool.Close()
	}
}

func (w *World) Close() {
	if w.aiPool != nil {
		w.aiPool.Close()
		w.aiPool = nil
	}
}

func (w *World) Enqueue(m Msg) {
	w.inbox = append(w.inbox, m)
}

func (w *World) Tick(dt float32) {
	// Allow input processing even if game is over (e.g., restart, game options/setting for later)
	for _, m := range w.inbox {
		switch msg := m.(type) {
		case MsgInput:
			// Prevent movement during pause, game over, or upgrade selection
			if !w.GameOver && !w.Upgrade.Active && !w.Paused {

				w.applyInput(dt, msg.Input)
			}
		case MsgChooseUpgrade:
			if !w.GameOver {
				w.applyUpGradeChoice(msg.Choice)
			}
		case MsgRestart:
			if w.GameOver || w.Paused {
				w.Reset()
			}
		case MsgTogglePause:
			if !w.GameOver && !w.Upgrade.Active {
				w.Paused = !w.Paused
			}
		case MsgSaveSnapshot:
			err := w.SaveSnapshot(msg.Path)
			if msg.Reply != nil {
				select {
				case msg.Reply <- err:
				default:
				}
			}
		case MsgLoadSnapshot:
			err := w.LoadSnapshot(msg.Path)
			if msg.Reply != nil {
				select {
				case msg.Reply <- err:
				default:
				}
			}
		}

	}
	w.inbox = w.inbox[:0]
	w.drainAIResults()

	// stop simulating during game over or menu
	if w.GameOver || w.Upgrade.Active || w.Paused {
		return
	}

	w.aiTick++
	intents := w.consumeAIIntentsForTick(w.aiTick - 1)

	if w.LastAttackT > 0 {
		w.LastAttackT -= dt
		if w.LastAttackT < 0 {
			w.LastAttackT = 0
		}
	}

	w.TimeSurvived += dt

	w.updateDifficulty()
	w.updateSpawning(dt)
	w.updateEnemies(dt, intents)
	w.updateCombat(dt)
	w.updateRunnerRangedShots(dt)
	w.updateKnockback(dt)
	w.updateContactDamage(dt)
	w.updateEnemyProjectiles(dt)
	w.updateXPOrbs(dt)
	w.updateWeaponDrops()
	w.updateShake(dt)
	w.updateLevelUp()
	w.submitAIJob(w.aiTick)
}

func (w *World) applyInput(dt float32, in input.State) {

	var dir Vec2
	if in.Up {
		dir.Y -= 1
	}
	if in.Down {
		dir.Y += 1
	}
	if in.Left {
		dir.X -= 1
	}
	if in.Right {
		dir.X += 1
	}

	if dir.X != 0 || dir.Y != 0 {
		dir = dir.Norm()
		w.Player.Pos.X += dir.X * w.Player.Speed * dt
		w.Player.Pos.Y += dir.Y * w.Player.Speed * dt
	}

	// clamp to bounds
	w.Player.Pos.X = clamp(w.Player.Pos.X, 0, w.W)
	w.Player.Pos.Y = clamp(w.Player.Pos.Y, 0, w.H)
}

func (w *World) Draw(screen *ebiten.Image, assets AssetProvider) {
	screen.Fill(color.RGBA{15, 15, 18, 255})

	// camera centered on player
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	camX := float32(sw)/2 - w.Player.Pos.X
	camY := float32(sh)/2 - w.Player.Pos.Y

	// offset camera for damage shake

	camX += w.ShakeOff.X
	camY += w.ShakeOff.Y

	// world background``
	vector.FillRect(
		screen,
		camX, camY,
		w.W, w.H,
		color.RGBA{30, 30, 36, 255},
		false, // anti-alias
	)

	// XP orbs
	for _, o := range w.Orbs {
		vector.FillCircle(
			screen,
			camX+o.Pos.X,
			camY+o.Pos.Y,
			o.R,
			color.RGBA{240, 210, 80, 255},
			false,
		)
	}

	// weapon drops
	for _, d := range w.Drops {
		dx := camX + d.Pos.X
		dy := camY + d.Pos.Y
		switch d.Kind {
		case WeaponSpear:
			vector.StrokeLine(screen, dx-d.R, dy+d.R, dx+d.R, dy-d.R, 2, color.RGBA{110, 220, 255, 255}, false)
		case WeaponNova:
			vector.FillCircle(screen, dx, dy, d.R, color.RGBA{230, 90, 220, 220}, false)
			vector.StrokeCircle(screen, dx, dy, d.R+2, 1, color.RGBA{255, 160, 250, 255}, false)
		case WeaponFang:
			vector.FillRect(screen, dx-d.R*0.5, dy-d.R, d.R, d.R*2, color.RGBA{255, 120, 120, 255}, false)
		default:
			vector.FillRect(screen, dx-d.R, dy-d.R*0.35, d.R*2, d.R*0.7, color.RGBA{255, 220, 120, 255}, false)
		}
	}

	// enemy projectiles
	for _, s := range w.Shots {
		sx := camX + s.Pos.X
		sy := camY + s.Pos.Y
		vector.FillCircle(screen, sx, sy, s.R, color.RGBA{255, 95, 95, 230}, false)
		vector.StrokeCircle(screen, sx, sy, s.R+1, 1, color.RGBA{255, 200, 200, 255}, false)
	}

	// Enemy rendering with visual variety
	for _, e := range w.Enemies {
		ex := camX + e.Pos.X
		ey := camY + e.Pos.Y

		switch e.Kind {
		case EnemyRunner:
			// Fast, elongated diamond-like enemy (stretched horizontally)
			clr := color.RGBA{240, 170, 60, 255}
			if e.HitT > 0 {
				clr = color.RGBA{255, 255, 255, 255}
			}

			// Draw as two overlapping rectangles to form diamond
			// Horizontal part
			vector.FillRect(
				screen,
				ex-e.R*1.2, ey-e.R*0.4,
				e.R*2.4, e.R*0.8,
				clr,
				false,
			)

			// Vertical part (smaller)
			vector.FillRect(
				screen,
				ex-e.R*0.4, ey-e.R*0.8,
				e.R*0.8, e.R*1.6,
				clr,
				false,
			)

			// Bright center core
			vector.FillRect(
				screen,
				ex-e.R*0.25, ey-e.R*0.25,
				e.R*0.5, e.R*0.5,
				color.RGBA{255, 220, 120, 255},
				false,
			)

		case EnemyTank:
			// Large, beefy tank with armor plating
			baseClr := color.RGBA{170, 110, 240, 255}
			if e.HitT > 0 {
				baseClr = color.RGBA{255, 255, 255, 255}
			}

			// Large square body
			vector.FillRect(
				screen,
				ex-e.R, ey-e.R,
				e.R*2, e.R*2,
				baseClr,
				false,
			)

			// Armor plates (dark lines)
			darkClr := color.RGBA{120, 70, 180, 255}

			// Horizontal armor lines
			vector.FillRect(
				screen,
				ex-e.R*0.9, ey-e.R*0.4,
				e.R*1.8, e.R*0.2,
				darkClr,
				false,
			)
			vector.FillRect(
				screen,
				ex-e.R*0.9, ey+e.R*0.2,
				e.R*1.8, e.R*0.2,
				darkClr,
				false,
			)

			// Vertical center line
			vector.FillRect(
				screen,
				ex-e.R*0.1, ey-e.R*0.9,
				e.R*0.2, e.R*1.8,
				darkClr,
				false,
			)

			// Core/weak point
			vector.FillRect(
				screen,
				ex-e.R*0.3, ey-e.R*0.3,
				e.R*0.6, e.R*0.6,
				color.RGBA{220, 160, 255, 255},
				false,
			)
		default: // EnemyNormal
			clr := color.RGBA{220, 80, 80, 255}
			if e.HitT > 0 {
				clr = color.RGBA{255, 180, 180, 255}
			}

			vector.FillCircle(
				screen,
				ex, ey,
				e.R,
				clr,
				false,
			)

			// small "eye"
			eyeR := e.R * 0.35
			vector.FillCircle(
				screen,
				ex, ey,
				eyeR,
				color.RGBA{150, 40, 40, 255},
				false,
			)
		}
	}
	// attack line (fade normalized)
	if w.LastAttackT > 0 {
		const lastAttackMax float32 = 0.08
		t := w.LastAttackT / lastAttackMax

		if t < 0 {
			t = 0
		}

		if t > 1 {
			t = 1
		}

		alpha := uint8(255 * t)
		switch w.LastAttackWeapon {
		case WeaponNova:
			vector.StrokeCircle(
				screen,
				camX+w.Player.Pos.X,
				camY+w.Player.Pos.Y,
				w.LastAttackRadius,
				2,
				color.RGBA{255, 130, 230, alpha},
				false,
			)
		case WeaponSpear:
			vector.StrokeLine(
				screen,
				camX+w.Player.Pos.X,
				camY+w.Player.Pos.Y,
				camX+w.LastAttackPos.X,
				camY+w.LastAttackPos.Y,
				3,
				color.RGBA{120, 230, 255, alpha},
				false,
			)
			vector.FillCircle(screen, camX+w.LastAttackPos.X, camY+w.LastAttackPos.Y, 3, color.RGBA{200, 245, 255, alpha}, false)
		case WeaponFang:
			midX := (w.Player.Pos.X + w.LastAttackPos.X) * 0.5
			midY := (w.Player.Pos.Y + w.LastAttackPos.Y) * 0.5
			vector.StrokeLine(screen, camX+w.Player.Pos.X, camY+w.Player.Pos.Y, camX+midX, camY+midY, 2, color.RGBA{255, 110, 110, alpha}, false)
			vector.StrokeLine(screen, camX+midX, camY+midY, camX+w.LastAttackPos.X, camY+w.LastAttackPos.Y, 2, color.RGBA{255, 170, 170, alpha}, false)
		default: // whip
			vector.StrokeLine(
				screen,
				camX+w.Player.Pos.X,
				camY+w.Player.Pos.Y,
				camX+w.LastAttackPos.X,
				camY+w.LastAttackPos.Y,
				2, // line width
				color.RGBA{255, 255, 100, alpha},
				false,
			)
			vector.StrokeLine(
				screen,
				camX+w.Player.Pos.X,
				camY+w.Player.Pos.Y,
				camX+(w.Player.Pos.X*0.35+w.LastAttackPos.X*0.65),
				camY+(w.Player.Pos.Y*0.35+w.LastAttackPos.Y*0.65),
				1,
				color.RGBA{255, 220, 140, alpha},
				false,
			)
		}
	}

	// draw player
	px := camX + w.Player.Pos.X
	py := camY + w.Player.Pos.Y

	if playerImg := assets.Get("player"); playerImg != nil {
		b := playerImg.Bounds()
		iw, ih := b.Dx(), b.Dy()

		if iw > 0 && ih > 0 {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(-float64(iw)/2, -float64(ih)/2)
			op.GeoM.Scale(
				float64((w.Player.R*2)/float32(iw)),
				float64((w.Player.R*2)/float32(ih)),
			)
			op.GeoM.Translate(float64(px), float64(py))

			if w.Player.HurtTimer > 0 {
				op.ColorScale.Scale(1.25, 1.25, 1.25, 1.0)
			}
			screen.DrawImage(playerImg, op)
		}
	} else {
		pclr := color.RGBA{80, 200, 120, 255}
		if w.Player.HurtTimer > 0 {
			pclr = color.RGBA{200, 240, 200, 255}
		}
		vector.FillCircle(
			screen,
			px,
			py,
			w.Player.R,
			pclr,
			false,
		)
	}

	// HUD (top-left, screen space)
	hud := fmt.Sprintf(
		"HP: %.0f/%.0f\nLV: %d  XP: %.0f/%.0f\nWeapon: %s\nKills: %d\nEnemies: %d  Orbs: %d  Drops: %d\nSpawnEvery: %.2fs\nTime: %.1fs",
		w.Player.HP, w.Player.MaxHP,
		w.Player.Level, w.Player.XP, w.Player.XPToNext,
		weaponDef(w.Player.Weapon).Name,
		w.Stats.EnemiesKilled,
		len(w.Enemies), len(w.Orbs), len(w.Drops),
		w.spawnEvery,
		w.TimeSurvived,
	)

	ebitenutil.DebugPrintAt(screen, hud, 8, 8)

	// ---- Modal overlays (priority: GameOver > Upgrade > Paused) ----

	// Game over overlay
	if w.GameOver {
		vector.FillRect(
			screen,
			0, 0,
			float32(sw), float32(sh),
			color.RGBA{0, 0, 0, 180},
			false,
		)
		ebitenutil.DebugPrintAt(screen, "GAME OVER", 8, 90)
		ebitenutil.DebugPrintAt(screen, "Press R to restart", 8, 110)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Time Survived: %.1fs", w.TimeSurvived), 8, 130)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Level Reached: %d", w.Player.Level), 8, 150)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Kills: %d", w.Stats.EnemiesKilled), 8, 170)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Enemies Spawned: %d", w.Stats.EnemiesSpawned), 8, 190)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Damage Taken: %.0f", w.Stats.DamageTaken), 8, 210)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("XP Collected: %.0f", w.Stats.XPCollected), 8, 230)

		return
	}

	// Upgrade menu overlay
	if w.Upgrade.Active {

		vector.FillRect(
			screen,
			0, 0,
			float32(sw), float32(sh),
			color.RGBA{0, 0, 0, 180},
			false,
		)

		// menu text
		ebitenutil.DebugPrintAt(screen, "LEVEL UP! Choose an upgrade: ", 12, 120)
		o0 := w.Upgrade.Options[0]
		o1 := w.Upgrade.Options[1]
		ebitenutil.DebugPrintAt(screen, o0.Title, 12, 138)
		ebitenutil.DebugPrintAt(screen, "    "+o0.Desc, 12, 152)
		ebitenutil.DebugPrintAt(screen, o1.Title, 12, 174)
		ebitenutil.DebugPrintAt(screen, "    "+o1.Desc, 12, 190)
		ebitenutil.DebugPrintAt(screen, "Press 1 or 2", 12, 212)
	}

	// Pause overlay
	if w.Paused {
		vector.FillRect(
			screen,
			0, 0,
			float32(sw), float32(sh),
			color.RGBA{0, 0, 0, 140},
			false,
		)
		ebitenutil.DebugPrintAt(screen, "PAUSED", 8, 9)
		ebitenutil.DebugPrintAt(screen, "Press the space bar to resume", 8, 110)
		ebitenutil.DebugPrintAt(screen, "Press R to restart", 8, 130)
	}
}
