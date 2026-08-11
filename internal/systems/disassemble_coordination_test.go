package systems

import (
	"math/rand/v2"
	"slices"
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

// backpackItemNames は所持品にあるアイテム名の一覧を返す。所有者は問わない
func backpackItemNames(world w.World) []string {
	var names []string
	q := ecs.NewFilter1[gc.LocationInBackpack](world.ECS).Query()
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
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

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

	// 継続アクティビティは1フレームで複数ターン早送りされるので、通常は1フレームで完走する。
	// 上限に達しても続きが進むよう、余裕を持たせたフレーム数で完走を待つ
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

// TestTurnSystem_分解中も隊員がターンごとに行動する は、プレイヤーの
// 継続アクティビティ中にAIフェーズが回り、隊員が行動していることを検証する
func TestTurnSystem_分解中も隊員がターンごとに行動する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	disComp := activity.NewDisassembleActivity(crate, player, world)
	_, err = activity.Execute(disComp, player, world)
	require.NoError(t, err)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 分解が完了するまで進める。継続アクティビティ中も各ターンで AI フェーズが回るので、
	// 早送りで一気に進んでも、その間に隊員は行動している
	for i := 0; i < 300 && query.HasActivity(world, player); i++ {
		require.NoError(t, runCoordFrame(world))
	}

	require.False(t, query.HasActivity(world, player), "分解が完了する")
	assert.True(t, world.Components.LastActivity.Has(member),
		"分解中もAIフェーズが回り、隊員が行動しているべき")
}

// TestTurnSystem_隊員が分解産出を拾いに来る は、分解完了後に湧いた産出を
// 隊員の拾得AIが回収する一連の協調を検証する
func TestTurnSystem_隊員が分解産出を拾いに来る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)
	require.Equal(t, gc.PolicyPickup, world.Components.SquadAI.Get(member).ItemPickup,
		"前提: 隊員の拾得ポリシーは既定で有効")

	_, err = lifecycle.SpawnBackpackItem(world, "monkey_wrench", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	disComp := activity.NewDisassembleActivity(crate, player, world)
	_, err = activity.Execute(disComp, player, world)
	require.NoError(t, err)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 分解完走まで回す
	for i := 0; i < 300 && query.HasActivity(world, player); i++ {
		require.NoError(t, runCoordFrame(world))
	}
	require.False(t, query.HasActivity(world, player), "分解が完了するべき")
	require.Contains(t, fieldItemNames(world), "Hardwood", "完了直後は産出がフィールドにあるべき")

	// 隊員が産出へ移動し拾い終えるまで回す。完了後の Player フェーズは入力待ちに
	// なるため、プレイヤーがターンを送るだけの操作を模擬して世界を進める
	picked := false
	for range 600 {
		if turnState.Phase == gc.TurnPhasePlayer && !query.HasActivity(world, player) {
			turnState.Phase = gc.TurnPhaseAI
		}
		require.NoError(t, runCoordFrame(world))
		names := fieldItemNames(world)
		found := slices.Contains(names, "Hardwood")
		if !found {
			picked = true
			break
		}
	}

	require.True(t, picked, "隊員が産出を拾い終えるべき")
	assert.Contains(t, backpackItemNames(world), "Hardwood",
		"拾った産出は誰かの所持品に入っているべき")
}
