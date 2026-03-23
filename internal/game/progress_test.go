package game

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"horde-lab/internal/world"
)

func TestLoadProfileNormalizesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	raw := []byte(`{"name":"","character":"unknown","customization":"bogus"}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	got, err := loadProfile(path)
	if err != nil {
		t.Fatalf("loadProfile failed: %v", err)
	}

	want := defaultProfile()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized profile mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestSaveGameRoundTrip(t *testing.T) {
	w := world.NewWorld(640, 360)
	defer w.Close()

	w.Player.Level = 3
	w.Player.Weapon = world.WeaponSpear
	w.TimeSurvived = 42.5

	want := SaveGame{
		Version: saveGameVersion,
		SavedAt: time.Date(2026, time.March, 23, 12, 0, 0, 0, time.UTC),
		Profile: PlayerProfile{
			Version:       profileVersion,
			Name:          "Hunter",
			Character:     "yuri_han",
			Customization: "Azure",
		},
		Snapshot: w.BuildSnapshot(),
	}

	path := filepath.Join(t.TempDir(), "savegame.json")
	if err := saveSaveGame(path, want); err != nil {
		t.Fatalf("saveSaveGame failed: %v", err)
	}

	got, err := loadSaveGame(path)
	if err != nil {
		t.Fatalf("loadSaveGame failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("savegame mismatch after round-trip\n got: %#v\nwant: %#v", got, want)
	}
}

func TestHighscoresSortAndRoundTrip(t *testing.T) {
	entries := []HighscoreEntry{
		{Name: "B", Score: 900, Kills: 12, TimeSurvived: 55},
		{Name: "A", Score: 1200, Kills: 10, TimeSurvived: 40},
		{Name: "C", Score: 900, Kills: 15, TimeSurvived: 30},
	}

	sortHighscores(entries)

	if entries[0].Name != "A" || entries[1].Name != "C" || entries[2].Name != "B" {
		t.Fatalf("unexpected highscore order: %#v", entries)
	}

	want := HighscoreFile{
		Version: highscoreVersion,
		Entries: entries,
	}

	path := filepath.Join(t.TempDir(), "highscores.json")
	if err := saveHighscores(path, want); err != nil {
		t.Fatalf("saveHighscores failed: %v", err)
	}

	got, err := loadHighscores(path)
	if err != nil {
		t.Fatalf("loadHighscores failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("highscore mismatch after round-trip\n got: %#v\nwant: %#v", got, want)
	}
}

func TestCalcScore(t *testing.T) {
	s := world.Snapshot{
		Player:       world.Player{Level: 4},
		Stats:        world.Stats{EnemiesKilled: 7},
		TimeSurvived: 12.3,
	}

	got := calcScore(s)
	want := 7*100 + 4*50 + 123
	if got != want {
		t.Fatalf("calcScore mismatch: got %d want %d", got, want)
	}
}
