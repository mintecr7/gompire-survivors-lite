package game_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"horde-lab/internal/game"
	"horde-lab/internal/world"
)

func TestLoadProfileNormalizesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profile.json")
	raw := []byte(`{"name":"","character":"unknown","customization":"bogus"}`)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write profile fixture: %v", err)
	}

	got, err := game.LoadProfile(path)
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	want := game.DefaultProfile()
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

	want := game.SaveGame{
		Version: 1,
		SavedAt: time.Date(2026, time.March, 23, 12, 0, 0, 0, time.UTC),
		Profile: game.PlayerProfile{
			Version:       game.DefaultProfile().Version,
			Name:          "Hunter",
			Character:     "yuri_han",
			Customization: "Azure",
		},
		Snapshot: w.BuildSnapshot(),
	}

	path := filepath.Join(t.TempDir(), "savegame.json")
	if err := game.SaveSaveGame(path, want); err != nil {
		t.Fatalf("SaveSaveGame failed: %v", err)
	}

	got, err := game.LoadSaveGame(path)
	if err != nil {
		t.Fatalf("LoadSaveGame failed: %v", err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("savegame mismatch after round-trip\n got: %#v\nwant: %#v", got, want)
	}
}

func TestHighscoresSortAndRoundTrip(t *testing.T) {
	entries := []game.HighscoreEntry{
		{Name: "B", Score: 900, Kills: 12, TimeSurvived: 55},
		{Name: "A", Score: 1200, Kills: 10, TimeSurvived: 40},
		{Name: "C", Score: 900, Kills: 15, TimeSurvived: 30},
	}

	game.SortHighscores(entries)

	if entries[0].Name != "A" || entries[1].Name != "C" || entries[2].Name != "B" {
		t.Fatalf("unexpected highscore order: %#v", entries)
	}

	want := game.HighscoreFile{
		Version: 1,
		Entries: entries,
	}

	path := filepath.Join(t.TempDir(), "highscores.json")
	if err := game.SaveHighscores(path, want); err != nil {
		t.Fatalf("SaveHighscores failed: %v", err)
	}

	got, err := game.LoadHighscores(path)
	if err != nil {
		t.Fatalf("LoadHighscores failed: %v", err)
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

	got := game.CalcScore(s)
	want := 7*100 + 4*50 + 123
	if got != want {
		t.Fatalf("CalcScore mismatch: got %d want %d", got, want)
	}
}
