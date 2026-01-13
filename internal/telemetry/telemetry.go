package telemetry

import (
	"horde-lab/internal/commons/logger_config"
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

func (s *Sink) loop() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var kills int
	var dmg float32
	var frames int
	var dtSum float32

	for {
		select {
		case <-s.quit:
			return

		case ev := <-s.In:
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
			avgDt := float32(0)
			if frames > 0 {
				avgDt = dtSum / float32(frames)
			}
			logger_config.Logger.Info(
				"[telemetry] kills=%d dmg=%.0f frames=%d avgDt=%.4fs",
				kills, dmg, frames, avgDt,
			)
			// reset batch
			kills = 0
			dmg = 0
			frames = 0
			dtSum = 0
		}
	}
}
