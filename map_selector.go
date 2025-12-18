package main

import (
	"image/color"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"wowmap/ui"
)

type MapSelector struct {
	allMapNames      []string
	filteredMapNames []string

	active bool

	filter ui.TextField
	list   ui.ListBox
}

func NewMapSelector(mapNames []string) *MapSelector {
	names := append([]string{}, mapNames...)
	sort.Strings(names)

	return &MapSelector{
		allMapNames:      names,
		filteredMapNames: append([]string{}, names...),
	}
}

func (ms *MapSelector) Open() {
	ms.active = true
	ms.filter = ui.TextField{}

	ms.applyFilter()

	ms.list.Reset()
	ms.list.SetCount(len(ms.filteredMapNames))
}

func (ms *MapSelector) Close() {
	ms.active = false
}

func (ms *MapSelector) IsActive() bool {
	return ms.active
}

/* ===============================
   UPDATE
   =============================== */

func (ms *MapSelector) Update() (string, bool) {
	if !ms.active {
		return "", false
	}

	// Filter input
	old := ms.filter.Value
	ms.filter.Update()
	if ms.filter.Value != old {
		ms.applyFilter()
	}

	// Cancel
	if ui.Escape() {
		ms.Close()
		return "", false
	}

	// List update
	if idx, ok := ms.list.Update(); ok {
		if idx >= 0 && idx < len(ms.filteredMapNames) {
			return ms.filteredMapNames[idx], true
		}
	}

	return "", false
}

/* ===============================
   DRAW
   =============================== */

func (ms *MapSelector) Draw(screen *ebiten.Image) {
	if !ms.active {
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	panelW, panelH := 520, 420
	panelX := (w - panelW) / 2
	panelY := (h - panelH) / 2

	headerH := 26
	filterH := 24
	lineH := 18

	// Panel
	ui.DrawRect(screen, panelX, panelY, panelW, panelH, color.RGBA{30, 30, 30, 240})
	ui.DrawRect(screen, panelX, panelY, panelW, headerH, color.RGBA{45, 45, 45, 255})

	text.Draw(
		screen,
		"Select Map",
		basicfont.Face7x13,
		panelX+10,
		panelY+18,
		color.White,
	)

	// Filter
	filterY := panelY + headerH
	ui.DrawRect(screen, panelX+8, filterY+4, panelW-16, filterH-6, color.RGBA{20, 20, 20, 255})

	text.Draw(screen, "Filter:", basicfont.Face7x13, panelX+14, filterY+18, color.White)
	text.Draw(screen, ms.filter.Value, basicfont.Face7x13, panelX+70, filterY+18, color.White)

	if ms.filter.CaretVisible() {
		caretX := panelX + 70 + ms.filter.CursorPos*7
		ui.DrawRect(screen, caretX, filterY+6, 2, 14, color.White)
	}

	// List
	listY := filterY + filterH + 4
	listH := panelY + panelH - listY - 8

	ms.list.X = panelX + 6
	ms.list.Y = listY
	ms.list.W = panelW - 12
	ms.list.H = listH
	ms.list.LineH = lineH
	ms.list.SetCount(len(ms.filteredMapNames))

	start := ms.list.Scroll
	end := start + ms.list.VisibleRows()
	if end > len(ms.filteredMapNames) {
		end = len(ms.filteredMapNames)
	}

	for i := start; i < end; i++ {
		y := listY + (i-start)*lineH

		if i == ms.list.Index {
			ui.DrawRect(screen, panelX+6, y, panelW-12, lineH, color.RGBA{70, 70, 70, 255})
		}

		text.Draw(
			screen,
			ms.filteredMapNames[i],
			basicfont.Face7x13,
			panelX+12,
			y+14,
			color.White,
		)
	}

	ms.list.DrawScrollbar(screen)
}

/* ===============================
   FILTER
   =============================== */

func (ms *MapSelector) applyFilter() {
	ms.filteredMapNames = ms.filteredMapNames[:0]

	f := strings.ToLower(strings.TrimSpace(ms.filter.Value))

	for _, name := range ms.allMapNames {
		if f == "" || strings.Contains(strings.ToLower(name), f) {
			ms.filteredMapNames = append(ms.filteredMapNames, name)
		}
	}

	ms.list.Reset()
	ms.list.SetCount(len(ms.filteredMapNames))
}
