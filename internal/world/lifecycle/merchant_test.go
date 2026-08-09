package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPopulateMerchantStock_在庫にアイテムと隊員候補を積む は、商人の在庫がアイテムと隊員候補で
// 構成され、アイテムは定義数、候補は3〜5人で名前が重複しないことを固定する。
func TestPopulateMerchantStock_在庫にアイテムと隊員候補を積む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	require.NoError(t, PopulateMerchantStock(world, merchant, world.Config.RNG))

	stock := query.GetStorageItems(world, merchant)

	var recruits, items int
	names := map[string]bool{}
	for _, e := range stock {
		if world.Components.Abilities.Has(e) {
			recruits++
			name := world.Components.Name.Get(e).Name
			assert.False(t, names[name], "候補名は重複しない: %s", name)
			names[name] = true
			// 候補はフィールドに出ない inert 実体
			assert.False(t, world.Components.GridElement.Has(e), "候補は座標を持たない")
		} else {
			items++
		}
	}

	assert.Equal(t, len(merchantStockItems), items, "アイテムは定義数だけ積む")
	assert.GreaterOrEqual(t, recruits, 3, "候補は3人以上")
	assert.LessOrEqual(t, recruits, 5, "候補は5人以下")
}
