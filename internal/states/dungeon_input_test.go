package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	es "github.com/kijimaD/ruins/internal/engine/states"
	"github.com/kijimaD/ruins/internal/inputmapper"
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
