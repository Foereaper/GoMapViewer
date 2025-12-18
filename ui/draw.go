package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

/* ===============================
   DRAW HELPERS
   =============================== */

func DrawRect(dst *ebiten.Image, x, y, w, h int, c color.Color) {
	img := ebiten.NewImage(w, h)
	img.Fill(c)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}