package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"horde-lab/internal/world"
)

const (
	profileVersion   = 1
	saveGameVersion  = 1
	highscoreVersion = 1
)

var characterChoices = []string{
	"daniel_kim",
	"hana_choi",
	"jayden_park",
	"mina_kang",
	"seojun_lee",
	"yuri_han",
}

var customizationChoices = []string{
	"Crimson",
	"Ivory",
	"Azure",
	"Emerald",
}

type PlayerProfile struct {
	Version       int    `json:"version"`
	Name          string `json:"name"`
	Character     string `json:"character"`
	Customization string `json:"customization"`
}

type SaveGame struct {
	Version  int            `json:"version"`
	SavedAt  time.Time      `json:"saved_at"`
	Profile  PlayerProfile  `json:"profile"`
	Snapshot world.Snapshot `json:"snapshot"`
}

type HighscoreEntry struct {
	At            time.Time `json:"at"`
	Name          string    `json:"name"`
	Character     string    `json:"character"`
	Customization string    `json:"customization"`
	Kills         int       `json:"kills"`
	Level         int       `json:"level"`
	TimeSurvived  float32   `json:"time_survived"`
	Score         int       `json:"score"`
}

type HighscoreFile struct {
	Version int              `json:"version"`
	Entries []HighscoreEntry `json:"entries"`
}

func defaultProfile() PlayerProfile {
	return PlayerProfile{
		Version:       profileVersion,
		Name:          "Hunter",
		Character:     characterChoices[0],
		Customization: customizationChoices[0],
	}
}

func characterDisplayName(id string) string {
	if id == "" {
		return "Unknown"
	}
	parts := strings.Split(id, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

func saveJSONAtomic(path string, v any) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}
	if err := ensureParentDir(path); err != nil {
		return fmt.Errorf("ensure parent dir: %w", err)
	}
	blob, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func loadProfile(path string) (PlayerProfile, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return PlayerProfile{}, err
	}
	var p PlayerProfile
	if err := json.Unmarshal(blob, &p); err != nil {
		return PlayerProfile{}, err
	}
	if p.Name == "" {
		p.Name = "Hunter"
	}
	if !slices.Contains(characterChoices, p.Character) {
		p.Character = characterChoices[0]
	}
	if !slices.Contains(customizationChoices, p.Customization) {
		p.Customization = customizationChoices[0]
	}
	if p.Version == 0 {
		p.Version = profileVersion
	}
	return p, nil
}

func saveProfile(path string, p PlayerProfile) error {
	p.Version = profileVersion
	return saveJSONAtomic(path, p)
}

func loadSaveGame(path string) (SaveGame, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return SaveGame{}, err
	}
	var sg SaveGame
	if err := json.Unmarshal(blob, &sg); err != nil {
		return SaveGame{}, err
	}
	if sg.Version != saveGameVersion {
		return SaveGame{}, fmt.Errorf("unsupported savegame version: %d", sg.Version)
	}
	return sg, nil
}

func saveSaveGame(path string, sg SaveGame) error {
	sg.Version = saveGameVersion
	return saveJSONAtomic(path, sg)
}

func loadHighscores(path string) (HighscoreFile, error) {
	blob, err := os.ReadFile(path)
	if err != nil {
		return HighscoreFile{}, err
	}
	var hs HighscoreFile
	if err := json.Unmarshal(blob, &hs); err != nil {
		return HighscoreFile{}, err
	}
	if hs.Version == 0 {
		hs.Version = highscoreVersion
	}
	return hs, nil
}

func saveHighscores(path string, hs HighscoreFile) error {
	hs.Version = highscoreVersion
	return saveJSONAtomic(path, hs)
}

func calcScore(s world.Snapshot) int {
	return s.Stats.EnemiesKilled*100 + int(s.TimeSurvived*10) + s.Player.Level*50
}

func sortHighscores(entries []HighscoreEntry) {
	slices.SortFunc(entries, func(a, b HighscoreEntry) int {
		if a.Score != b.Score {
			if a.Score > b.Score {
				return -1
			}
			return 1
		}
		if a.Kills != b.Kills {
			if a.Kills > b.Kills {
				return -1
			}
			return 1
		}
		if a.TimeSurvived > b.TimeSurvived {
			return -1
		}
		if a.TimeSurvived < b.TimeSurvived {
			return 1
		}
		return 0
	})
}
