package states

import (
	"slices"
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpawnDebugStageCrateItems_木箱にモジュールと装着アイテムと道具を入れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テンプレートが置く木箱を模して wooden_crate を1つ置く
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 5, 5)
	require.NoError(t, err)

	require.NoError(t, spawnDebugStageCrateItems(world))

	items := query.GetStorageItems(world, crate)
	got := make([]string, 0, len(items))
	for _, e := range items {
		got = append(got, query.GetEntityID(e, world))
	}
	want := slices.Repeat([]string{"cube_range_module"}, debugStageModuleCount)
	for _, id := range debugStageFacilityItems {
		want = append(want, slices.Repeat([]string{id}, debugStageFacilityCount)...)
	}
	want = append(want, "pickaxe")
	assert.ElementsMatch(t, want, got)
}

func TestSpawnDebugStageCrateItems_木箱が無ければエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 木箱はテンプレートが必ず置く。無いのは退行なので握りつぶさず error を返す
	require.Error(t, spawnDebugStageCrateItems(world))
}
