package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/shibukawa/snapsql"
	"github.com/shibukawa/snapsql/explain"
)

func buildTableMetadataFromConfig(tables map[string]snapsql.TablePerformance) map[string]explain.TableMetadata {
	if len(tables) == 0 {
		return nil
	}

	meta := make(map[string]explain.TableMetadata, len(tables))
	for rawKey, value := range tables {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if key == "" {
			continue
		}

		converted := explain.TableMetadata{
			ExpectedRows:  value.ExpectedRows,
			AllowFullScan: value.AllowFullScan,
		}

		meta[key] = converted

		if parts := strings.SplitN(key, ".", 2); len(parts) == 2 && parts[1] != "" {
			if _, exists := meta[parts[1]]; !exists {
				meta[parts[1]] = converted
			}
		}
	}

	if len(meta) == 0 {
		return nil
	}

	return meta
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d/time.Microsecond)
	}

	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d)/float64(time.Millisecond))
	}

	return d.Round(time.Millisecond).String()
}
