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
// 再構築できることを固定する。
func TestOverworldState_セーブ往復で帯状態が復元される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20
	const k = 3

	world := testutil.InitTestWorld(t)
	factory := NewOverworldState(mapplanner.PlannerTypeOverworldField, &NewGameParams{RunSeed: 12345, ChunkW: chunkW, ChunkH: chunkH, K: k})
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
	require.Equal(t, 1, int(query.GetDungeon(world).SeamlessBand.EastIndex), "永続状態に同期される")

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
	assert.Equal(t, 1, int(sb.EastIndex), "EastIndex が復元される")
	assert.Equal(t, uint64(12345), sb.RunSeed, "RunSeed が復元される")
	assert.Equal(t, chunkW, sb.ChunkW, "ChunkW が復元される")
	assert.Equal(t, k, int(sb.K), "K が復元される")

	// 寒波前線の config が復元される
	assert.True(t, sb.Front.Active, "FrontActive が復元される")
	assert.Equal(t, chunkW*frontColdWidthChunks, sb.Front.ColdWidth, "FrontColdWidth が復元される")
	assert.Equal(t, frontAdvanceTurns, sb.Front.AdvanceTurns, "FrontAdvanceTurns が復元される")
	assert.Equal(t, consts.Tile(frontStep), sb.Front.Step, "FrontStep が復元される")

	// 復元ワールドでロード用ファクトリから OverworldState を起動 → Band が eastIndex=1 で再構築される
	loadFactory := NewOverworldState(mapplanner.PlannerTypeOverworldField, nil)
	loadState, err := loadFactory()
	require.NoError(t, err)
	ow2, ok := loadState.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, ow2.OnStart(world2))

	assert.Equal(t, 1, int(ow2.band.EastIndex()), "ロード復元で Band が eastIndex=1 で再構築される")
	// 帯タイルは serde で復元済み（再生成していない）ことの傍証: Level 幅が帯全幅のまま
	assert.Equal(t, chunkW*k, query.GetDungeon(world2).Level.TileWidth, "帯全幅の Level が保たれる")

	assert.True(t, ow2.frontCfg.AdvanceTurns == frontAdvanceTurns && ow2.frontCfg.Step == frontStep,
		"ロード復元で寒波前線 config も再構築される")

	// 復元ワールドに帯タイルが存在する（serde 復元）
	count := 0
	tileQuery := ecs.NewFilter1[gc.GridElement](world2.ECS).Query()
	for tileQuery.Next() {
		count++
	}
	assert.Positive(t, count, "帯タイルが serde で復元されている")
}

// TestOverworldState_前線が総ターン数で前進する は、寒波前線の現在位置が GameTime.TotalTurns
// から決定的に導出され、ターン経過で東へ進み、SeamlessBand.Front.EastAbsX に反映されることを固定する。
func TestOverworldState_前線が総ターン数で前進する(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 40, 20

	world := testutil.InitTestWorld(t)
	factory := NewOverworldState(mapplanner.PlannerTypeOverworldField, &NewGameParams{RunSeed: 1, ChunkW: chunkW, ChunkH: chunkH, K: 3})
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	d := query.GetDungeon(world)
	// 開始時（TotalTurns=0）は StartEast のまま。StartEast = bandOriginX(0) + chunkW = +chunkW（西チャンク東端）
	d.GameTime.TotalTurns = 0
	st.updateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW), d.SeamlessBand.Front.EastAbsX, "0ターンは開始位置 +chunkW（西チャンク東端）")

	// frontAdvanceTurns ごとに frontStep 前進する。AdvanceTurns 未満は動かない
	d.GameTime.TotalTurns = frontAdvanceTurns - 1
	st.updateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW), d.SeamlessBand.Front.EastAbsX, "AdvanceTurns 未満は前進しない")

	d.GameTime.TotalTurns = frontAdvanceTurns
	st.updateFront(world)
	assert.Equal(t, consts.AbsTileX(chunkW)+consts.AbsTileX(frontStep), d.SeamlessBand.Front.EastAbsX, "AdvanceTurns で 1 段前進する")

	// 決定的: 同じターン数なら同じ位置
	before := d.SeamlessBand.Front.EastAbsX
	st.updateFront(world)
	assert.Equal(t, before, d.SeamlessBand.Front.EastAbsX, "冪等（導出値）")
}
