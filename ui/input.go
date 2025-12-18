package ui

import (
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

/* ===============================
   TEXT INPUT HELPERS
   =============================== */

// ReadPrintableRunes returns printable runes typed this frame.
// Use this instead of calling ebiten.AppendInputChars everywhere.
func ReadPrintableRunes() []rune {
	out := []rune{}
	for _, r := range ebiten.AppendInputChars(nil) {
		if unicode.IsPrint(r) {
			out = append(out, r)
		}
	}
	return out
}

/* ===============================
   KEY HELPERS
   =============================== */

func KeyPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func AnyKeyPressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
	}
	return false
}

/* ===============================
   NAVIGATION HELPERS
   =============================== */

func Up() bool    { return inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) }
func Down() bool  { return inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) }
func Left() bool  { return inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) }
func Right() bool { return inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) }

func Home() bool { return inpututil.IsKeyJustPressed(ebiten.KeyHome) }
func End() bool  { return inpututil.IsKeyJustPressed(ebiten.KeyEnd) }

func Enter() bool  { return inpututil.IsKeyJustPressed(ebiten.KeyEnter) }
func Escape() bool { return inpututil.IsKeyJustPressed(ebiten.KeyEscape) }

func Backspace() bool { return inpututil.IsKeyJustPressed(ebiten.KeyBackspace) }
func Delete() bool    { return inpututil.IsKeyJustPressed(ebiten.KeyDelete) }

/* ===============================
   MOUSE HELPERS
   =============================== */

func MouseJustPressed() bool {
	return inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
}

func MousePos() (int, int) {
	return ebiten.CursorPosition()
}

func MouseWheelY() float64 {
	_, y := ebiten.Wheel()
	return y
}
