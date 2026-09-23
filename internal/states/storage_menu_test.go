package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageMenuState_燃料投入は可燃物だけ通す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 燃料。可燃物
	fuel := world.ECS.NewEntity()
	world.Components.Material.Add(fuel, &gc.Material{Kind: oapi.COAL})
	world.Components.Weight.Add(fuel, &gc.Weight{Milligram: consts.MilligramPerKg})

	// 非可燃物。Material を持たないので燃焼熱量は0
	stone := world.ECS.NewEntity()

	st := &StorageMenuState{itemFilter: isFuelItem}
	filtered := st.filterStacks(world, []query.Stack{
		{Rep: fuel, Count: 1}, {Rep: stone, Count: 1},
	})

	require.Len(t, filtered, 1, "可燃物だけ残す。貨物は別ロケーションなので燃料タンクには現れない")
	assert.Equal(t, fuel, filtered[0].Rep)
}

func TestStorageMenuState_熱量列はshowHeatのときだけ埋まる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// COAL は 800/kg。1kg を2個で束の総熱量は 800 × 2 = 1600
	fuel := world.ECS.NewEntity()
	world.Components.Material.Add(fuel, &gc.Material{Kind: oapi.COAL})
	world.Components.Weight.Add(fuel, &gc.Weight{Milligram: consts.MilligramPerKg})
	stack := []query.Stack{{Rep: fuel, Count: 2}}

	off := (&StorageMenuState{}).toStorageItemData(world, stack)
	require.Len(t, off, 1)
	assert.Empty(t, off[0].Heat, "showHeat 未指定は熱量列を出さない")

	on := (&StorageMenuState{showHeat: true}).toStorageItemData(world, stack)
	require.Len(t, on, 1)
	assert.Equal(t, consts.Heat(1600).String(), on[0].Heat, "showHeat は束の総熱量を出す")
}

func TestStorageMenuState_storeOnlyは投入タブだけ返す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 1, Y: 1}, "ash")
	require.NoError(t, err)
	storage := world.ECS.NewEntity()

	only := &StorageMenuState{storageEntity: storage, storeOnly: true}
	props, err := only.Fetch(world)
	require.NoError(t, err)
	require.Len(t, props.Tabs, 1, "取り出しタブを隠し投入タブだけにする")
	assert.Equal(t, tabIDStore, props.Tabs[0].ID)

	both := &StorageMenuState{storageEntity: storage}
	props2, err := both.Fetch(world)
	require.NoError(t, err)
	require.Len(t, props2.Tabs, 2, "既定は取り出し・投入の両タブ")
	assert.Equal(t, tabIDRetrieve, props2.Tabs[0].ID)
}

func TestStorageMenuState_フィルタ未指定なら全て通す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	a := world.ECS.NewEntity()
	b := world.ECS.NewEntity()

	st := &StorageMenuState{}
	filtered := st.filterStacks(world, []query.Stack{{Rep: a, Count: 1}, {Rep: b, Count: 1}})

	assert.Len(t, filtered, 2, "フィルタ未指定は素通しする")
}
