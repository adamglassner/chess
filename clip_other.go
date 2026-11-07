//go:build !windows

// Other systems: no clipboard; the moves are written to a file instead.
// (c) 2026 Ironite
package main

import (
	"errors"
	"os"
	"path/filepath"
)

func copyText(s string) error {
	return os.WriteFile(filepath.Join(saveDir(), "last-game.pgn"), []byte(s), 0o644)
}

func pasteText() (string, error) { return "", errors.New("no clipboard") }

func audioUnlock() {}
