package kanban

import (
	"errors"
	"io/fs"
)

var (
	// ErrEmbeddedAssetsDisabled is returned when embedded assets are not available
	ErrEmbeddedAssetsDisabled = errors.New("embedded assets disabled")

// Temporarily disabled for debugging
// //go:embed frontend/dist/*
// embeddedDist embed.FS
)

func EmbeddedDistFS() (fs.FS, error) {
	return nil, ErrEmbeddedAssetsDisabled
}
