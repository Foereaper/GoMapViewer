package main

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

type TileCache struct {
	mu    sync.Mutex
	tiles map[string]*ebiten.Image
}

func NewTileCache() *TileCache {
	return &TileCache{
		tiles: make(map[string]*ebiten.Image),
	}
}

func (c *TileCache) Get(path string) (*ebiten.Image, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	img, ok := c.tiles[path]
	return img, ok
}

func (c *TileCache) Put(path string, img *ebiten.Image) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tiles[path] = img
}

func (c *TileCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.tiles)
}