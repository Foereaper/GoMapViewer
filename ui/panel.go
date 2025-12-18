package ui

import (
	"image/color"
	"github.com/hajimehoshi/ebiten/v2"
)

type Panel struct {
	X, Y, W, H int
	BG         color.Color
}

func (p Panel) Draw(screen *ebiten.Image) {
	DrawRect(screen, p.X, p.Y, p.W, p.H, p.BG)
}
