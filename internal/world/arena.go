package world

import (
	"math"
	"math/rand"
)

func generateObstacles(worldW, worldH float32, cfg Config, seed int64, anchor Vec2) []Obstacle {
	if cfg.ObstacleCount <= 0 || cfg.ObstacleRadiusMax <= 0 {
		return nil
	}

	minRadius := cfg.ObstacleRadiusMin
	maxRadius := cfg.ObstacleRadiusMax
	if minRadius <= 0 {
		minRadius = 20
	}
	if maxRadius < minRadius {
		maxRadius = minRadius
	}

	safeRadius := cfg.StartSafeRadius
	if safeRadius <= 0 {
		safeRadius = minf(cfg.SpawnRadius*0.45, minf(worldW, worldH)*0.2)
	}

	rng := rand.New(rand.NewSource(seed ^ 0x5e0f1a77))
	out := make([]Obstacle, 0, cfg.ObstacleCount)
	maxAttempts := cfg.ObstacleCount * 24

	for attempts := 0; len(out) < cfg.ObstacleCount && attempts < maxAttempts; attempts++ {
		radius := minRadius
		if maxRadius > minRadius {
			radius += rng.Float32() * (maxRadius - minRadius)
		}

		pos := Vec2{
			X: radius + rng.Float32()*maxf(1, worldW-radius*2),
			Y: radius + rng.Float32()*maxf(1, worldH-radius*2),
		}
		if dist2(pos, anchor) < squaref(radius+safeRadius) {
			continue
		}

		blocked := false
		for _, existing := range out {
			gap := radius + existing.R + cfg.ObstaclePadding*2 + 18
			if dist2(pos, existing.Pos) < gap*gap {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}

		out = append(out, Obstacle{Pos: pos, R: radius})
	}

	return out
}

func (w *World) resolveEntityPosition(pos Vec2, r float32) Vec2 {
	hiX := maxf(r, w.W-r)
	hiY := maxf(r, w.H-r)
	pos.X = clamp(pos.X, r, hiX)
	pos.Y = clamp(pos.Y, r, hiY)

	for range 2 {
		resolved := true
		for _, obstacle := range w.Obstacles {
			minDist := obstacle.R + r + w.Cfg.ObstaclePadding
			delta := pos.Sub(obstacle.Pos)
			d2 := delta.X*delta.X + delta.Y*delta.Y
			if d2 >= minDist*minDist {
				continue
			}

			resolved = false
			if d2 == 0 {
				pos = Vec2{X: obstacle.Pos.X + minDist, Y: obstacle.Pos.Y}
			} else {
				d := float32(math.Sqrt(float64(d2)))
				pos = obstacle.Pos.Add(delta.Mul(minDist / d))
			}

			pos.X = clamp(pos.X, r, hiX)
			pos.Y = clamp(pos.Y, r, hiY)
		}
		if resolved {
			break
		}
	}

	return pos
}

func (w *World) overlapsObstacle(pos Vec2, r float32) bool {
	for _, obstacle := range w.Obstacles {
		minDist := obstacle.R + r + w.Cfg.ObstaclePadding
		if dist2(pos, obstacle.Pos) < minDist*minDist {
			return true
		}
	}
	return false
}

func squaref(v float32) float32 {
	return v * v
}
