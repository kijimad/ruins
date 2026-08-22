package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPopulateMerchantStock_在庫にアイテムを積む は、商人の在庫が定義の総個数の実体で
// 構成され、複数個の品が1スタックに束ねられることを固定する。
func TestPopulateMerchantStock_在庫にアイテムを積む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	require.NoError(t, PopulateMerchantStock(world, merchant, world.Resources.Config.RNG))

	total := 0
	for _, item := range merchantStockItems {
		total += item.Count
	}
	stock := query.GetStorageItems(world, merchant)
	assert.Len(t, stock, total, "1個1エンティティで定義の総個数だけ積む")

	// 複数個の品は店頭で1スタックに束ねられる
	stacks := query.StorageStacks(world, merchant)
	assert.Len(t, stacks, len(merchantStockItems), "1品種1スタックに束ねられる")
}
