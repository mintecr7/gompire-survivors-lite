package game

import (
	// "fmt"
	"horde-lab/internal/assets"
	"horde-lab/internal/telemetry"
	"horde-lab/internal/world"
	"log"
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

	snapshotPath string
	saveReply    chan error
	loadReply    chan error

	replayPath string
	replay     world.ReplayFile
	replayTick uint64

	replayMode     bool
	replayFrameIdx int
}

func New() *Game {
	g := &Game{
		w:            world.NewWorld(2000, 2000), // world size
		last:         time.Now(),
		fixedStep:    time.Second / 60,
		snapshotPath: ".dist/snapshot.json",
		replayPath:   ".dist/replay.json",
	}
	g.resetReplayRecording()
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
	restartPressed := ReadRestart()
	pausePressed := ReadPaused()
	choose0 := inpututil.IsKeyJustPressed(ebiten.Key1) || inpututil.IsKeyJustPressed(ebiten.KeyKP1)
	choose1 := inpututil.IsKeyJustPressed(ebiten.Key2) || inpututil.IsKeyJustPressed(ebiten.KeyKP2)

	if ReadSaveSnapshot() && g.saveReply == nil {
		g.saveReply = make(chan error, 1)
		g.w.Enqueue(world.MsgSaveSnapshot{
			Path:  g.snapshotPath,
			Reply: g.saveReply,
		})
	}
	if ReadLoadSnapshot() && g.loadReply == nil {
		g.loadReply = make(chan error, 1)
		g.w.Enqueue(world.MsgLoadSnapshot{
			Path:  g.snapshotPath,
			Reply: g.loadReply,
		})
	}
	if ReadSaveReplay() && !g.replayMode {
		if err := world.SaveReplayFile(g.replayPath, g.replay); err != nil {
			log.Printf("save replay: %v", err)
		}
	}
	if ReadStartReplay() && !g.replayMode {
		if err := g.startReplayFromFile(); err != nil {
			log.Printf("start replay: %v", err)
		}
	}
	// if in.Down || in.Left || in.Right || in.Up {

	// 	fmt.Println("input values", in)
	// }

	// fixed-step simulation
	for g.accum >= g.fixedStep {
		if g.replayMode {
			if g.replayFrameIdx >= len(g.replay.Frames) {
				log.Printf("replay complete: frames=%d", len(g.replay.Frames))
				g.replayMode = false
				g.replayFrameIdx = 0
				g.resetReplayRecording()
				break
			}

			frame := g.replay.Frames[g.replayFrameIdx]
			g.enqueueReplayFrame(frame)
			g.replayFrameIdx++
		} else {
			frame := world.ReplayFrame{
				Tick:        g.replayTick,
				Input:       in,
				Choose:      -1,
				TogglePause: pausePressed,
				Restart:     restartPressed,
			}
			if choose0 {
				frame.Choose = 0
			} else if choose1 {
				frame.Choose = 1
			}
			g.enqueueReplayFrame(frame)
			g.replay.Frames = append(g.replay.Frames, frame)
			g.replayTick++

			restartPressed = false
			pausePressed = false
			choose0 = false
			choose1 = false
		}

		g.w.Tick(float32(g.fixedStep.Seconds()))
		g.accum -= g.fixedStep
	}
	g.emitWorldDeltas(now)
	g.pollPersistenceReplies()

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

func (g *Game) pollPersistenceReplies() {
	if g.saveReply != nil {
		select {
		case err := <-g.saveReply:
			if err != nil {
				log.Printf("save snapshot: %v", err)
			}
			g.saveReply = nil
		default:
		}
	}
	if g.loadReply != nil {
		select {
		case err := <-g.loadReply:
			if err != nil {
				log.Printf("load snapshot: %v", err)
			} else if !g.replayMode {
				g.resetReplayRecording()
			}
			g.loadReply = nil
		default:
		}
	}
}

func (g *Game) enqueueReplayFrame(frame world.ReplayFrame) {
	g.w.Enqueue(world.MsgInput{Input: frame.Input})
	if frame.Restart {
		g.w.Enqueue(world.MsgRestart{})
	}
	if frame.TogglePause {
		g.w.Enqueue(world.MsgTogglePause{})
	}
	if frame.Choose == 0 || frame.Choose == 1 {
		g.w.Enqueue(world.MsgChooseUpgrade{Choice: frame.Choose})
	}
}

func (g *Game) resetReplayRecording() {
	initial := g.w.BuildSnapshot()
	h, err := world.BuildReplayHeader(initial, float32(g.fixedStep.Seconds()))
	if err != nil {
		log.Printf("build replay header: %v", err)
		h = world.ReplayHeader{
			Version:          world.ReplayVersion,
			FixedStepSeconds: float32(g.fixedStep.Seconds()),
			Seed:             initial.RNGSeed,
		}
	}
	g.replay = world.ReplayFile{
		Header:  h,
		Initial: initial,
		Frames:  make([]world.ReplayFrame, 0, 4096),
	}
	g.replayTick = 0
}

func (g *Game) startReplayFromFile() error {
	rep, err := world.LoadReplayFile(g.replayPath)
	if err != nil {
		return err
	}
	if err := g.w.ApplySnapshot(rep.Initial); err != nil {
		return err
	}

	g.replay = rep
	g.replayMode = true
	g.replayFrameIdx = 0
	g.replayTick = 0
	return nil
}
