package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPopulateMerchantStock_在庫にアイテムを積む は、商人の在庫が定義数のアイテムで
// 構成されることを固定する。
func TestPopulateMerchantStock_在庫にアイテムを積む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	require.NoError(t, PopulateMerchantStock(world, merchant, world.Config.RNG))

	stock := query.GetStorageItems(world, merchant)
	assert.Len(t, stock, len(merchantStockItems), "アイテムは定義数だけ積む")
}
