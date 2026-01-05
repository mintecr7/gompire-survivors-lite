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

## Tech stack (initial)

- Go (latest stable recommended)
- Rendering/input: **Ebiten** (likely), but kept modular so the core simulation can run headless.
- Embeddings/networking are not part of this repo (those come later in the successor project).

> Note: If you prefer a different rendering approach (SDL bindings, Raylib-Go, etc.), the architecture still holds.

---

## Suggested project layout

This is the intended structure once code starts landing:

/cmd/game # main entrypoint (client)
/internal/world # simulation loop, systems, entities
/internal/render # renderer adapter (Ebiten or other)
/internal/assets # async asset pipeline
/internal/jobs # worker pool + job definitions
/internal/telemetry # logging/metrics
/internal/math # vector helpers, spatial hash, etc.
/docs # design notes, decisions, diagrams


---

## Getting started (once code exists)

### Run
```bash
go run ./cmd/game
```
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

- **v0.1**: basic loop (move, spawn, hit, die, level up)
- **v0.2**: async asset loader + telemetry goroutine
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