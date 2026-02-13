package main

import (
	"log"

	"horde-lab/internal/game"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("Go-mpire survivors v0.1")

	g := game.New()
	defer g.Close()

	if err := ebiten.RunGame(g); err != nil {
		log.Printf("run game: %v", err)
	}
}
