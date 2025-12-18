package main

import (
	"image/color"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"wowmap/blp"
	"wowmap/ui"
)

/* =======================
   Constants
   ======================= */

const (
	MaxMapTiles      = 64
	ADTWorldTileSize = 533.3333
	ADTGridCenter    = 32
)

/* =======================
   Types
   ======================= */

type tileKey struct{ x, y int }

type Game struct {
    // App context
    ctx      *AppContext

	// Map data
	mapNames []string
	current  int
	tiles    map[tileKey]*ebiten.Image

	// Tile cache
    cache *TileCache

	// Tile metadata
	minX, minY   int
	tileW, tileH int

	// Camera
	camX, camY float64
	zoom       float64

	// Mouse
	lastMX, lastMY int
	dragging       bool

	// UI
	selector *MapSelector
    
    // Startup error handling
    bootErr  error
    exiting  bool
}

/* =======================
   Initialization
   ======================= */

func NewGame(ctx *AppContext, bootErr error) *Game {
    g := &Game{
        ctx:     ctx,
        bootErr: bootErr,
        tiles:   make(map[tileKey]*ebiten.Image),
        zoom:    1,
        current: -1,
    }

    // If startup failed, return game early
    if bootErr != nil {
        return g
    }

    names := make([]string, 0, len(ctx.Minimaps))
    for k := range ctx.Minimaps {
        names = append(names, k)
    }
    sort.Strings(names)
    
    g.cache = NewTileCache()
    g.mapNames = names
    g.selector = NewMapSelector(names)
    g.selector.Open()

    return g
}

func mpqPath(p string) string {
	p = strings.ReplaceAll(p, "/", "\\")
	return strings.ToUpper(p)
}

func (g *Game) loadMap(index int) {
	g.tiles = make(map[tileKey]*ebiten.Image)
	g.current = index

	tiles := g.ctx.Minimaps[g.mapNames[index]]
	if len(tiles) == 0 {
		return
	}

	g.minX, g.minY = tiles[0].X, tiles[0].Y
	g.tileW, g.tileH = 0, 0

	for _, t := range tiles {
        path := mpqPath("textures/minimap/" + t.Hash)

        // Cache hit
        if cached, ok := g.cache.Get(path); ok {
            if g.tileW == 0 {
                b := cached.Bounds()
                g.tileW = b.Dx()
                g.tileH = b.Dy()
            }
            g.tiles[tileKey{t.X, t.Y}] = cached
            continue
        }

        // Cache miss
        img, err := blp.DecodeBLPFromFS(g.ctx.FS, path)
        if err != nil {
            log.Println(t.Hash, err)
            continue
        }

        if g.tileW == 0 {
            g.tileW = img.Bounds().Dx()
            g.tileH = img.Bounds().Dy()
        }

        eimg := ebiten.NewImageFromImage(img)
        g.cache.Put(path, eimg)
        g.tiles[tileKey{t.X, t.Y}] = eimg
    }

	g.camX, g.camY = 0, 0
	g.zoom = 1
	g.centerCameraOnTiles()
}

/* =======================
   Update
   ======================= */

func (g *Game) Update() error {
    // Startup error mode
    if g.bootErr != nil {
        if inpututil.IsKeyJustReleased(ebiten.KeyEnter) ||
           inpututil.IsKeyJustReleased(ebiten.KeyEscape) {
            return ebiten.Termination
        }
        return nil
    }

	mx, my := ebiten.CursorPosition()

	// Open map selector
	if inpututil.IsKeyJustReleased(ebiten.KeyM) && !g.selector.IsActive() {
		g.selector.Open()
	}

	// Selector active
	if g.selector.IsActive() {
		if name, ok := g.selector.Update(); ok {
			for i, n := range g.mapNames {
				if n == name {
					g.loadMap(i)
					break
				}
			}
			g.selector.Close()
		}
		return nil
	}

	// Mouse drag panning
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !g.dragging {
			g.dragging = true
			g.lastMX, g.lastMY = mx, my
		} else {
			g.camX -= float64(mx-g.lastMX) / g.zoom
			g.camY -= float64(my-g.lastMY) / g.zoom
			g.lastMX, g.lastMY = mx, my
		}
	} else {
		g.dragging = false
	}

	// Keyboard pan
	const panSpeed = 600.0
	step := (panSpeed / 60.0) / g.zoom
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		step *= 3
	}

	if ebiten.IsKeyPressed(ebiten.KeyLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camX -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camX += step
	}
	if ebiten.IsKeyPressed(ebiten.KeyUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camY -= step
	}
	if ebiten.IsKeyPressed(ebiten.KeyDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camY += step
	}

	// Zoom
	_, wy := ebiten.Wheel()
	if wy != 0 {
		oldZoom := g.zoom
		g.zoom *= math.Pow(1.1, wy)
		g.zoom = math.Max(0.02, math.Min(10, g.zoom))

		wx := float64(mx)/oldZoom + g.camX
		wy := float64(my)/oldZoom + g.camY
		g.camX = wx - float64(mx)/g.zoom
		g.camY = wy - float64(my)/g.zoom
	}

	return nil
}

/* =======================
   Draw
   ======================= */

func (g *Game) Draw(screen *ebiten.Image) {
    // Startup error mode
    if g.bootErr != nil {
        drawStartupError(screen, g.bootErr.Error())
        return
    }

	g.drawMapBounds(screen)
	g.drawMapTiles(screen)
	g.drawHeader(screen)

	if g.selector.IsActive() {
		g.selector.Draw(screen)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

/* =======================
   Rendering helpers
   ======================= */

func drawStartupError(screen *ebiten.Image, msg string) {
    w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

    panelW, panelH := 640, 220
    panelX := (w - panelW) / 2
    panelY := (h - panelH) / 2

    // Dim background
    ui.DrawRect(screen, 0, 0, w, h, color.RGBA{0, 0, 0, 160})

    // Panel
    ui.DrawRect(screen, panelX, panelY, panelW, panelH, color.RGBA{30, 30, 30, 240})
    ui.DrawRect(screen, panelX, panelY, panelW, 28, color.RGBA{70, 20, 20, 255})

    text.Draw(
        screen,
        "Startup Error",
        basicfont.Face7x13,
        panelX+12,
        panelY+20,
        color.White,
    )

    // Error message
    y := panelY + 56
    for _, line := range wrapText(msg, panelW-40, 7) {
        text.Draw(
            screen,
            line,
            basicfont.Face7x13,
            panelX+20,
            y,
            color.White,
        )
        y += 16
    }

    text.Draw(
        screen,
        "Press Enter or Esc to exit",
        basicfont.Face7x13,
        panelX+20,
        panelY+panelH-18,
        color.RGBA{200, 200, 200, 255},
    )
}

func (g *Game) drawMapTiles(screen *ebiten.Image) {
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()

	viewLeft := g.camX
	viewRight := g.camX + float64(w)/g.zoom
	viewTop := g.camY
	viewBottom := g.camY + float64(h)/g.zoom

	minTX := int(math.Floor(viewLeft/ADTWorldTileSize)) - 1
	maxTX := int(math.Ceil(viewRight/ADTWorldTileSize)) + 1
	minTY := int(math.Floor(viewTop/ADTWorldTileSize)) - 1
	maxTY := int(math.Ceil(viewBottom/ADTWorldTileSize)) + 1

	for tx := minTX; tx <= maxTX; tx++ {
		for ty := minTY; ty <= maxTY; ty++ {
			gx := tx + ADTGridCenter
			gy := ty + ADTGridCenter

			if gx < 0 || gx >= MaxMapTiles || gy < 0 || gy >= MaxMapTiles {
				continue
			}

			img := g.tiles[tileKey{gx, gy}]
			if img == nil {
				continue
			}

			worldX := float64(tx) * ADTWorldTileSize
			worldY := float64(ty) * ADTWorldTileSize

			sx := (worldX - g.camX) * g.zoom
			sy := (worldY - g.camY) * g.zoom

			var op ebiten.DrawImageOptions
			op.GeoM.Scale(
				g.zoom*ADTWorldTileSize/float64(g.tileW),
				g.zoom*ADTWorldTileSize/float64(g.tileH),
			)
			op.GeoM.Translate(sx, sy)

			screen.DrawImage(img, &op)
		}
	}
}

func (g *Game) drawMapBounds(screen *ebiten.Image) {
	half := float64(ADTGridCenter) * ADTWorldTileSize

	x1 := (-half - g.camX) * g.zoom
	y1 := (-half - g.camY) * g.zoom
	x2 := (half - g.camX) * g.zoom
	y2 := (half - g.camY) * g.zoom

	col := color.RGBA{255, 0, 0, 200}
	th := 2.0

	ebitenutil.DrawRect(screen, x1, y1, x2-x1, th, col)
	ebitenutil.DrawRect(screen, x1, y2-th, x2-x1, th, col)
	ebitenutil.DrawRect(screen, x1, y1, th, y2-y1, col)
	ebitenutil.DrawRect(screen, x2-th, y1, th, y2-y1, col)
}

func (g *Game) drawHeader(screen *ebiten.Image) {
	ui.DrawRect(screen, 10, 10, 360, 26, color.RGBA{40, 40, 40, 255})

	label := "Map: <none>  (Press M)"
	if g.current >= 0 {
		label = "Map: " + g.mapNames[g.current] + "  (Press M)"
	}

	text.Draw(screen, label, basicfont.Face7x13, 18, 28, color.White)
}

/* =======================
   Helpers
   ======================= */

func (g *Game) centerCameraOnTiles() {
	if len(g.tiles) == 0 {
		return
	}

	var sumX, sumY float64
	for k := range g.tiles {
		sumX += float64(k.x-ADTGridCenter) * ADTWorldTileSize
		sumY += float64(k.y-ADTGridCenter) * ADTWorldTileSize
	}

	count := float64(len(g.tiles))
	centerX := sumX / count
	centerY := sumY / count

	w, h := ebiten.WindowSize()
	if w == 0 || h == 0 {
		w, h = 1920, 1080
	}

	g.camX = centerX - float64(w)/(2*g.zoom)
	g.camY = centerY - float64(h)/(2*g.zoom)
}

func wrapText(s string, maxPx int, charW int) []string {
    maxChars := maxPx / charW
    var lines []string

    for _, p := range strings.Split(s, "\n") {
        for len(p) > maxChars {
            lines = append(lines, p[:maxChars])
            p = p[maxChars:]
        }
        lines = append(lines, p)
    }
    return lines
}