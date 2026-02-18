package world

import "math/rand"

func (w *World) ensureRNG() {
	if w.rng != nil {
		return
	}
	if w.rngSeed == 0 {
		w.rngSeed = 1
	}
	w.rng = rand.New(rand.NewSource(w.rngSeed))
}

func (w *World) randFloat32() float32 {
	w.ensureRNG()
	w.rngCalls++
	return w.rng.Float32()
}

func (w *World) randIntn(n int) int {
	w.ensureRNG()
	w.rngCalls++
	return w.rng.Intn(n)
}
