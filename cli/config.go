package cli

import (
	"github.com/shibukawa/snapsql"
)

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*snapsql.Config, error) {
	return snapsql.LoadConfig(configPath)
}
