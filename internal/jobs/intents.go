package jobs

import (
	"math"
	"sync"
)

type EnemySnapshot struct {
	EnemyID int
	X       float32
	Y       float32
}

type IntentRequest struct {
	Tick    uint64
	PlayerX float32
	PlayerY float32
	Enemies []EnemySnapshot
}

type EnemyIntent struct {
	EnemyID int
	DirX    float32
	DirY    float32
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

		dirX, dirY := normalize(dx, dy)
		out.Intents[i] = EnemyIntent{
			EnemyID: e.EnemyID,
			DirX:    dirX,
			DirY:    dirY,
		}
	}

	return out
}

func normalize(x, y float32) (float32, float32) {
	m2 := x*x + y*y
	if m2 == 0 {
		return 0, 0
	}

	inv := float32(1.0 / math.Sqrt(float64(m2)))
	return x * inv, y * inv
}
