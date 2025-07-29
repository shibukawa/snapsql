package testdata

import "embed"

//go:embed acceptancetests/*/*.sql acceptancetests/*/*.json
var AcceptanceTests embed.FS

// GetFS returns the embedded filesystem
func GetFS() embed.FS {
	return AcceptanceTests
}
