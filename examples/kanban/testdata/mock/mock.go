package mock

import (
	"embed"
	"sync"

	"github.com/shibukawa/snapsql/langs/snapsqlgo"
)

//go:embed board_archive.json board_create.json board_get.json board_list.json board_tree.json card_comment_create.json card_comment_list.json card_create.json card_move.json card_postpone.json card_reorder.json card_update.json list_archive.json list_create.json list_rename.json list_reorder.json
var embeddedFiles embed.FS

var (
	providerOnce sync.Once
	provider     snapsqlgo.MockProvider
)

// Provider exposes the embedded mock data as a snapsqlgo.MockProvider.
func Provider() snapsqlgo.MockProvider {
	providerOnce.Do(func() {
		provider = snapsqlgo.NewEmbeddedMockProvider(embeddedFiles)
	})
	return provider
}
