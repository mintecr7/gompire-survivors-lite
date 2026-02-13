package game

import (
	// "fmt"
	"horde-lab/internal/assets"
	"horde-lab/internal/telemetry"
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

	// telemetry sink
	telemetry *telemetry.Sink

	// cumulative stat baselines (for delta events)
	lastKills  int
	lastDamage float32
}

func New() *Game {
	g := &Game{
		w:         world.NewWorld(2000, 2000), // world size
		last:      time.Now(),
		fixedStep: time.Second / 60,
	}
	g.loader = assets.NewLoader()
	g.assets = NewAssetManager(g.loader)
	g.telemetry = telemetry.NewSink()

	// schedule loads early
	g.assets.Request("player", "player.webp")
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
	g.sendTelemetry(telemetry.Event{
		Kind: "frame",
		F:    float32(frameDt.Seconds()),
		At:   now,
	})

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
	g.emitWorldDeltas(now)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.w.Draw(screen, g.assets)
}

func (g *Game) Layout(outsideW, outsideH int) (int, int) {
	return outsideW, outsideH
}

func (g *Game) Close() {
	if g.loader != nil {
		g.loader.Close()
		g.loader = nil
	}
	if g.telemetry != nil {
		g.telemetry.Close()
		g.telemetry = nil
	}
	if g.w != nil {
		g.w.Close()
		g.w = nil
	}
}

func (g *Game) emitWorldDeltas(at time.Time) {
	stats := g.w.Stats

	if stats.EnemiesKilled < g.lastKills {
		g.lastKills = stats.EnemiesKilled
	} else {
		deltaKills := stats.EnemiesKilled - g.lastKills
		if deltaKills > 0 {
			g.sendTelemetry(telemetry.Event{
				Kind: "kill",
				I:    deltaKills,
				At:   at,
			})
			g.lastKills = stats.EnemiesKilled
		}
	}

	if stats.DamageTaken < g.lastDamage {
		g.lastDamage = stats.DamageTaken
	} else {
		deltaDamage := stats.DamageTaken - g.lastDamage
		if deltaDamage > 0 {
			g.sendTelemetry(telemetry.Event{
				Kind: "damage",
				F:    deltaDamage,
				At:   at,
			})
			g.lastDamage = stats.DamageTaken
		}
	}
}

func (g *Game) sendTelemetry(ev telemetry.Event) {
	if g.telemetry == nil {
		return
	}

	select {
	case g.telemetry.In <- ev:
	default:
		// Drop on backpressure to avoid stalling the fixed-step loop.
	}
}
