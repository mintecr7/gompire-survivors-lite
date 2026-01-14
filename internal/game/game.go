package game

import (
	// "fmt"
	"horde-lab/internal/assets"
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

	// asset loader
	loader *assets.Loader
	assets *AssetManager
}

func New() *Game {
	g := &Game{
		w:         world.NewWorld(2000, 2000), // world size
		last:      time.Now(),
		fixedStep: time.Second / 60,
	}
	g.loader = assets.NewLoader()
	g.assets = NewAssetManager(g.loader)

	// schedule loads early
	g.assets.Request("player", "assets/player.webp")
	return g
}

func (g *Game) Update() error {
	now := time.Now()
	g.assets.Poll()

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
		g.w.Enqueue(world.MsgTogglePause{})
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
	g.w.Draw(screen, g.assets)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return outsideW, outsideH
}

func (g *Game) Close() {
	g.loader.Close()
}
