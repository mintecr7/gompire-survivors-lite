package jobs

import (
	"math"
	"sync"
)

type EnemyRole uint8

const (
	EnemyRoleNormal EnemyRole = iota
	EnemyRoleRunner
	EnemyRoleTank
)

type IntentMode uint8

const (
	IntentModePursue IntentMode = iota
	IntentModeStrafe
	IntentModeKite
	IntentModePressure
	IntentModeHold
)

type EnemySnapshot struct {
	EnemyID int
	Role    EnemyRole
	X       float32
	Y       float32
	Radius  float32
}

type IntentRequest struct {
	Tick    uint64
	PlayerX float32
	PlayerY float32
	Enemies []EnemySnapshot
}

type EnemyIntent struct {
	EnemyID        int
	MoveX          float32
	MoveY          float32
	SpeedScale     float32
	PreferredRange float32
	Mode           IntentMode
}

type IntentResult struct {
	Tick    uint64
	Intents []EnemyIntent
}

type IntentPool struct {
	Req  chan IntentRequest
	Res  chan IntentResult
	quit chan struct{}

	closeOnce sync.Once
	wg        sync.WaitGroup
}

func NewIntentPool(workerCount, queueSize int) *IntentPool {
	if workerCount < 1 {
		workerCount = 1
	}
	if queueSize < 1 {
		queueSize = 1
	}

	p := &IntentPool{
		Req:  make(chan IntentRequest, queueSize),
		Res:  make(chan IntentResult, queueSize),
		quit: make(chan struct{}),
	}

	p.wg.Add(workerCount)
	for range workerCount {
		go p.worker()
	}

	return p
}

func (p *IntentPool) Close() {
	p.closeOnce.Do(func() {
		close(p.quit)
		p.wg.Wait()
	})
}

func (p *IntentPool) worker() {
	defer p.wg.Done()

	for {
		select {
		case <-p.quit:
			return

		case req := <-p.Req:
			res := ComputeIntents(req)

			// Never block worker shutdown on a full result queue.
			select {
			case <-p.quit:
				return
			case p.Res <- res:
			default:
			}
		}
	}
}

func ComputeIntents(req IntentRequest) IntentResult {
	out := IntentResult{
		Tick:    req.Tick,
		Intents: make([]EnemyIntent, len(req.Enemies)),
	}

	for i, e := range req.Enemies {
		dx := req.PlayerX - e.X
		dy := req.PlayerY - e.Y

		dist := distance(dx, dy)
		chaseX, chaseY := normalize(dx, dy)
		if chaseX == 0 && chaseY == 0 {
			chaseX, chaseY = fallbackDirection(e.EnemyID)
		}

		sepRadius := maxf(24.0, e.Radius*3.2)
		sepX, sepY := separation(req.Enemies, i, sepRadius)

		mode := IntentModePursue
		preferred := float32(65.0)
		speedScale := float32(1.0)
		sepWeight := float32(0.3)
		baseX, baseY := chaseX, chaseY

		switch e.Role {
		case EnemyRoleRunner:
			preferred = 110
			sepWeight = 0.2
			tanX, tanY := perpendicular(chaseX, chaseY, e.EnemyID)

			switch {
			case dist > 190:
				mode = IntentModePursue
				speedScale = 1.20
				baseX, baseY = blend(chaseX, chaseY, tanX, tanY, 0.15)
			case dist > 95:
				mode = IntentModeStrafe
				speedScale = 1.10
				baseX, baseY = blend(chaseX, chaseY, tanX, tanY, 0.55)
			default:
				mode = IntentModeKite
				speedScale = 1.28
				baseX, baseY = blend(-chaseX, -chaseY, tanX, tanY, 0.65)
			}

		case EnemyRoleTank:
			preferred = 45
			sepWeight = 0.55

			switch {
			case dist > 150:
				mode = IntentModePressure
				speedScale = 0.95
			case dist > 80:
				mode = IntentModePressure
				speedScale = 0.75
			default:
				mode = IntentModeHold
				speedScale = 0.42
			}

		default:
			preferred = 65
			sepWeight = 0.32

			if dist < 80 {
				mode = IntentModePressure
				speedScale = 0.85
			}
		}

		moveX, moveY := normalize(baseX+sepX*sepWeight, baseY+sepY*sepWeight)
		if moveX == 0 && moveY == 0 {
			moveX, moveY = normalize(baseX, baseY)
		}

		out.Intents[i] = EnemyIntent{
			EnemyID:        e.EnemyID,
			MoveX:          moveX,
			MoveY:          moveY,
			SpeedScale:     clampf(speedScale, 0.2, 1.5),
			PreferredRange: preferred,
			Mode:           mode,
		}
	}

	return out
}

func separation(enemies []EnemySnapshot, selfIdx int, radius float32) (float32, float32) {
	self := enemies[selfIdx]
	r2 := radius * radius
	var sx, sy float32

	for i, other := range enemies {
		if i == selfIdx {
			continue
		}

		dx := self.X - other.X
		dy := self.Y - other.Y
		d2 := dx*dx + dy*dy
		if d2 == 0 || d2 > r2 {
			continue
		}

		inv := float32(1.0 / math.Sqrt(float64(d2)))
		weight := 1 - (d2 / r2)
		sx += dx * inv * weight
		sy += dy * inv * weight
	}

	return normalize(sx, sy)
}

func blend(ax, ay, bx, by, bWeight float32) (float32, float32) {
	aWeight := 1 - bWeight
	return ax*aWeight + bx*bWeight, ay*aWeight + by*bWeight
}

func perpendicular(x, y float32, seed int) (float32, float32) {
	if seed%2 == 0 {
		return -y, x
	}
	return y, -x
}

func fallbackDirection(seed int) (float32, float32) {
	if seed%2 == 0 {
		return 1, 0
	}
	return 0, 1
}

func distance(x, y float32) float32 {
	return float32(math.Sqrt(float64(x*x + y*y)))
}

func normalize(x, y float32) (float32, float32) {
	m2 := x*x + y*y
	if m2 == 0 {
		return 0, 0
	}

	inv := float32(1.0 / math.Sqrt(float64(m2)))
	return x * inv, y * inv
}

func clampf(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
