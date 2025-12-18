package main

import (
	"image"
	"image/draw"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
    
    "wowmap/blp"
)

type TileRef struct {
	X, Y int
	Hash string
}

/* =======================
   Public API
   ======================= */

// ParseMD5Translate loads and parses md5translate.trs from disk.
func ParseMD5Translate(path string) (map[string][]TileRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseMD5Translate(data)
}

// ParseMD5TranslateFromFS loads and parses md5translate.trs from an fs.FS.
func ParseMD5TranslateFromFS(fsys fs.FS, path string) (map[string][]TileRef, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return parseMD5Translate(data)
}

// ParseMD5TranslateFromBytes parses md5translate.trs from raw bytes.
func ParseMD5TranslateFromBytes(data []byte) (map[string][]TileRef, error) {
	return parseMD5Translate(data)
}

/* =======================
   Core parser
   ======================= */

func parseMD5Translate(data []byte) (map[string][]TileRef, error) {
	lines := strings.Split(string(data), "\n")

	result := make(map[string][]TileRef)
	var currentDir string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Directory header
		if strings.HasPrefix(line, "dir: ") {
			currentDir = strings.TrimSpace(strings.TrimPrefix(line, "dir: "))
			result[currentDir] = nil
			continue
		}

		if currentDir == "" {
			continue
		}

		// Expected format: <path> <hash>
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		file := filepath.Base(fields[0])
		if !strings.HasSuffix(strings.ToLower(file), ".blp") {
			continue
		}

		name := strings.TrimSuffix(file, ".blp")
		toks := strings.Split(name, "_")
		if len(toks) < 2 {
			continue
		}

		x, ok := parseTrailingInt(toks[len(toks)-2])
		if !ok {
			continue
		}
		y, ok := parseTrailingInt(toks[len(toks)-1])
		if !ok {
			continue
		}

		result[currentDir] = append(result[currentDir], TileRef{
			X:    x,
			Y:    y,
			Hash: fields[1],
		})
	}

	// Prune empty directories
	for dir, tiles := range result {
		if len(tiles) == 0 {
			delete(result, dir)
		}
	}

	return result, nil
}

/* =======================
   Tile stitching
   ======================= */

// StitchTiles assembles a full image from minimap tiles.
func StitchTiles(tiles []TileRef, blpDir string) (image.Image, error) {
	if len(tiles) == 0 {
		return nil, nil
	}

	minX, minY := tiles[0].X, tiles[0].Y
	maxX, maxY := minX, minY

	for _, t := range tiles {
		if t.X < minX {
			minX = t.X
		}
		if t.Y < minY {
			minY = t.Y
		}
		if t.X > maxX {
			maxX = t.X
		}
		if t.Y > maxY {
			maxY = t.Y
		}
	}

	// Load first tile to determine tile size
	first, err := blp.DecodeBLP(filepath.Join(blpDir, tiles[0].Hash))
	if err != nil {
		return nil, err
	}
	tw := first.Bounds().Dx()
	th := first.Bounds().Dy()

	dst := image.NewRGBA(image.Rect(
		0, 0,
		(maxX-minX+1)*tw,
		(maxY-minY+1)*th,
	))
	draw.Draw(dst, dst.Bounds(), image.Transparent, image.Point{}, draw.Src)

	for _, t := range tiles {
		img, err := blp.DecodeBLP(filepath.Join(blpDir, t.Hash))
		if err != nil {
			continue // missing tiles are normal in some maps
		}

		x := (t.X - minX) * tw
		y := (t.Y - minY) * th

		draw.Draw(
			dst,
			image.Rect(x, y, x+tw, y+th),
			img,
			image.Point{},
			draw.Src,
		)
	}

	return dst, nil
}

/* =======================
   Helpers
   ======================= */

// parseTrailingInt extracts trailing digits from a string.
func parseTrailingInt(s string) (int, bool) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	if i == len(s) {
		return 0, false
	}
	v, err := strconv.Atoi(s[i:])
	return v, err == nil
}