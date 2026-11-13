package query_test

import (
	"testing"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type disassemblyResult struct {
	id    string
	count int
	ok    bool
}

func TestItemDisassemblyYield(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		item string
		want disassemblyResult
	}{
		{"金属は重さ1kgごとに鉄くず1個", "hammer", disassemblyResult{"scrap_iron", 4, true}},
		{"装着する設備は分解できない", "deployable_workbench", disassemblyResult{"", 0, false}},
		{"戻し先の素材そのものは分解できない", "scrap_iron", disassemblyResult{"", 0, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			world := testutil.InitTestWorld(t)
			_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
			require.NoError(t, err)
			item, err := lifecycle.SpawnBackpackItem(world, tt.item, 1)
			require.NoError(t, err)
			spec, err := raw.NewItemSpec(world.Resources.RawMaster, tt.item)
			require.NoError(t, err)

			id, count, ok := query.ItemDisassemblyYield(world, item)
			assert.Equal(t, tt.want, disassemblyResult{id, count, ok}, "実体から導く")
			id, count, ok = query.SpecDisassemblyYield(world, spec)
			assert.Equal(t, tt.want, disassemblyResult{id, count, ok}, "spec から導く")
		})
	}
}
