package game

import (
	"horde-lab/internal/shared/input"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func ReadInput() input.State {
	return input.State{
		Up:    ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp),
		Down:  ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown),
		Left:  ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft),
		Right: ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight),
	}
}

func ReadRestart() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyR) || inpututil.IsKeyJustPressed(ebiten.KeyEnter)
}

func ReadPaused() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeySpace)
}

func ReadSaveSnapshot() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyF5)
}

func ReadLoadSnapshot() bool {
	return inpututil.IsKeyJustPressed(ebiten.KeyF9)
}
