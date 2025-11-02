package main

import (
	"github.com/shibukawa/snapsql"
)

// Note: backward-compatibility type aliases were removed. Use snapsql.Config and
// other types from the snapsql package directly in new code.

// LoadConfig loads configuration from the specified file
func LoadConfig(configPath string) (*snapsql.Config, error) {
	return snapsql.LoadConfig(configPath)
}
