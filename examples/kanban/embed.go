package kanban

import (
	"errors"
	"io/fs"
)

var (
// Temporarily disabled for debugging
// //go:embed frontend/dist/*
// embeddedDist embed.FS
)

func EmbeddedDistFS() (fs.FS, error) {
	return nil, errors.New("embedded assets disabled")
}
