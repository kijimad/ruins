package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverworldState_セーブ往復で帯状態が復元される は、シームレスワールドの帯状態
// （SeamlessBand）が serde に乗り、ロード後に OverworldState が同じ eastIndex で
// 再構築できることを固定する（B3: セーブ有界化の基盤）。
func TestOverworldState_セーブ往復で帯状態が復元される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20
	const k = 3

	world := testutil.InitTestWorld(t)
	factory := NewOverworldState(12345, chunkW, chunkH, k, mapplanner.PlannerTypeOverworldField)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	// 東へ1回シフトして eastIndex=1 にする
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 2 * chunkW
	require.NoError(t, st.maybeShift(world))
	require.Equal(t, 1, int(st.band.EastIndex()))
	require.Equal(t, 1, query.GetDungeon(world).SeamlessBand.EastIndex, "永続状態に同期される")

	// セーブ往復（メモリ内）
	sm, err := save.NewSerializationManager()
	require.NoError(t, err)
	jsonData, err := sm.GenerateWorldJSON(world)
	require.NoError(t, err)

	world2 := testutil.InitTestWorld(t)
	require.NoError(t, sm.RestoreWorldFromJSON(world2, jsonData))

	// SeamlessBand が復元されている
	sb := query.GetDungeon(world2).SeamlessBand
	assert.True(t, sb.Active, "Active が復元される")
	assert.Equal(t, 1, sb.EastIndex, "EastIndex が復元される")
	assert.Equal(t, uint64(12345), sb.RunSeed, "RunSeed が復元される")
	assert.Equal(t, chunkW, sb.ChunkW, "ChunkW が復元される")
	assert.Equal(t, k, sb.K, "K が復元される")

	// 復元ワールドでロード用ファクトリから OverworldState を起動 → Band が eastIndex=1 で再構築される
	loadFactory := NewOverworldStateForLoad(mapplanner.PlannerTypeOverworldField)
	loadState, err := loadFactory()
	require.NoError(t, err)
	ow2, ok := loadState.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, ow2.OnStart(world2))

	assert.Equal(t, 1, int(ow2.band.EastIndex()), "ロード復元で Band が eastIndex=1 で再構築される")
	// 帯タイルは serde で復元済み（再生成していない）ことの傍証: Level 幅が帯全幅のまま
	assert.Equal(t, chunkW*k, query.GetDungeon(world2).Level.TileWidth, "帯全幅の Level が保たれる")

	// 復元ワールドに帯タイルが存在する（serde 復元）
	count := 0
	tileQuery := ecs.NewFilter1[gc.GridElement](world2.ECS).Query()
	for tileQuery.Next() {
		count++
	}
	assert.Positive(t, count, "帯タイルが serde で復元されている")
}
