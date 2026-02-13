package telemetry

import (
	"horde-lab/internal/commons/logger_config"
	"horde-lab/internal/world"
	"log/slog"
	"sync"
	"time"
)

type Event struct {
	Kind string
	I    int
	F    float32
	At   time.Time
}

type Batch struct {
	Kills  int
	Dmg    float32
	Frames int
	AvgDt  float32
}

type Sink struct {
	In   chan Event
	quit chan struct{}

	closeOnce sync.Once
	interval  time.Duration
	emit      func(Batch)
}

func NewSink() *Sink {
	return newSink(2*time.Second, nil)
}

func newSink(interval time.Duration, emit func(Batch)) *Sink {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if emit == nil {
		emit = emitBatchLog
	}

	s := &Sink{
		In:       make(chan Event, 256),
		quit:     make(chan struct{}),
		interval: interval,
		emit:     emit,
	}
	go s.loop()
	return s
}

// Close stops the sink loop
func (s *Sink) Close() {
	s.closeOnce.Do(func() {
		close(s.quit)
	})
}

func (s *Sink) loop() {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	var (
		kills  int
		dmg    float32
		frames int
		dtSum  float32
	)

	for {
		select {
		case <-s.quit:
			return

		case ev, ok := <-s.In:
			if !ok {
				// input channel closed by producer => stop the loop
				return
			}
			switch ev.Kind {
			case "kill":
				kills += ev.I
			case "damage":
				dmg += ev.F
			case "frame":
				frames++
				dtSum += ev.F
			}

		case <-ticker.C:
			var avgDt float32
			if frames > 0 {
				avgDt = dtSum / float32(frames)
			}

			s.emit(Batch{
				Kills:  kills,
				Dmg:    dmg,
				Frames: frames,
				AvgDt:  avgDt,
			})

			// reset batch
			kills = 0
			dmg = 0
			frames = 0
			dtSum = 0
		}
	}
}

func emitBatchLog(b Batch) {
	logger_config.Logger.Info(
		"telemetry batch",
		slog.Int("kills", b.Kills),
		slog.Float64("dmg", float64(b.Dmg)),
		slog.Int("frames", b.Frames),
		slog.Float64("avg_dt_s", float64(b.AvgDt)),
	)
}

// Helper to emit snapshot metrics if you prefer
func EmitWorldSnapshot(ch chan<- Event, w *world.World) {
	select {
	case ch <- Event{Kind: "kill", I: w.Stats.EnemiesKilled, At: time.Now()}:
	default:
	}
}
