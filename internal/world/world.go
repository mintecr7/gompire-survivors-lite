package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Msg interface{ isMsg() }

type MsgInput struct{ Input any }

func (MsgInput) isMsg() {}

type World struct {
	W, H float32

	// simple in-process inbox for v0.1
	inbox []Msg

	Player Player
}

type Player struct {
	Pos   Vec2
	Speed float32
}

func NewWorld(w, h float32) *World {
	return &World{
		W: w, H: h,
		Player: Player{
			Pos:   Vec2{X: w / 2, Y: h / 2},
			Speed: 260,
		},
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
			// we'll replace `any` with a proper type once we finalize packages
			w.applyInput(dt, msg.Input)
		}
	}
	w.inbox = w.inbox[:0]
}

func (w *World) applyInput(dt float32, input any) {
	// avoid import cycle by not referencing game.InputState yet.
	// We'll tighten types after v0.1 wiring is stable.
	type inputState struct{ Up, Down, Left, Right bool }
	in, ok := input.(inputState)
	if !ok {
		// try the concrete type from game package via struct shape is not possible directly
		// so in v0.1, we’ll keep this shim and later move InputState to a shared package.
		return
	}

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

	// simple camera centered on player
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	camX := float32(sw)/2 - w.Player.Pos.X
	camY := float32(sh)/2 - w.Player.Pos.Y

	// draw world bounds
	vector.FillRect(
		screen,
		camX, camY,
		w.W, w.H,
		color.RGBA{30, 30, 36, 255},
		false, // anti-alias
	)

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
