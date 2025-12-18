package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

/* ===============================
   LIST BOX
   =============================== */

type ListBox struct {
	// Geometry
	X, Y, W, H int
	LineH      int

	// Content
	Count int

	// State
	Index   int // selected index
	Scroll  int // first visible index
	Hovered int // hovered index, -1 if none

	UpTicks   int
	DownTicks int

	dragging    bool
	dragOffsetY int

	// When true, we avoid snapping Scroll back to keep Index visible.
	// This is set when the user scrolls (wheel/scrollbar) and cleared when Index changes.
	suppressEnsure bool
}

/* ===============================
   UPDATE
   =============================== */

// Update handles mouse + keyboard interaction.
// Returns (activatedIndex, true) when user activates an item (Enter or click).
func (lb *ListBox) Update() (int, bool) {
	lb.Hovered = -1

	mx, my := ebiten.CursorPosition()

	// Mouse wheel

	_, wy := ebiten.Wheel()
	if wy != 0 {
		lb.Scroll -= int(wy)
		lb.ClampScroll()
		lb.suppressEnsure = true
	}

	// Scrollbar interaction

	if lb.hasScrollbar() {
		tx, ty, tw, th := lb.thumbRect()
		_, trackY, _, trackH := lb.scrollbarRect()

		// Start dragging (thumb only)
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			if mx >= tx && mx <= tx+tw && my >= ty && my <= ty+th {
				lb.dragging = true
				lb.dragOffsetY = my - ty
				lb.suppressEnsure = true
				lb.Hovered = -1
			}
		}

		// Dragging
		if lb.dragging {
			if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
				newY := my - lb.dragOffsetY
				maxY := trackY + trackH - th
				if newY < trackY {
					newY = trackY
				}
				if newY > maxY {
					newY = maxY
				}

				den := trackH - th
				if den <= 0 {
					lb.Scroll = 0
				} else {
					ratio := float64(newY-trackY) / float64(den)
					lb.Scroll = int(ratio * float64(lb.Count-lb.VisibleRows()))
				}
				lb.ClampScroll()
			} else {
				// Drag ended: keep scroll where user left it
				lb.dragging = false
				lb.Hovered = -1
				lb.suppressEnsure = true
			}
		}
	}

	// Mouse hover and hover-to-select

	// Don't hover-select while dragging OR while cursor is over the scrollbar.
	if !lb.dragging && !lb.mouseOverScrollbar(mx, my) {
		if idx := lb.HoverIndex(mx, my); idx != -1 {
			lb.Hovered = idx
			if lb.Index != idx {
				lb.Index = idx
				lb.suppressEnsure = false // user changed selection; now it's ok to keep it visible
			}
		}
	}

	// Mouse click activation

	// If the mouse is over the scrollbar, do NOT activate list items.
	if !lb.dragging && lb.Hovered != -1 && !lb.mouseOverScrollbar(mx, my) &&
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return lb.Index, true
	}

	// Keyboard navigation

	const (
		initialDelay = 15
		repeatRate   = 3
	)

	indexChanged := false

	if ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		lb.DownTicks++
		if lb.DownTicks == 1 || (lb.DownTicks > initialDelay && (lb.DownTicks-initialDelay)%repeatRate == 0) {
			lb.Index++
			indexChanged = true
		}
	} else {
		lb.DownTicks = 0
	}

	if ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		lb.UpTicks++
		if lb.UpTicks == 1 || (lb.UpTicks > initialDelay && (lb.UpTicks-initialDelay)%repeatRate == 0) {
			lb.Index--
			indexChanged = true
		}
	} else {
		lb.UpTicks = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		lb.Index += lb.VisibleRows()
		indexChanged = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		lb.Index -= lb.VisibleRows()
		indexChanged = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyHome) {
		lb.Index = 0
		indexChanged = true
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnd) {
		lb.Index = lb.Count - 1
		indexChanged = true
	}

	if indexChanged {
		lb.suppressEnsure = false
	}

	lb.ClampIndex()

	if !lb.dragging && !lb.suppressEnsure {
		lb.EnsureVisible()
	}

	// Activation

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		return lb.Index, true
	}

	return -1, false
}

/* ===============================
   HIT TESTING
   =============================== */

func (lb *ListBox) HoverIndex(mx, my int) int {
	if mx < lb.X || mx > lb.X+lb.W {
		return -1
	}
	if my < lb.Y || my >= lb.Y+lb.H {
		return -1
	}

	row := (my - lb.Y) / lb.LineH
	idx := lb.Scroll + row

	if idx >= 0 && idx < lb.Count {
		return idx
	}
	return -1
}

/* ===============================
   SCROLLING
   =============================== */

func (lb *ListBox) VisibleRows() int {
	if lb.LineH <= 0 {
		return 0
	}
	return lb.H / lb.LineH
}

func (lb *ListBox) EnsureVisible() {
	vis := lb.VisibleRows()
	if vis <= 0 {
		return
	}

	if lb.Index < lb.Scroll {
		lb.Scroll = lb.Index
	}
	if lb.Index >= lb.Scroll+vis {
		lb.Scroll = lb.Index - vis + 1
	}

	lb.ClampScroll()
}

func (lb *ListBox) ClampScroll() {
	max := lb.Count - lb.VisibleRows()
	if max < 0 {
		max = 0
	}
	if lb.Scroll < 0 {
		lb.Scroll = 0
	}
	if lb.Scroll > max {
		lb.Scroll = max
	}
}

/* ===============================
   SCROLLBAR (DRAW + GEOMETRY)
   =============================== */

func (lb *ListBox) DrawScrollbar(screen *ebiten.Image) {
	if !lb.hasScrollbar() {
		return
	}

	trackX, trackY, trackW, trackH := lb.scrollbarRect()
	tx, ty, tw, th := lb.thumbRect()

	// Track
	DrawRect(screen, trackX, trackY, trackW, trackH, color.RGBA{25, 25, 25, 255})

	// Thumb
	DrawRect(screen, tx, ty, tw, th, color.RGBA{90, 90, 90, 255})
}

func (lb *ListBox) scrollbarWidth() int { return 10 }

func (lb *ListBox) hasScrollbar() bool {
	return lb.Count > lb.VisibleRows()
}

func (lb *ListBox) scrollbarRect() (x, y, w, h int) {
	w = lb.scrollbarWidth()
	x = lb.X + lb.W - w
	y = lb.Y
	h = lb.H
	return
}

func (lb *ListBox) thumbRect() (x, y, w, h int) {
	trackX, trackY, trackW, trackH := lb.scrollbarRect()

	vis := lb.VisibleRows()
	if vis <= 0 || lb.Count <= 0 {
		return 0, 0, 0, 0
	}

	ratio := float64(vis) / float64(lb.Count)
	h = int(float64(trackH) * ratio)
	if h < 12 {
		h = 12
	}

	maxScroll := lb.Count - vis
	if maxScroll <= 0 {
		y = trackY
	} else {
		y = trackY + int(float64(trackH-h)*float64(lb.Scroll)/float64(maxScroll))
	}

	x = trackX
	w = trackW
	return
}

func (lb *ListBox) mouseOverScrollbar(mx, my int) bool {
	if !lb.hasScrollbar() {
		return false
	}
	x, y, w, h := lb.scrollbarRect()
	return mx >= x && mx <= x+w && my >= y && my <= y+h
}

/* ===============================
   INDEX SAFETY
   =============================== */

func (lb *ListBox) ClampIndex() {
	if lb.Count <= 0 {
		lb.Index = 0
		return
	}
	if lb.Index < 0 {
		lb.Index = 0
	}
	if lb.Index >= lb.Count {
		lb.Index = lb.Count - 1
	}
}

/* ===============================
   UTIL
   =============================== */

func (lb *ListBox) SetCount(n int) {
	lb.Count = n
	lb.ClampIndex()
	lb.ClampScroll()
}

func (lb *ListBox) Reset() {
	lb.Index = 0
	lb.Scroll = 0
	lb.Hovered = -1
	lb.UpTicks = 0
	lb.DownTicks = 0
	lb.dragging = false
	lb.dragOffsetY = 0
	lb.suppressEnsure = false
}
