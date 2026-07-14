package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/route"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addBackpackLoot は背嚢に重量・価値を持つ戦利品を1つ足す。
func addBackpackLoot(world w.World, kg float64, value int) {
	e := world.ECS.NewEntity()
	world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{})
	world.Components.Weight.Add(e, &gc.Weight{Kg: kg})
	world.Components.Value.Add(e, &gc.Value{Value: value})
	world.Components.Name.Add(e, &gc.Name{Name: "戦利品"})
}

// TestMacroMap_LootFeedsSupply は潜行の戦利品が積載重量と供給に帰結すること（micro→macro 密結合）を検証する。
func TestMacroMap_LootFeedsSupply(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.SetCaravanRun(world, gc.NewCaravanRun(1, route.ExpeditionDeepVault))

	// 潜行前の背嚢価値を控えた状態から潜行して戻った、という状況を作る
	st := &MacroMapState{divingRuin: true, cargoValueBeforeDive: 0}
	addBackpackLoot(world, 6, 80) // 重量6・価値80 の戦利品を得た

	require.NoError(t, st.OnResume(world))

	run := query.GetCaravanRun(world)
	require.NotNil(t, run)
	assert.Equal(t, route.Weight(6), run.Supply.Cargo, "戦利品が積載重量になる")
	assert.Equal(t, 100+80/8, run.Supply.Food, "稼いだ価値の一部を糧食として回収")
	assert.Equal(t, 50+80/16, run.Supply.Fuel, "稼いだ価値の一部を燃料として回収")
	assert.False(t, st.divingRuin, "回収後は潜行フラグが下りる")
}

// TestMacroMap_ReturnUpdatesCargoOnly は潜行以外の帰還では供給回収せず積載だけ更新することを検証する。
func TestMacroMap_ReturnUpdatesCargoOnly(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.SetCaravanRun(world, gc.NewCaravanRun(1, route.ExpeditionDeepVault))

	st := &MacroMapState{divingRuin: false}
	addBackpackLoot(world, 4, 30)

	require.NoError(t, st.OnResume(world))

	run := query.GetCaravanRun(world)
	assert.Equal(t, route.Weight(4), run.Supply.Cargo, "積載は更新される")
	assert.Equal(t, 100, run.Supply.Food, "潜行以外では糧食は回収されない")
}
