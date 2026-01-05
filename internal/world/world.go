package world

import (
	"fmt"
	"horde-lab/internal/shared/input"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Msg interface{ isMsg() }

type MsgInput struct{ Input input.State }

func (MsgInput) isMsg() {}

type World struct {
	W, H float32

	inbox []Msg

	Player Player

	Enemies []Enemy

	// spawning
	spawnTimer float32
	spawnEvery float32
	rng        *rand.Rand
}

type Player struct {
	Pos   Vec2
	Speed float32
}

type Enemy struct {
	Pos   Vec2
	Speed float32
	R     float32
}

func NewWorld(w, h float32) *World {
	return &World{
		W: w, H: h,
		Player: Player{
			Pos:   Vec2{X: w / 2, Y: h / 2},
			Speed: 260,
		},
		Enemies:    make([]Enemy, 0, 256),
		spawnEvery: 0.75,
		rng:        rand.New(rand.NewSource(1)),
	}
}

func (w *World) Enqueue(m Msg) {
	w.inbox = append(w.inbox, m)
}

func (w *World) Tick(dt float32) {
	// process inbox (v0.1: only input)
	for _, m := range w.inbox {
		switch msg := m.(type) {
		case MsgInput:
			// input is from internal/game; keep loose typing for now
			w.applyInput(dt, msg.Input)
		}
	}
	w.updateSpawning(dt)
	w.updateEnemies(dt)

	w.inbox = w.inbox[:0]
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

	for _, e := range w.Enemies {
		vector.FillRect(
			screen,
			camX+e.Pos.X,
			camY+e.Pos.Y,
			e.R, e.R,
			color.RGBA{220, 80, 80, 255},
			false,
		)
	}

	// draw player
	const r float32 = 10
	vector.FillCircle(
		screen,
		camX+w.Player.Pos.X-r,
		camY+w.Player.Pos.Y-r,
		r,
		color.RGBA{80, 200, 120, 255},
		false,
	)
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
	// Tune radius based on your window size; this is a simple start.
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
		Pos:   pos,
		Speed: 120,
		R:     9,
	})
}

func (w *World) updateEnemies(dt float32) {
	p := w.Player.Pos
	for i := range w.Enemies {
		e := &w.Enemies[i]
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
