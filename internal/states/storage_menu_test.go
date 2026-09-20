package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageMenuState_燃料投入は可燃物だけ通し貨物を除く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 燃料。可燃物で貨物印が無い
	fuel := world.ECS.NewEntity()
	world.Components.Material.Add(fuel, &gc.Material{Kind: oapi.COAL})
	world.Components.Weight.Add(fuel, &gc.Weight{Milligram: consts.MilligramPerKg})

	// 貨物。可燃物だが Stowed が付くので燃料投入には出さない
	cargo := world.ECS.NewEntity()
	world.Components.Material.Add(cargo, &gc.Material{Kind: oapi.COAL})
	world.Components.Weight.Add(cargo, &gc.Weight{Milligram: consts.MilligramPerKg})
	world.Components.Stowed.Add(cargo, &gc.Stowed{})

	// 非可燃物。Material を持たないので燃焼熱量は0
	stone := world.ECS.NewEntity()

	st := &StorageMenuState{itemFilter: isFuelItem}
	filtered := st.filterStacks(world, []query.Stack{
		{Rep: fuel, Count: 1}, {Rep: cargo, Count: 1}, {Rep: stone, Count: 1},
	})

	require.Len(t, filtered, 1, "燃料だけ残す。貨物と非可燃物は除く")
	assert.Equal(t, fuel, filtered[0].Rep)
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
