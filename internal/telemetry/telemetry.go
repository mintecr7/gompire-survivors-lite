package telemetry

import (
	"horde-lab/internal/commons/logger_config"
	"log/slog"
	"time"
)

type Event struct {
	Kind string
	I    int
	F    float32
	At   time.Time
}

type Sink struct {
	In   chan Event
	quit chan struct{}
}

func NewSink() *Sink {
	s := &Sink{
		In:   make(chan Event, 256),
		quit: make(chan struct{}),
	}
	go s.loop()
	return s
}

// Close stops the sink loop
func (s *Sink) Close() {
	// safe even if called multiple times? only if you guard it; keeping simple:
	close(s.quit)
}

func (s *Sink) loop() {
	ticker := time.NewTicker(2 * time.Second)
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

			// Structured logging (slog-style)
			logger_config.Logger.Info(
				"telemetry batch",
				slog.Int("kills", kills),
				slog.Float64("dmg", float64(dmg)),
				slog.Int("frames", frames),
				slog.Float64("avg_dt_s", float64(avgDt)),
			)

			// reset batch
			kills = 0
			dmg = 0
			frames = 0
			dtSum = 0
		}
	}
}
