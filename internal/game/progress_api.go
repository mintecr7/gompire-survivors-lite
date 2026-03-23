package game

import "horde-lab/internal/world"

func DefaultProfile() PlayerProfile {
	return defaultProfile()
}

func LoadProfile(path string) (PlayerProfile, error) {
	return loadProfile(path)
}

func SaveProfile(path string, p PlayerProfile) error {
	return saveProfile(path, p)
}

func LoadSaveGame(path string) (SaveGame, error) {
	return loadSaveGame(path)
}

func SaveSaveGame(path string, sg SaveGame) error {
	return saveSaveGame(path, sg)
}

func LoadHighscores(path string) (HighscoreFile, error) {
	return loadHighscores(path)
}

func SaveHighscores(path string, hs HighscoreFile) error {
	return saveHighscores(path, hs)
}

func CalcScore(s world.Snapshot) int {
	return calcScore(s)
}

func SortHighscores(entries []HighscoreEntry) {
	sortHighscores(entries)
}
