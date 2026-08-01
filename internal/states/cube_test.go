package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnterExitCube_内装へ入り元の位置へ戻る はキューブ内装への出入りを検証する。
// 入るとオーバーワールドが退避しキューブ内装が現ステージになり、出ると元のタイルへ戻る。
func TestEnterExitCube_内装へ入り元の位置へ戻る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	playerPos := consts.Coord[consts.Tile]{X: 4, Y: 5}
	player, err := lifecycle.SpawnPlayer(world, playerPos, "Ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	d := query.GetDungeon(world)
	d.CurrentStage = gc.NewOverworldStage()
	band := addStageEntity(t, world, gc.NewOverworldStage())

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}

	// 入る: オーバーワールドを退避し、キューブ内装が現ステージになる
	require.NoError(t, st.enterCube(world, cube))
	interiorKey := gc.NewCubeInteriorStage()
	assert.Equal(t, interiorKey, d.CurrentStage, "現ステージはキューブ内装")
	assert.True(t, world.Components.Suspended.Has(band), "オーバーワールドは退避される")

	// 出口 prop の戻り先は入場時のプレイヤータイル
	exitProp, _, ok := findPortal(world, gc.InteractionExitCube)
	require.True(t, ok, "内装に出口がある")
	require.True(t, world.Components.PortalConnection.Has(exitProp))
	assert.Equal(t, playerPos, world.Components.PortalConnection.Get(exitProp).Coord, "戻り先は入場時のプレイヤータイル")

	// 据えたランタンの重量が総重量に乗り、押しコストが基準より重くなる
	// これがブレーキと引力の対。置いた物が押しを重くする
	assert.Positive(t, query.CubeWeight(world, interiorKey), "内装のランタンが総重量に乗る")
	assert.Greater(t, query.PushCost(query.CubeWeight(world, interiorKey)), consts.PushCostBase, "物を置くと空のキューブより押しが重い")

	// 内装は1階層。降り/上りの階段ポータルは無い。あると降りて panic するため
	_, _, hasNext := findPortal(world, gc.InteractionPortalNext)
	assert.False(t, hasNext, "内装に降り階段は無い")
	_, _, hasPrev := findPortal(world, gc.InteractionPortalPrev)
	assert.False(t, hasPrev, "内装に上り階段は無い")

	// 出る: オーバーワールドが再稼働し、入場した元タイルへ戻る
	require.NoError(t, st.exitCube(world))
	assert.Equal(t, gc.NewOverworldStage(), d.CurrentStage, "オーバーワールドへ戻る")
	assert.False(t, world.Components.Suspended.Has(band), "オーバーワールドが再稼働する")
	assert.Equal(t, playerPos, world.Components.GridElement.Get(player).Coord, "入場した元タイルへ戻る")
}

// TestEnterCube_内装で保存して読み込んでも落ちない はキューブ内装での serde 往復を検証する。
// 内装は定義を持たないランタイムステージなので、復帰時に遺跡定義として解決しようとして落ちる
// 不具合の回帰。復帰では世界が復元済みで定義解決は不要にする。
func TestEnterCube_内装で保存して読み込んでも落ちない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 4, Y: 5}, "Ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	query.GetDungeon(world).CurrentStage = gc.NewOverworldStage()
	addStageEntity(t, world, gc.NewOverworldStage())

	st := &DungeonState{DefinitionName: dungeon.DungeonOverworld.Name()}
	require.NoError(t, st.enterCube(world, cube))
	interiorKey := gc.NewCubeInteriorStage()
	require.Equal(t, interiorKey, query.GetDungeon(world).CurrentStage)

	// 内装にいる状態で保存して別 world へ読み込む
	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "cube_interior"))
	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "cube_interior"))

	assert.Equal(t, interiorKey, query.GetDungeon(newWorld).CurrentStage, "内装が現ステージのまま復元される")

	// 復帰状態を組み立てて開始する。以前はここで定義解決に失敗して落ちていた
	resume, err := newResumeStateFactory(newWorld)()
	require.NoError(t, err)
	require.NoError(t, resume.OnStart(newWorld), "内装で読み込んでも定義解決で落ちない")
}
