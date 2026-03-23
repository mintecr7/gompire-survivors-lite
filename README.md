# (Go)-Mpire Survivors-lite

A learning-driven game project to practice **Go** and **concurrent programming**
by building a **top-down horde survival** game (think “Vampire Survivors-lite”).
The focus is not just gameplay—it's clean architecture: a deterministic
simulation loop plus concurrent subsystems that are practical and idiomatic in
Go.

Planned progression:

1. **Single-player horde survival** (this repo)
2. **Authoritative multiplayer arena** (successor repo / major milestone)
3. **Doom-like project** (long-term)

Current status:

- Core `v0.1` to `v0.4` foundations are implemented.
- The project is now in the "polish + expand gameplay" stage: richer content,
  procedural systems, balancing, and cleanup.

---

## Why this exists

I’m studying Go and concurrency by building something real:

- Fast iteration
- Clear scope
- Lots of natural systems (AI, spawning, upgrades, persistence, telemetry) that
  benefit from concurrency

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

Subsystems communicate with the world via **channels/messages**, not direct
state access.

---

## MVP scope (v0.1)

A playable 5–10 minute loop:

- player movement + collision with bounds (obstacles later)
- enemy spawn system + simple pursuit
- auto-attack weapon + damage + deaths
- XP drops + level-ups + a few upgrades
- basic HUD (HP, XP/level, timer/wave)

---

## Milestone status

### v0.1 — Stable baseline (`done`)

- single-threaded `World.Tick()`
- fixed-step deterministic simulation
- channel-backed message inbox into the world

### v0.2 — Async assets + telemetry (`done`)

- asset loader goroutine (requests/results channels)
- metrics/logging goroutine (batching)
- async asset handoff used by rendering

### v0.3 — Worker pool jobs (`done`)

- AI intent jobs (compute decisions off-thread)
- results applied by world on the next tick
- richer role-based enemy intent payloads

### v0.4 — Persistence + replay foundation (`implemented`)

- save/load snapshots
- replay record/playback
- savegame/profile/highscores
- determinism + persistence tests

### Next focus

- procedural generation (waves/maps)
- more enemy/weapon/content variety
- balancing, polish, and render split cleanup

---

## Tech stack (current)

- Go **1.24+**
- Rendering/input: **Ebiten v2**
- Embeddings/networking are not part of this repo (those come later in the
  successor project).

---

## Current project layout

The repo currently looks like this:

- `/cmd/game` - main entrypoint (Ebiten run loop)
- `/internal/game` - fixed-step integration, replay/save/profile glue
- `/internal/world` - deterministic world state, systems, combat, leveling,
  snapshot/replay support, drawing
- `/internal/jobs` - worker-pool AI intent jobs
- `/internal/assets` - async asset loader and embedded asset fallback
- `/internal/telemetry` - telemetry sink/batching
- `/internal/shared/input` - shared input state types
- `/internal/render` - render package placeholder (future split from world)
- `/internal/replay` - replay-related package space for future expansion
- `/internal/commons/logger_config` - shared structured logger setup
- `/internal/*/test` - per-module test packages

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
- `F5`: save snapshot (`.dist/snapshot.json`)
- `F9`: load snapshot (`.dist/snapshot.json`)
- `F6`: save replay (`.dist/replay.json`)
- `F10`: load + start replay (`.dist/replay.json`)
- `F7`: stop and save game (`.dist/savegame.json`)
- `F8`: load saved game (`.dist/savegame.json`)
- `C`: continue paused game
- `F1`: cycle character preset
- `F2`: cycle customization preset

### Test

Tests live in per-module `test/` directories such as `internal/world/test` and
`internal/game/test`.

```bash
go test ./...
```

---

## Design Notes

### The World Owner Pattern

One goroutine owns all state.

External inputs arrive through a buffered inbox channel.

Tick loop processes:

- inbox messages
- system updates
- queued results from workers
- snapshot for rendering

This pattern directly carries over to the multiplayer successor, where network
goroutines push inputs to the server's world loop.

### Roadmap

- **✅** **v0.1**: basic loop (move, spawn, hit, die, level up)
- **✅** **v0.2**: async asset loader + telemetry goroutine
- **✅** **v0.3**: worker pool for AI intent/jobs
- **✅** **v0.4 foundation**: snapshots, replay, savegame/profile/highscores,
  and determinism coverage
- **Next**: procedural generation, more content, balance, and render cleanup
- **v1.0**: "fun enough" release (balance + polish)

### Successor Repo

- authoritative server tick loop
- goroutine-per-connection input read
- snapshot broadcast with backpressure
- client interpolation (optional prediction later)

### Contributing / Notes

This is primarily a personal learning project. Issues/PRs are welcome if they
align with:

- small scope
- clear concurrency value
- maintainable architecture
