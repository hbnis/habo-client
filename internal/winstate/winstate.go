// Package winstate persists the dashboard window's position and size
// across restarts.
package winstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	minWindowWidth  = 640
	minWindowHeight = 480
)

// userConfigDir is a seam for tests; production code always uses
// os.UserConfigDir.
var userConfigDir = os.UserConfigDir

// Bounds is a persisted window position and size.
type Bounds struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

func filePath() (string, error) {
	dir, err := userConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "albiondata-client", "window.json"), nil
}

// Load returns the previously saved window bounds. The second return
// value is false if nothing has been saved yet, the saved state can't
// be read, or the saved window is too small to be usable. Callers should
// fall back to their own defaults in those cases.
func Load() (Bounds, bool) {
	path, err := filePath()
	if err != nil {
		return Bounds{}, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Bounds{}, false
	}

	var b Bounds
	if err := json.Unmarshal(data, &b); err != nil {
		return Bounds{}, false
	}
	if b.Width < minWindowWidth || b.Height < minWindowHeight {
		return Bounds{}, false
	}
	return b, true
}

// Save persists the given window bounds, overwriting whatever was saved
// before.
func Save(b Bounds) error {
	path, err := filePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
