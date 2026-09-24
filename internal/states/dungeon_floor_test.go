package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpawnDebugStageCubeGear_木箱にモジュールと据付アイテムを入れる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// テンプレートが置く木箱を模して wooden_crate を1つ置く
	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 5, 5)
	require.NoError(t, err)

	require.NoError(t, spawnDebugStageCubeGear(world))

	items := query.GetStorageItems(world, crate)
	// モジュール + 据付ストレージ + 据付ランプ + 据付ベッド
	assert.Len(t, items, debugStageModuleCount+debugStageFacilityCount*3, "木箱にモジュールと据付アイテムが入る")
}

func TestSpawnDebugStageCubeGear_木箱が無ければエラー(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 木箱はテンプレートが必ず置く。無いのは退行なので握りつぶさず error を返す
	require.Error(t, spawnDebugStageCubeGear(world))
}
