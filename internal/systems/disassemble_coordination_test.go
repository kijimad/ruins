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

// TestTurnSystem_分解を完走してもAPが枯渇しない は、継続アクティビティが
// ターン交互処理で進み、旧方式のようなAP前借りの借金が発生しないことを検証する
func TestTurnSystem_分解を完走してもAPが枯渇しない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	result, err := activity.Execute(&activity.DisassembleActivity{Target: crate}, player, world)
	require.NoError(t, err)
	require.True(t, result.Success)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 1ステップ=1ゲームターン、1ターン=Player/AI/Endの3フレームで進む。
	// 20ターンの分解に余裕を持たせたフレーム数で完走を待つ
	for i := 0; i < 300 && query.HasActivity(world, player); i++ {
		require.NoError(t, runCoordFrame(world))
	}

	require.False(t, query.HasActivity(world, player), "分解が完了するべき")
	assert.False(t, world.ECS.Alive(crate), "分解したpropは消えるべき")
	assert.Contains(t, fieldItemNames(world), "硬木", "確定枠の産出が足元に落ちるべき")

	// 旧方式では完了時点でAPが約-2000の借金になっていた。
	// ターン交互処理では毎ターン消費と回復が均衡し、大きな負債にならない
	turnBased := world.Components.TurnBased.Get(player)
	assert.Greater(t, turnBased.AP.Current, -200,
		"AP前借りの借金が発生しないべき。旧方式ではここが約-2000になる")
}

// TestTurnSystem_分解中も隊員がターンごとに行動する は、プレイヤーの
// 継続アクティビティ中にAIフェーズが回り、隊員が行動していることを検証する
func TestTurnSystem_分解中も隊員がターンごとに行動する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)
	_, err = lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	_, err = activity.Execute(&activity.DisassembleActivity{Target: crate}, player, world)
	require.NoError(t, err)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 数ターン分だけ回す。分解はまだ完了しない
	for range 9 {
		require.NoError(t, runCoordFrame(world))
	}

	require.True(t, query.HasActivity(world, player), "分解はまだ継続中のはず")
	assert.True(t, world.Components.LastActivity.Has(member),
		"分解中もAIフェーズが回り、隊員が行動しているべき")
}

// TestTurnSystem_隊員が分解産出を拾いに来る は、分解完了後に湧いた産出を
// 隊員の拾得AIが回収する一連の協調を検証する
func TestTurnSystem_隊員が分解産出を拾いに来る(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	world.Config.RNG = rand.New(rand.NewPCG(7, 0))

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)
	member, err := lifecycle.SpawnDefaultSquadMember(world, player)
	require.NoError(t, err)
	require.Equal(t, gc.PolicyPickup, world.Components.SquadAI.Get(member).ItemPickup,
		"前提: 隊員の拾得ポリシーは既定で有効")

	_, err = lifecycle.SpawnBackpackItem(world, "モンキーレンチ", 1)
	require.NoError(t, err)
	crate, err := lifecycle.SpawnProp(world, "crate", 6, 5)
	require.NoError(t, err)

	_, err = activity.Execute(&activity.DisassembleActivity{Target: crate}, player, world)
	require.NoError(t, err)

	turnState := query.GetTurnState(world)
	turnState.Phase = gc.TurnPhasePlayer

	// 分解完走まで回す
	for i := 0; i < 300 && query.HasActivity(world, player); i++ {
		require.NoError(t, runCoordFrame(world))
	}
	require.False(t, query.HasActivity(world, player), "分解が完了するべき")
	require.Contains(t, fieldItemNames(world), "硬木", "完了直後は産出がフィールドにあるべき")

	// 隊員が産出へ移動し拾い終えるまで回す。完了後の Player フェーズは入力待ちに
	// なるため、プレイヤーがターンを送るだけの操作を模擬して世界を進める
	picked := false
	for range 600 {
		if turnState.Phase == gc.TurnPhasePlayer && !query.HasActivity(world, player) {
			turnState.Phase = gc.TurnPhaseAI
		}
		require.NoError(t, runCoordFrame(world))
		names := fieldItemNames(world)
		found := slices.Contains(names, "硬木")
		if !found {
			picked = true
			break
		}
	}

	require.True(t, picked, "隊員が産出を拾い終えるべき")
	assert.Contains(t, backpackItemNames(world), "硬木",
		"拾った産出は誰かの所持品に入っているべき")
}
