package mapspawner

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpawn_デバッグの町の木箱にモジュールと装着アイテムと道具が入る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	pt := mapplanner.LayoutPlannerType("debug_town")
	pt.Danger = 1
	plan, err := mapplanner.Plan(world, 0, 0, 1, pt)
	require.NoError(t, err)
	_, err = Spawn(world, plan)
	require.NoError(t, err)

	var crates []ecs.Entity
	q := query.ActiveFilter2[gc.RawID, gc.GridElement](world).Query()
	for q.Next() {
		if world.Components.RawID.Get(q.Entity()).ID == "debug_crate" {
			crates = append(crates, q.Entity())
		}
	}
	require.Len(t, crates, 1, "絵がデバッグの木箱を1つ置く")

	items := query.GetStorageItems(world, crates[0])
	got := make([]string, 0, len(items))
	for _, e := range items {
		got = append(got, query.GetEntityID(e, world))
	}
	want := slices.Concat(
		[]string{"pickaxe"},
		slices.Repeat([]string{"cube_range_module", "cube_fuel_module"}, consts.CubeModuleSlots),
		slices.Repeat([]string{"deployable_storage", "deployable_bed", "deployable_workbench"}, 2),
	)
	assert.ElementsMatch(t, want, got)
}
