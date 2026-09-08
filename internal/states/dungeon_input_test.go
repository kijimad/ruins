package states

import (
	"strings"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
	"github.com/kijimaD/ruins/internal/save"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDungeonState_DoAction_拾得は継続アクティビティ中に中断しない は ActionPickup が
// ターンと継続アクティビティのゲートを通ることを固定する。拾得はターンを消費しうる実ゲーム
// アクションなので、継続アクティビティ中は HasActivity で塞がれ、既存アクティビティを
// 中断してはいけない
func TestDungeonState_DoAction_拾得は継続アクティビティ中に中断しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	query.GetTurnState(world).Phase = gc.TurnPhasePlayer

	// 読書や休息に相当する継続アクティビティを実行中にする
	world.Components.Activity.Add(player, &gc.Activity{BehaviorName: gc.BehaviorRest, State: gc.ActivityStateRunning})
	require.True(t, query.CanPlayerAct(world), "前提: プレイヤーは行動可能なターン")
	require.True(t, query.HasActivity(world, player), "前提: 継続アクティビティ中")

	st := &DungeonState{}
	tr, err := st.DoAction(world, inputmapper.ActionPickup)
	require.NoError(t, err)

	assert.Equal(t, es.TransNone, tr.Type)
	assert.True(t, query.HasActivity(world, player), "拾得は継続アクティビティを中断しない")
}

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

// TestDismount_直上に残る は降車でプレイヤーがキューブの直上に残り Driving が外れることを固定する。
func TestDismount_直上に残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 7, Y: 7}, "ash")
	require.NoError(t, err)
	cube, err := lifecycle.SpawnCube(world, consts.Coord[consts.Tile]{X: 7, Y: 7})
	require.NoError(t, err)
	world.Components.Driving.Add(player, &gc.Driving{Vehicle: cube})

	(&DungeonState{}).dismount(world)

	assert.False(t, world.Components.Driving.Has(player), "降車で Driving が外れる")
	assert.Equal(t, consts.Coord[consts.Tile]{X: 7, Y: 7}, world.Components.GridElement.Get(player).Coord, "プレイヤーはキューブの直上に残る")

	var logged bool
	for _, e := range query.GetGameLog(world).GetRecentEntries(10) {
		if strings.Contains(e.Text(), "get off the cube") {
			logged = true
		}
	}
	assert.True(t, logged, "降車をゲームログに出す")
}
