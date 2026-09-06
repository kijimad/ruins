package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/dungeon"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDriving_運転中セーブは降車状態で復元される は、運転の一時状態が保存されず、ロード後は
// 歩き状態で再開されることを固定する。運転中は入力ゲートでセーブメニューを開けないので手動での
// 運転中セーブは起きないが、Driving を skipComponents で外す方針を往復で担保する。
func TestDriving_運転中セーブは降車状態で復元される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	manager, err := save.NewSerializationManager(save.WithSaveDir(t.TempDir()))
	require.NoError(t, err)
	require.NoError(t, manager.SaveWorld(world, "drive"))
	newWorld := testutil.InitTestWorld(t)
	require.NoError(t, manager.LoadWorld(newWorld, "drive"))

	np, err := query.GetPlayerEntity(newWorld)
	require.NoError(t, err)
	assert.False(t, newWorld.Components.Driving.Has(np), "運転の一時状態は保存されず降車状態で復元される")
	assert.True(t, newWorld.Components.GridElement.Has(np), "プレイヤーは座標を持ち歩ける")
}

// TestNewCubeMenuState_4項目を並べる はキューブメニューが収納・オークション・情報・閉じるの
// 4項目を並べることを検証する。
func TestNewCubeMenuState_4項目を並べる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 5, Y: 5})
	require.NoError(t, err)

	st, err := NewCubeMenuState(cube)
	require.NoError(t, err)
	menu, ok := st.(*ChoiceMenuState)
	require.True(t, ok, "キューブメニューは ChoiceMenu ベース")

	_, choices := menu.provide(world)
	assert.Len(t, choices, 4, "収納・オークション・キューブ情報・閉じるの4項目")
}

// TestOverworldMapState_キューブのチャンク位置を出す は大域地図にキューブのチャンク位置が
// マーカーとして載ることを検証する。
func TestOverworldMapState_キューブのチャンク位置を出す(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	drv := overworld.NewDriver(mapplanner.PlannerTypeOverworldField, dungeon.NewOverworldDefinition("オーバーワールド", 0, 30, 20, 3, 1), &overworld.NewGameParams{RunSeed: 42})
	require.NoError(t, drv.Start(world)) // プレイヤー近くにキューブを1体スポーンする

	st := &OverworldMapState{}
	require.NoError(t, st.OnStart(world))
	assert.NotEmpty(t, st.cubeCells, "大域地図にキューブのチャンク位置が載る")
}
