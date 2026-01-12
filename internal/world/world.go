package world

import (
	"fmt"
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"horde-lab/internal/shared/input"
)

func NewWorld(w, h float32) *World {
	cfg := DefaultConfig()
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
	}
	return &World{
		W: w, H: h,
		Cfg: cfg,

		Player:     pl,
		Enemies:    make([]Enemy, 0, 256),
		Orbs:       make([]XPOrb, 0, 256),
		spawnEvery: cfg.BaseSpawnEvery,

		rng: rand.New(rand.NewSource(1)),
	}
}

func (w *World) Reset() {
	// keep constants/config; reset mutable state
	*w = *NewWorld(w.W, w.H)
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
		}

	}
	w.inbox = w.inbox[:0]

	// stop simulating during game over or menu
	if w.GameOver || w.Upgrade.Active || w.Paused {
		return
	}

	if w.LastAttackT > 0 {
		w.LastAttackT -= dt
		if w.LastAttackT < 0 {
			w.LastAttackT = 0
		}
	}

	w.TimeSurvived += dt

	w.updateDifficulty()
	w.updateSpawning(dt)
	w.updateEnemies(dt)
	w.updateCombat(dt)
	w.updateKnockback(dt)
	w.updateContactDamage(dt)
	w.updateXPOrbs(dt)
	w.updateShake(dt)
	w.updateLevelUp()
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

func (w *World) Draw(screen *ebiten.Image) {
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

	// enemies (centered react; replace wit )
	for _, e := range w.Enemies {
		clr := color.RGBA{220, 80, 80, 255}
		if e.HitT > 0 {
			clr = color.RGBA{255, 220, 220, 255}
		}
		vector.FillRect(
			screen,
			camX+e.Pos.X,
			camY+e.Pos.Y,
			e.R, e.R,
			clr,
			false,
		)
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
	}

	// draw player
	pclr := color.RGBA{80, 200, 120, 255}
	if w.Player.HurtTimer > 0 {
		pclr = color.RGBA{200, 240, 200, 255}
	}
	vector.FillCircle(
		screen,
		camX+w.Player.Pos.X,
		camY+w.Player.Pos.Y,
		w.Player.R,
		pclr,
		false,
	)

	// HUD (top-left, screen space)
	hud := fmt.Sprintf(
		"HP: %.0f/%.0f\nLV: %d  XP: %.0f/%.0f\nKills: %d\nEnemies: %d  Orbs: %d\nSpawnEvery: %.2fs\nTime: %.1fs",
		w.Player.HP, w.Player.MaxHP,
		w.Player.Level, w.Player.XP, w.Player.XPToNext,
		w.Stats.EnemiesKilled,
		len(w.Enemies), len(w.Orbs),
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
		o0 := w.Upgrade.Option[0]
		o1 := w.Upgrade.Option[1]
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
