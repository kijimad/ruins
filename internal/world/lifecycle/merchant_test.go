package lifecycle

import (
	"testing"

	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// merchantStockTotal は raw の merchant の行の在庫の総個数と品種の数を返す
func merchantStockTotal(t *testing.T, world w.World) (total, kinds int) {
	t.Helper()
	member, err := raw.FindMember(world.Resources.RawMaster, "merchant")
	require.NoError(t, err)
	for _, item := range raw.PtrSlice(member.Stock) {
		total += item.Count
	}
	return total, len(raw.PtrSlice(member.Stock))
}

func TestPopulateStock_在庫にアイテムを積む(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	merchant := world.ECS.NewEntity()

	require.NoError(t, PopulateStock(world, merchant, "merchant"))

	total, kinds := merchantStockTotal(t, world)
	assert.Len(t, query.GetStorageItems(world, merchant), total, "1個1エンティティで行の総個数だけ積む")
	assert.Len(t, query.StorageStacks(world, merchant), kinds, "1品種1スタックに束ねられる")
}
