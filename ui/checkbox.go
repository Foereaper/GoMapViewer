package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

/* ===============================
   CHECKBOX WIDGET
   =============================== */

type Checkbox struct {
	Value   bool
	Focused bool

	// cached hover state (updated externally)
	hovered bool
    Disabled bool
}

/* =======================
   UPDATE
   ======================= */

// Update handles keyboard + mouse toggling.
// The caller is responsible for setting Focused and hover state.
func (cb *Checkbox) Update() {
	if !cb.Focused {
		return
	}
    
    if cb.Disabled {
        return
    }

	// Keyboard toggle
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		cb.Value = !cb.Value
	}
}

// Click toggles the checkbox (call on mouse press when hovered)
func (cb *Checkbox) Click() {
    if cb.Disabled {
		return
	}
	cb.Value = !cb.Value
}

/* =======================
   DRAW
   ======================= */

// Draw renders the checkbox at x,y
func (cb *Checkbox) Draw(
	screen *ebiten.Image,
	x, y int,
) {
	size := 14

	// Background
	bg := color.RGBA{40, 40, 40, 255}
	if cb.hovered {
		bg = color.RGBA{60, 60, 60, 255}
	}
	if cb.Focused {
		bg = color.RGBA{80, 80, 80, 255}
	}
    if cb.Disabled {
		bg = color.RGBA{30, 30, 30, 255}
	}

	DrawRect(screen, x, y, size, size, bg)

	// Border
	DrawRect(screen, x, y, size, 1, color.Black)
	DrawRect(screen, x, y+size-1, size, 1, color.Black)
	DrawRect(screen, x, y, 1, size, color.Black)
	DrawRect(screen, x+size-1, y, 1, size, color.Black)

	// Checkmark
	if cb.Value {
		DrawRect(screen, x+3, y+3, size-6, size-6, color.White)
	}
}

/* =======================
   HIT TESTING
   ======================= */

// Hover updates hover state and returns true if mouse is inside
func (cb *Checkbox) Hover(mx, my, x, y int) bool {
	size := 14
	cb.hovered =
		mx >= x && mx <= x+size &&
			my >= y && my <= y+size
	return cb.hovered
}
