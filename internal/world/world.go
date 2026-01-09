package world

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"horde-lab/internal/shared/input"
)

type MsgInput struct{ Input input.State }

type XPOrb struct {
	Pos   Vec2
	R     float32
	Value float32
}

type World struct {
	W, H float32

	inbox []Msg // TODO: use channel

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
}

type Enemy struct {
	Pos   Vec2
	Speed float32
	R     float32

	// combat
	HP    float32
	MaxHp float32
	HitT  float32 // hit flash timer (seconds)

	TouchDamage float32 // damage when colliding with player
}

func NewWorld(w, h float32) *World {
	pl := Player{
		Pos:   Vec2{X: w / 2, Y: h / 2},
		Speed: 260,
		R:     10,

		AttackCooldown: 0.45,
		AttackRange:    180,
		Damage:         35,

		MaxHP:        100,
		HP:           100,
		HurtCooldown: 0.35,

		Level:    1,
		XP:       0,
		XPToNext: xpTpNext(1),
	}
	return &World{
		W: w, H: h,
		Player:     pl,
		Enemies:    make([]Enemy, 0, 256),
		Orbs:       make([]XPOrb, 0, 256),
		spawnEvery: 0.75,
		rng:        rand.New(rand.NewSource(1)),
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

	w.updateSpawning(dt)
	w.updateEnemies(dt)
	w.updateCombat(dt)
	w.updateContactDamage(dt)
	w.updateXPOrbs(dt)
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
		fmt.Println("pos x: ", w.Player.Pos.X)
		fmt.Println("pos y: ", w.Player.Pos.Y)
		w.Player.Pos.X += dir.X * w.Player.Speed * dt
		w.Player.Pos.Y += dir.Y * w.Player.Speed * dt
		fmt.Println("pos x: ", w.Player.Pos.X)
		fmt.Println("pos y: ", w.Player.Pos.Y)
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
		"HP: %.0f/%.0f\nLV: %d  XP: %.0f/%.0f\nEnemies: %d  Orbs: %d\nTime: %.1fs",
		w.Player.HP, w.Player.MaxHP,
		w.Player.Level, w.Player.XP, w.Player.XPToNext,
		len(w.Enemies), len(w.Orbs),
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
		x := 12
		y := 120
		ebitenutil.DebugPrintAt(screen, "LEVEL UP! Choose an upgrade: ", x, y)
		y += 18

		o0 := w.Upgrade.Option[0]
		o1 := w.Upgrade.Option[1]

		ebitenutil.DebugPrintAt(screen, o0.Title, x, y)
		y += 16
		ebitenutil.DebugPrintAt(screen, "    "+o0.Desc, x, y)
		y += 22

		ebitenutil.DebugPrintAt(screen, o1.Title, x, y)
		y += 16
		ebitenutil.DebugPrintAt(screen, "    "+o1.Desc, x, y)
		y += 22

		ebitenutil.DebugPrintAt(screen, "Press 1 or 2", x, y)
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

func clamp(v, lo, hi float32) float32 {
	return float32(math.Max(float64(lo), math.Min(float64(hi), float64(v))))
}

func (w *World) updateSpawning(dt float32) {
	w.spawnTimer += dt
	for w.spawnTimer >= w.spawnEvery {
		w.spawnTimer -= w.spawnEvery
		w.spawnEnemyNearPlayer()
	}
}

func (w *World) spawnEnemyNearPlayer() {
	// Spawn in a ring around the player, slightly off-screen-ish.
	const spawnRadius float32 = 420

	ang := w.rng.Float32() * 2 * math.Pi
	off := Vec2{
		X: float32(math.Cos(float64(ang))) * spawnRadius,
		Y: float32(math.Sin(float64(ang))) * spawnRadius,
	}

	pos := w.Player.Pos.Add(off)

	// Clamp to world bounds so enemies always exist in-world
	pos.X = clamp(pos.X, 0, w.W)
	pos.Y = clamp(pos.Y, 0, w.H)

	w.Enemies = append(w.Enemies, Enemy{
		Pos:         pos,
		Speed:       220,
		R:           9,
		MaxHp:       75,
		HP:          75,
		TouchDamage: 10,
	})
}

func (w *World) updateEnemies(dt float32) {
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
		toP := p.Sub(e.Pos)
		if toP.X == 0 && toP.Y == 0 {
			continue
		}
		dir := toP.Norm()
		e.Pos = e.Pos.Add(dir.Mul(e.Speed * dt))

		// Clamp
		e.Pos.X = clamp(e.Pos.X, 0, w.W)
		e.Pos.Y = clamp(e.Pos.Y, 0, w.H)
	}
}

func (w *World) updateCombat(dt float32) {

	// cooldown timer
	if w.Player.AttackTimer > 0 {
		w.Player.AttackTimer -= dt
		if w.Player.AttackTimer > 0 {
			return
		}
	}

	// ready to attack: find nearest enemy in range
	idx := w.nearestEnemyInRange(w.Player.Pos, w.Player.AttackRange)

	if idx < 0 {
		return
	}

	// perform attack
	w.Player.AttackTimer = w.Player.AttackCooldown

	// deal Damage
	e := &w.Enemies[idx]
	e.HP -= w.Player.Damage
	e.HitT = 1.10 // flash duration

	w.LastAttackPos = e.Pos
	w.LastAttackT = 0.08

	if e.HP <= 0 {
		deathPos := e.Pos
		w.spawnXPOrb(deathPos, 5)
		w.removeEnemyAt(idx)
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
			w.Player.HurtTimer = w.Player.HurtCooldown

			if w.Player.HP <= 0 {
				w.Player.HP = 0
				w.GameOver = true
			}

			return
		}
	}

}

func (w *World) spawnXPOrb(pos Vec2, value float32) {
	w.Orbs = append(w.Orbs, XPOrb{
		Pos:   pos,
		R:     6,
		Value: value,
	})
}

func (w *World) updateXPOrbs(dt float32) {
	_ = dt // reserved for future motion/magnetism

	p := w.Player.Pos
	pickupR := w.Player.R + 10 // pickup padding

	for i := 0; i < len(w.Orbs); {
		o := w.Orbs[i]
		rr := pickupR + o.R

		if dist2(p, o.Pos) <= rr*rr {
			w.Player.XP += o.Value
			w.removeOrbAt(i)
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
		w.Player.XPToNext = xpTpNext(w.Player.Level)

		// queue one upgrade choice per level
		w.Upgrade.Pending++
		leveled = true

		// v0.1 simple reward: small heal on level up
		w.Player.HP = minf(w.Player.MaxHP, w.Player.HP+15)
		w.Player.MaxHP += 15
	}

	if leveled {
		w.openUpgradeMenuIfNeeded()
	}
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

func dist2(a, b Vec2) float32 {
	d := a.Sub(b)

	return d.X*d.X + d.Y*d.Y
}

func xpTpNext(level int) float32 {
	// v0.1 simple growth curve.
	// Level 1 -> 25,

	base := 25.0
	growth := 1.28

	return float32(base * math.Pow(growth, float64(level-1)))
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
