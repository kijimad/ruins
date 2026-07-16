package aiinput

import (
	"math/rand/v2"
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateMachine_Hostile(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatAttack,
		CombatCurrent:         gc.CombatAttack,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// 1ターン目：まだ待機継続
	rp.updateState(solo, false, 2)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "1ターン経過時は待機継続")

	// 3ターン目：待機時間終了で移動状態へ
	rp.updateState(solo, false, 3)
	assert.Equal(t, gc.AIStateDriving, solo.SubState, "待機時間終了で移動状態へ遷移")

	// プレイヤー発見で追跡状態へ
	rp.updateState(solo, true, 4)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "Hostileはプレイヤー発見で追跡状態へ遷移")
}

func TestStateMachine_Neutral(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 2,
	}

	// プレイヤーを発見しても追跡しない
	rp.updateState(solo, true, 2)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "Neutralはプレイヤーを見ても追跡しない")
}

func TestStateMachine_Cowardly(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// プレイヤー発見で逃亡状態へ
	rp.updateState(solo, true, 2)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState, "Cowardlyはプレイヤー発見で逃亡状態へ遷移")
}

func TestStateMachine_Fleeing_Recovery(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateFleeing,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 3,
	}

	// 逃亡中にプレイヤーが見えている間は逃亡継続
	rp.updateState(solo, true, 2)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState, "プレイヤーが見えている間は逃亡継続")

	// プレイヤーを見失い、逃亡時間終了でデフォルトに復帰
	solo.StartSubStateTurn = 1
	rp.updateState(solo, false, 5)
	assert.Equal(t, gc.AIStateDriving, solo.SubState, "プレイヤーを見失い逃亡時間終了で移動へ")
	assert.Equal(t, gc.CombatEvade, solo.CombatCurrent, "デフォルト態度に復帰")
}

func TestStateMachine_NeutralToHostile_StartChasing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatIgnore,
		CombatCurrent:         gc.CombatIgnore,
		SubState:              gc.AIStateDriving,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	// Neutralはプレイヤーを見ても追跡しない
	rp.updateState(solo, true, 2)
	assert.Equal(t, gc.AIStateDriving, solo.SubState, "Neutralは追跡しない")

	// 被ダメージでCombatCurrentがCombatAttackに変化した（ReactToHostile相当）
	solo.CombatCurrent = gc.CombatAttack

	// 次のターンでプレイヤーを見たら追跡を開始する
	rp.updateState(solo, true, 3)
	assert.Equal(t, gc.AIStateChasing, solo.SubState, "Hostile化後はプレイヤー発見で追跡開始")
}

func TestStateMachine_CowardlyToFleeing_StartFleeing(t *testing.T) {
	t.Parallel()

	rp := newSoloPlanner(newTestRNG())

	solo := &gc.SoloAI{
		CombatDefault:         gc.CombatEvade,
		CombatCurrent:         gc.CombatEvade,
		SubState:              gc.AIStateWaiting,
		StartSubStateTurn:     1,
		DurationSubStateTurns: 5,
	}

	rp.updateState(solo, false, 2)
	assert.Equal(t, gc.AIStateWaiting, solo.SubState, "プレイヤーが見えてなければまだ待機")

	rp.updateState(solo, true, 3)
	assert.Equal(t, gc.AIStateFleeing, solo.SubState, "CombatEvadeはプレイヤー発見で逃亡開始")
}

func TestVisionSystem(t *testing.T) {
	t.Parallel()

	vs := NewVisionSystem()
	assert.NotNil(t, vs, "VisionSystemが作成できること")
}

func TestProcessor(t *testing.T) {
	t.Parallel()

	processor := NewProcessor(newTestRNG())
	assert.NotNil(t, processor, "Processorが作成できること")
	assert.NotNil(t, processor.planners[gc.PlannerSolo])
	assert.NotNil(t, processor.planners[gc.PlannerSquad])
}

// containsEntity はスライスにエンティティが含まれるかを返す（テスト用）
func containsEntity(list []ecs.Entity, e ecs.Entity) bool {
	return slices.Contains(list, e)
}

func TestCullDistantSolo(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// 敵を生成し、状態を設定するヘルパ
	spawn := func(x, y int, state gc.AIStateSubState) ecs.Entity {
		e, err := lifecycle.SpawnEnemy(world, x, y, "火の玉")
		require.NoError(t, err)
		world.Components.SoloAI.Get(e).SubState = state
		return e
	}

	// activationRadius = VisionRadiusTiles(24) + margin(2) = 26。プレイヤーは(10,10)
	withinWaiting := spawn(15, 10, gc.AIStateWaiting)   // チェビシェフ距離5 → 処理
	boundaryWaiting := spawn(36, 10, gc.AIStateWaiting) // 距離26（境界）→ 処理
	beyondWaiting := spawn(37, 10, gc.AIStateWaiting)   // 距離27 → スキップ
	beyondDriving := spawn(50, 10, gc.AIStateDriving)   // 距離40 → スキップ
	beyondChasing := spawn(10, 60, gc.AIStateChasing)   // 距離50だが追跡中 → 処理
	beyondFleeing := spawn(60, 60, gc.AIStateFleeing)   // 距離50だが逃亡中 → 処理

	targets := []ecs.Entity{withinWaiting, boundaryWaiting, beyondWaiting, beyondDriving, beyondChasing, beyondFleeing}
	kept := cullDistantSolo(world, targets)

	assert.True(t, containsEntity(kept, withinWaiting), "圏内の待機敵は処理対象")
	assert.True(t, containsEntity(kept, boundaryWaiting), "境界（=半径ちょうど）の待機敵は処理対象")
	assert.False(t, containsEntity(kept, beyondWaiting), "圏外の待機敵はスキップ")
	assert.False(t, containsEntity(kept, beyondDriving), "圏外の徘徊敵はスキップ")
	assert.True(t, containsEntity(kept, beyondChasing), "圏外でも追跡中は距離無関係に処理")
	assert.True(t, containsEntity(kept, beyondFleeing), "圏外でも逃亡中は距離無関係に処理")
}

func TestCullDistantSolo_PlayerApproachActivates(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	player, err := lifecycle.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	enemy, err := lifecycle.SpawnEnemy(world, 40, 10, "火の玉") // 距離30 → 圏外
	require.NoError(t, err)
	world.Components.SoloAI.Get(enemy).SubState = gc.AIStateWaiting

	targets := []ecs.Entity{enemy}

	assert.Empty(t, cullDistantSolo(world, targets), "圏外の待機敵はスキップされる")

	// プレイヤーが近づくと（距離20 → 圏内）同じ敵が処理対象になる
	playerGrid := world.Components.GridElement.Get(player)
	playerGrid.X = 20
	assert.Len(t, cullDistantSolo(world, targets), 1, "接近後は圏内入りして処理対象になる")
}

func TestCullDistantSolo_NoPlayerProcessesAll(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤー不在（GetPlayerEntity が失敗）ではカリングせず全処理する
	enemy, err := lifecycle.SpawnEnemy(world, 100, 100, "火の玉")
	require.NoError(t, err)

	targets := []ecs.Entity{enemy}
	assert.Len(t, cullDistantSolo(world, targets), 1, "プレイヤー不在時はカリングせず全処理する")
}

// TestProcessAll_大規模でpanicしない は多数の敵を配置して数ターン AI 処理を回し、
// panic しないことを確認する L1 ストレスガード。攻撃経路（入れ子 ProcessTurn）の回帰など、
// 大規模時のみ顕在化するクラッシュを捕捉する。
func TestProcessAll_大規模でpanicしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 敵が重ならず配置でき移動も破綻しないよう大きめのマップにする
	d := query.GetDungeon(world)
	d.Level = gc.Level{TileWidth: 100, TileHeight: 100}

	_, err := lifecycle.SpawnPlayer(world, 50, 50, "Ash")
	require.NoError(t, err)

	// 固定 seed で全域に敵を配置（プレイヤー近傍は攻撃経路も通る）
	rng := rand.New(rand.NewPCG(1, 2))
	for range 500 {
		_, err := lifecycle.SpawnEnemy(world, rng.IntN(100), rng.IntN(100), "火の玉")
		require.NoError(t, err)
	}

	proc := NewProcessor(rand.New(rand.NewPCG(3, 4)))
	// 数ターン回して panic・エラーがないこと
	for range 3 {
		require.NoError(t, proc.ProcessAll(world))
		require.NoError(t, query.RestoreAllActionPoints(world))
	}
}

// TestProcessAll_AIフェーズで空間インデックスを再構築しない は §8 の増分更新を実際のホット文脈
// （AIフェーズで多数の敵が移動する）で守る L1 ガード。旧実装（per-move 無効化）なら移動数に比例して
// buildSpatialIndex が走るため、BuildCount の増分で再構築チャーンの再発を決定的に検知する。
func TestProcessAll_AIフェーズで空間インデックスを再構築しない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).Level = gc.Level{TileWidth: 60, TileHeight: 60}

	_, err := lifecycle.SpawnPlayer(world, 30, 30, "Ash")
	require.NoError(t, err)

	// プレイヤー近傍に敵を多数配置（活性半径内＝毎ターン処理・移動される）
	rng := rand.New(rand.NewPCG(1, 2))
	for range 40 {
		x := 30 + rng.IntN(21) - 10
		y := 30 + rng.IntN(21) - 10
		_, err := lifecycle.SpawnEnemy(world, x, y, "火の玉")
		require.NoError(t, err)
	}

	si := query.GetSpatialIndex(world) // 初回構築
	require.NotNil(t, si)
	before := si.BuildCount

	proc := NewProcessor(rand.New(rand.NewPCG(3, 4)))
	const turns = 3
	for range turns {
		require.NoError(t, proc.ProcessAll(world))
		require.NoError(t, query.RestoreAllActionPoints(world))
	}

	// AIフェーズでは40体×数ターン分の敵移動が起きるが、増分更新されるため再構築は起きない。
	// 旧実装なら移動数（数十〜百）に比例して BuildCount が増える
	assert.LessOrEqual(t, si.BuildCount-before, turns,
		"AIフェーズの敵移動は増分更新され、再構築が移動数に比例しない（BuildCount増分=%d）", si.BuildCount-before)
}

// TestCullDistantSolo_ScalingInvariant はカリングが処理対象数を活性半径内に制限する不変条件を守る。
// パフォーマンス劣化（カリングの無効化・削除）を壁時計に依存せず決定的に検出する回帰ガード。
// カリングが壊れると processed == total となりこのテストが落ちる。
func TestCullDistantSolo_ScalingInvariant(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	_, err := lifecycle.SpawnPlayer(world, 25, 25, "Ash")
	require.NoError(t, err)

	// マップ全域に敵を格子配置する。大半はプレイヤーの活性半径外に位置する
	const (
		gridN   = 20 // 20x20 = 400体
		spacing = 15
		offset  = 3
	)
	total := 0
	for gx := range gridN {
		for gy := range gridN {
			_, err := lifecycle.SpawnEnemy(world, offset+gx*spacing, offset+gy*spacing, "火の玉")
			require.NoError(t, err)
			total++
		}
	}

	var allSolo []ecs.Entity
	soloQuery := ecs.NewFilter2[gc.SoloAI, gc.GridElement](world.ECS).Query()
	for soloQuery.Next() {
		allSolo = append(allSolo, soloQuery.Entity())
	}
	processed := len(cullDistantSolo(world, allSolo))

	// カリングが効いていれば処理数は活性半径内の敵のみに絞られ、総数を大きく下回る
	assert.Positive(t, processed, "圏内の敵は処理される")
	assert.Less(t, processed, total/4, "カリングは処理数を活性半径内に制限する（total=%d, processed=%d）", total, processed)
}
