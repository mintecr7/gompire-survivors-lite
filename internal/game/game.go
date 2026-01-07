package game

import (
	// "fmt"
	"horde-lab/internal/world"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Game struct {
	w *world.World

	// fixed tick
	accum     time.Duration
	last      time.Time
	fixedStep time.Duration
}

func New() *Game {
	return &Game{
		w:         world.NewWorld(2000, 2000), // world size
		last:      time.Now(),
		fixedStep: time.Second / 60,
	}
}

func (g *Game) Update() error {
	now := time.Now()
	frameDt := now.Sub(g.last)
	g.last = now

	// avoid spiral of death on long pauses
	if frameDt > 250*time.Millisecond {
		frameDt = 250 * time.Millisecond
	}

	g.accum += frameDt

	in := ReadInput()

	if ReadRestart() {
		g.w.Enqueue(world.MsgRestart{})
	}

	if ReadPaused() {
		g.w.Enqueue(world.MsgPaused{})
	}
	// if in.Down || in.Left || in.Right || in.Up {

	// 	fmt.Println("input values", in)
	// }

	// fixed-step simulation
	for g.accum >= g.fixedStep {
		g.w.Enqueue(world.MsgInput{Input: in})

		// upgrade selection (edge-triggered)
		if inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.KeyKP1) {
			g.w.Enqueue(world.MsgChooseUpgrade{Choice: 0})
		}

		if inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyKP2) {
			g.w.Enqueue(world.MsgChooseUpgrade{Choice: 1})
		}

		g.w.Tick(float32(g.fixedStep.Seconds()))
		g.accum -= g.fixedStep
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.w.Draw(screen)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return outsideW, outsideH
}
