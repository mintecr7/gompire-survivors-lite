package game

import (
	"time"

	"horde-lab/internal/world"

	"github.com/hajimehoshi/ebiten/v2"
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

	// fixed-step simulation
	for g.accum >= g.fixedStep {
		g.w.Enqueue(world.MsgInput{Input: in})
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
