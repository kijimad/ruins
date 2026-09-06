package systems

import (
	"math/rand/v2"
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// runCoordFrame はDungeonState.Updateのシステム実行順序を模擬する
func runCoordFrame(world w.World) error {
	deadSys := &DeadCleanupSystem{}
	if err := deadSys.Update(world); err != nil {
		return err
	}
	turnSys := &TurnSystem{}
	return turnSys.Update(world)
}

// fieldItemNames はフィールドに落ちているアイテム名の一覧を返す
func fieldItemNames(world w.World) []string {
	var names []string
	q := ecs.NewFilter1[gc.LocationOnField](world.ECS).Query()
	for q.Next() {
		names = append(names, query.GetEntityName(q.Entity(), world))
	}
	return names
}

// TestTurnSystem_分解を完走してもAPが枯渇しない は、継続アクティビティの
// 毎ターンのAP消費がターン終了の回復と均衡し、完了時に大きな負債が
// 残らないことを検証する
func TestTurnSystem_分解を完走してもAPが枯渇しない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Resources.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	disComp := activity.NewDisassembleActivity(crate, player, world)
	result, err := activity.Execute(disComp, player, world)
	require.NoError(t, err)
	require.True(t, result.Success)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 継続アクティビティは1フレームで上限ぶんのターンを早送りし、超過分は次フレームへ持ち越す。
	// 完走までフレームを重ねる。余裕を持たせたフレーム数で完走を待つ
	for i := 0; i < 300 && query.HasActivity(world, player); i++ {
		require.NoError(t, runCoordFrame(world))
	}

	require.False(t, query.HasActivity(world, player), "分解が完了するべき")
	assert.False(t, world.ECS.Alive(crate), "分解したpropは消えるべき")
	assert.Contains(t, fieldItemNames(world), "Hardwood", "確定枠の産出が足元に落ちるべき")

	// 毎ターンの消費100はターン終了の回復とおおむね均衡する。
	// 大きな負債はアクティビティの複数ステップが1ターン内で走った兆候になる
	turnBased := world.Components.TurnBased.Get(player)
	assert.Greater(t, turnBased.AP.Current, -200,
		"完了時のAPは毎ターンの回復と均衡し、大きな負債にならないべき")
}
