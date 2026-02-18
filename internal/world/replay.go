package world

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"horde-lab/internal/shared/input"
)

const ReplayVersion = 1

type ReplayHeader struct {
	Version          int     `json:"version"`
	FixedStepSeconds float32 `json:"fixed_step_seconds"`
	Seed             int64   `json:"seed"`
	ConfigHash       string  `json:"config_hash"`
}

type ReplayFrame struct {
	Tick        uint64      `json:"tick"`
	Input       input.State `json:"input"`
	Choose      int         `json:"choose"`
	TogglePause bool        `json:"toggle_pause"`
	Restart     bool        `json:"restart"`
}

type ReplayFile struct {
	Header  ReplayHeader  `json:"header"`
	Initial Snapshot      `json:"initial"`
	Frames  []ReplayFrame `json:"frames"`
}

func BuildReplayHeader(initial Snapshot, fixedStepSeconds float32) (ReplayHeader, error) {
	cfgBlob, err := json.Marshal(initial.Cfg)
	if err != nil {
		return ReplayHeader{}, fmt.Errorf("marshal replay config: %w", err)
	}
	sum := sha256.Sum256(cfgBlob)
	return ReplayHeader{
		Version:          ReplayVersion,
		FixedStepSeconds: fixedStepSeconds,
		Seed:             initial.RNGSeed,
		ConfigHash:       hex.EncodeToString(sum[:]),
	}, nil
}

func SaveReplayFile(path string, rep ReplayFile) error {
	if path == "" {
		return fmt.Errorf("replay path is empty")
	}
	if rep.Header.Version != ReplayVersion {
		return fmt.Errorf("unsupported replay version: got %d want %d", rep.Header.Version, ReplayVersion)
	}
	blob, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal replay: %w", err)
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("ensure replay dir: %w", err)
		}
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, blob, 0o644); err != nil {
		return fmt.Errorf("write replay temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename replay temp file: %w", err)
	}
	return nil
}

func LoadReplayFile(path string) (ReplayFile, error) {
	if path == "" {
		return ReplayFile{}, fmt.Errorf("replay path is empty")
	}

	blob, err := os.ReadFile(path)
	if err != nil {
		return ReplayFile{}, fmt.Errorf("read replay file: %w", err)
	}

	var rep ReplayFile
	if err := json.Unmarshal(blob, &rep); err != nil {
		return ReplayFile{}, fmt.Errorf("decode replay file: %w", err)
	}
	if rep.Header.Version != ReplayVersion {
		return ReplayFile{}, fmt.Errorf("unsupported replay version: got %d want %d", rep.Header.Version, ReplayVersion)
	}
	if rep.Initial.Version != SnapshotVersion {
		return ReplayFile{}, fmt.Errorf("unsupported snapshot version in replay: got %d want %d", rep.Initial.Version, SnapshotVersion)
	}
	return rep, nil
}
