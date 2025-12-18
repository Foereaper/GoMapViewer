package main

import (
	"fmt"
	"log"
	"os"
    "io/fs"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"wowmap/mpq"
	"wowmap/vfs"
)

type AppContext struct {
	Cfg      *Config
    FS       fs.FS
	Minimaps map[string][]TileRef
    
    
}

func main() {
    ctx, err := initAppContext()

    var bootErr error
    if err == nil {
        err = initVFS(ctx)
    }
    if err != nil {
        bootErr = err
    }

    game := NewGame(ctx, bootErr)

    ebiten.SetWindowSize(1920, 1080)
    ebiten.SetWindowResizable(true)
    ebiten.SetWindowTitle("WoW Map Viewer")

    if err := ebiten.RunGame(game); err != nil {
        log.Fatal(err)
    }
}

func initAppContext() (*AppContext, error) {
	cfgPath := "config.json"

	cfg, created, err := loadOrInitConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	if created {
		return nil, fmt.Errorf("config.json created, please edit it and restart")
	}

	ctx := &AppContext{
		Cfg: cfg,
	}

	return ctx, nil
}

func initVFS(ctx *AppContext) error {
	stack := vfs.New()

	wowDataDir := ctx.Cfg.WowDataPath
	if err := LoadMPQs(stack, wowDataDir); err != nil {
		return err
	}

	ctx.FS = vfs.NewFS(stack)

	md5Path := "Textures/Minimap/md5translate.trs"
	maps, err := ParseMD5TranslateFromFS(ctx.FS, md5Path)
	if err != nil {
		return err
	}

	ctx.Minimaps = maps
	return nil
}

// LoadMPQs loads MPQs in Blizzard's static patch order.
// Missing MPQs are skipped (normal for some installs).
func LoadMPQs(stack *vfs.MPQStack, dataDir string) error {
	// Case-insensitive filename lookup in Data dir
	findInDir := func(want string) (string, bool) {
		want = strings.ToLower(want)

		entries, err := os.ReadDir(dataDir)
		if err != nil {
			return "", false
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.ToLower(e.Name()) == want {
				return filepath.Join(dataDir, e.Name()), true
			}
		}
		return "", false
	}

	// Open and add MPQ if present
	addIfExists := func(name string) error {
		full, ok := findInDir(name)
		if !ok {
			return nil
		}

		a, err := mpq.Open(full)
		if err != nil {
			return fmt.Errorf("open %s: %w", full, err)
		}

		// NOTE: MPQs stay open for the lifetime of the app
		return stack.Add(a)
	}

	// Fixed base order
	fixed := []string{
		"base.MPQ",
		"common.MPQ",
		"common-2.MPQ",
		"expansion.MPQ",
		"lichking.MPQ",
		"patch.MPQ",
		"patch-2.MPQ",
		"patch-3.MPQ",
	}

	for _, name := range fixed {
		if err := addIfExists(name); err != nil {
			return err
		}
	}

	// Numeric patches: patch-4 .. patch-9
	for n := 4; n <= 9; n++ {
		if err := addIfExists(fmt.Sprintf("patch-%d.MPQ", n)); err != nil {
			return err
		}
	}

	// Letter patches: patch-a .. patch-z
	for ch := 'a'; ch <= 'z'; ch++ {
		if err := addIfExists(fmt.Sprintf("patch-%c.MPQ", ch)); err != nil {
			return err
		}
	}

	return nil
}
