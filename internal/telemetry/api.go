package telemetry

import "time"

func NewSinkWithEmitter(interval time.Duration, emit func(Batch)) *Sink {
	return newSink(interval, emit)
}
