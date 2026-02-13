# (Go)-Mpire Survivors-lite 

A learning-driven game project to practice **Go** and **concurrent programming** by building a **top-down horde survival** game (think “Vampire Survivors-lite”). The focus is not just gameplay—it's clean architecture: a deterministic simulation loop plus concurrent subsystems that are practical and idiomatic in Go.

Planned progression:
1. **Single-player horde survival** (this repo)
2. **Authoritative multiplayer arena** (successor repo / major milestone)
3. **Doom-like project** (long-term)

---

## Why this exists

I’m studying Go and concurrency by building something real:
- Fast iteration
- Clear scope
- Lots of natural systems (AI, spawning, upgrades, persistence, telemetry) that benefit from concurrency

---

## Core principles

### 1) Deterministic world state
- The simulation runs in a **single goroutine** (the “world owner”).
- All game state mutation happens there.
- This keeps debugging sane and avoids race-condition chaos.

### 2) Concurrency where it actually helps
Concurrency is used for “satellite work” that should not block the tick:
- asset loading / decoding
- AI intent computation (job workers)
- procedural generation (waves, maps)
- logging / telemetry
- save/load I/O

### 3) Message passing over shared mutable state
Subsystems communicate with the world via **channels/messages**, not direct state access.

---

## MVP scope (v0.1)

A playable 5–10 minute loop:
- player movement + collision with bounds (obstacles later)
- enemy spawn system + simple pursuit
- auto-attack weapon + damage + deaths
- XP drops + level-ups + a few upgrades
- basic HUD (HP, XP/level, timer/wave)

---

## Concurrency milestones

### v0.1 — Stable baseline
- single-threaded `World.Tick()`
- event/message channel into the world

### v0.2 — Async assets + telemetry
- asset loader goroutine (requests/results channels)
- metrics/logging goroutine (batching)

### v0.3 — Worker pool jobs
- AI intent jobs (compute decisions off-thread)
- results applied by world on the next tick

### v0.4 — Persistence + replay
- save/load snapshots
- deterministic replay experiments (optional)

---

## Tech stack (current)

- Go **1.24+**
- Rendering/input: **Ebiten v2**
- Embeddings/networking are not part of this repo (those come later in the successor project).

---

## Current project layout

The repo currently looks like this:

`/cmd/game` - main entrypoint (Ebiten run loop)  
`/internal/game` - fixed-step game integration, input handling, asset polling  
`/internal/world` - deterministic world state, systems, combat, leveling, drawing  
`/internal/assets` - async asset loader (request/result channels)  
`/internal/telemetry` - telemetry sink/batching prototype  
`/internal/shared/input` - shared input state types  
`/internal/commons/logger_config` - shared structured logger setup  
`/internal/render` - render package placeholder (future split from world)


---

## Getting started

### Prerequisites
```bash
go version
```
Use Go 1.24 or newer.

### Run
```bash
go run ./cmd/game
```

### Controls
- `WASD` or Arrow keys: move
- `Space`: pause/resume
- `1` / `2`: choose level-up upgrade
- `R` or `Enter`: restart (when paused or game over)

### Test
```bash
go test ./...
```

---

## Design Notes

### The World Owner Pattern

One goroutine owns all state.

External inputs arrive as messages.

Tick loop processes:
- inbox messages
- system updates
- queued results from workers
- snapshot for rendering

This pattern directly carries over to the multiplayer successor, where network goroutines push inputs to the server's world loop.

### Roadmap

- **:white_check_mark:** **v0.1**: basic loop (move, spawn, hit, die, level up)
- **:white_check_mark:** **v0.2**: async asset loader + telemetry goroutine
- **v0.3**: worker pool for AI intent/jobs
- **v0.4**: persistence + replay experiments
- **v1.0**: "fun enough" release (balance + polish)

### Successor Repo

- authoritative server tick loop
- goroutine-per-connection input read
- snapshot broadcast with backpressure
- client interpolation (optional prediction later)

### Contributing / Notes

This is primarily a personal learning project. Issues/PRs are welcome if they align with:

- small scope
- clear concurrency value
- maintainable architecture
