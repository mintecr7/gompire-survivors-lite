package world

func buildWaveState(cfg Config, index int, seed int64) WaveState {
	if index < 1 {
		index = 1
	}

	duration := cfg.WaveDuration
	if duration <= 0 {
		duration = 20
	}

	seedBias := positiveModInt(int(seed), 4)
	runnerWeight := minInt(6, 1+(index-1)/2+(seedBias%2))
	tankWeight := 0
	if index >= 3 {
		tankWeight = minInt(4, 1+(index-3)/2+seedBias/2)
	}
	normalWeight := maxInt(2, 7-index/3-seedBias/3)
	guaranteedTankAt := 0
	if tankWeight > 0 {
		guaranteedTankAt = maxInt(6, 18-index*2-seedBias)
	}

	return WaveState{
		Index:            index,
		Label:            waveLabel(index, runnerWeight, tankWeight),
		StartTime:        float32(index-1) * duration,
		Duration:         duration,
		SpawnRateScale:   1 + 0.14*float32(index-1),
		NormalWeight:     normalWeight,
		RunnerWeight:     runnerWeight,
		TankWeight:       tankWeight,
		GuaranteedTankAt: guaranteedTankAt,
	}
}

func buildWaveStateForTime(cfg Config, t float32, seed int64) WaveState {
	duration := cfg.WaveDuration
	if duration <= 0 {
		duration = 20
	}

	index := 1 + int(t/duration)
	return buildWaveState(cfg, index, seed)
}

func waveLabel(index, runnerWeight, tankWeight int) string {
	switch {
	case tankWeight >= 3:
		return "Bulwark Surge"
	case runnerWeight >= 5:
		return "Raptor Swarm"
	case index%4 == 0:
		return "Harvest Moon"
	case index%3 == 0:
		return "Ash Drift"
	default:
		return "Grave Wind"
	}
}

func (w *World) updateWaveState() {
	next := buildWaveStateForTime(w.Cfg, w.TimeSurvived, w.rngSeed)
	if next.Index != w.Wave.Index {
		w.Wave = next
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func positiveModInt(v, m int) int {
	if m <= 0 {
		return 0
	}
	v %= m
	if v < 0 {
		v += m
	}
	return v
}
