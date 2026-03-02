package game

import (
	// "fmt"
	"fmt"
	"horde-lab/internal/assets"
	"horde-lab/internal/telemetry"
	"horde-lab/internal/world"
	"log"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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

	profilePath   string
	saveGamePath  string
	highscorePath string
	profile       PlayerProfile
	highscores    HighscoreFile
	gameOverSaved bool
}

func New() *Game {
	g := &Game{
		w:             world.NewWorld(2000, 2000), // world size
		last:          time.Now(),
		fixedStep:     time.Second / 60,
		snapshotPath:  ".dist/snapshot.json",
		replayPath:    ".dist/replay.json",
		profilePath:   ".dist/player_profile.json",
		saveGamePath:  ".dist/savegame.json",
		highscorePath: ".dist/highscores.json",
	}

	if p, err := loadProfile(g.profilePath); err == nil {
		g.profile = p
	} else {
		g.profile = defaultProfile()
		if err := saveProfile(g.profilePath, g.profile); err != nil {
			log.Printf("init profile: %v", err)
		}
	}
	if hs, err := loadHighscores(g.highscorePath); err == nil {
		g.highscores = hs
	} else {
		g.highscores = HighscoreFile{Version: highscoreVersion, Entries: make([]HighscoreEntry, 0, 16)}
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
	if ReadStopAndSaveGame() && !g.replayMode {
		if err := g.saveCurrentGame(); err != nil {
			log.Printf("save game: %v", err)
		} else {
			log.Printf("game saved to %s", g.saveGamePath)
		}
		if !g.w.Paused && !g.w.GameOver {
			g.w.Enqueue(world.MsgTogglePause{})
		}
	}
	if ReadLoadSavedGame() && !g.replayMode {
		if err := g.loadSavedGame(); err != nil {
			log.Printf("load game: %v", err)
		} else {
			log.Printf("loaded saved game from %s", g.saveGamePath)
		}
	}
	if ReadStartReplay() && !g.replayMode {
		if err := g.startReplayFromFile(); err != nil {
			log.Printf("start replay: %v", err)
		}
	}
	if ReadContinuePaused() && g.w.Paused && !g.w.GameOver {
		g.w.Enqueue(world.MsgTogglePause{})
	}
	if ReadCycleCharacter() && !g.replayMode {
		g.cycleCharacter()
	}
	if ReadCycleCustomization() && !g.replayMode {
		g.cycleCustomization()
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
	g.captureHighscoreOnGameOver()
	g.pollPersistenceReplies()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.w.Draw(screen, g.assets)
	best := "-"
	if len(g.highscores.Entries) > 0 {
		top := g.highscores.Entries[0]
		best = fmt.Sprintf("%d (%s)", top.Score, top.Name)
	}
	status := fmt.Sprintf(
		"Player: %s  Character: %s  Style: %s\nBest Score: %s\nF1: cycle character  F2: cycle style  F7: stop+save  F8: load save  C: continue paused",
		g.profile.Name,
		g.profile.Character,
		g.profile.Customization,
		best,
	)
	ebitenutil.DebugPrintAt(screen, status, 8, screen.Bounds().Dy()-40)
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

func (g *Game) saveCurrentGame() error {
	sg := SaveGame{
		Version:  saveGameVersion,
		SavedAt:  time.Now(),
		Profile:  g.profile,
		Snapshot: g.w.BuildSnapshot(),
	}
	return saveSaveGame(g.saveGamePath, sg)
}

func (g *Game) loadSavedGame() error {
	sg, err := loadSaveGame(g.saveGamePath)
	if err != nil {
		return err
	}
	if err := g.w.ApplySnapshot(sg.Snapshot); err != nil {
		return err
	}
	g.profile = sg.Profile
	g.gameOverSaved = g.w.GameOver
	g.resetReplayRecording()
	return nil
}

func (g *Game) cycleCharacter() {
	idx := slices.Index(characterChoices, g.profile.Character)
	if idx < 0 {
		idx = 0
	}
	g.profile.Character = characterChoices[(idx+1)%len(characterChoices)]
	if err := saveProfile(g.profilePath, g.profile); err != nil {
		log.Printf("save profile: %v", err)
	}
}

func (g *Game) cycleCustomization() {
	idx := slices.Index(customizationChoices, g.profile.Customization)
	if idx < 0 {
		idx = 0
	}
	g.profile.Customization = customizationChoices[(idx+1)%len(customizationChoices)]
	if err := saveProfile(g.profilePath, g.profile); err != nil {
		log.Printf("save profile: %v", err)
	}
}

func (g *Game) captureHighscoreOnGameOver() {
	if !g.w.GameOver {
		g.gameOverSaved = false
		return
	}
	if g.gameOverSaved {
		return
	}

	s := g.w.BuildSnapshot()
	entry := HighscoreEntry{
		At:            time.Now(),
		Name:          g.profile.Name,
		Character:     g.profile.Character,
		Customization: g.profile.Customization,
		Kills:         s.Stats.EnemiesKilled,
		Level:         s.Player.Level,
		TimeSurvived:  s.TimeSurvived,
		Score:         calcScore(s),
	}
	g.highscores.Entries = append(g.highscores.Entries, entry)
	sortHighscores(g.highscores.Entries)
	if len(g.highscores.Entries) > 20 {
		g.highscores.Entries = g.highscores.Entries[:20]
	}
	if err := saveHighscores(g.highscorePath, g.highscores); err != nil {
		log.Printf("save highscores: %v", err)
	}
	g.gameOverSaved = true
}
