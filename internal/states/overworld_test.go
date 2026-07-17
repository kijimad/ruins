package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverworldState_OnStart_初期帯とプレイヤー中央(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(777, chunkW, chunkH, k, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	// Level は帯全幅
	assert.Equal(t, chunkW*k, query.GetDungeon(world).Level.TileWidth, "Levelは帯全幅")

	// 各スロットにタイルが存在する（初期帯 K チャンクが埋まっている）
	slotCounts := make([]int, k)
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		x := world.Components.GridElement.Get(q.Entity()).X
		if x >= 0 && x < chunkW*k {
			slotCounts[int(x/chunkW)]++
		}
	}
	for i, c := range slotCounts {
		assert.NotZero(t, c, "スロット%d に初期チャンクが生成される", i)
	}

	// プレイヤーは中央チャンクの中央に居る
	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	pg := world.Components.GridElement.Get(player)
	assert.Equal(t, consts.Tile(k/2)*chunkW+chunkW/2, pg.X, "プレイヤーX は中央チャンク中央")
}

func TestOverworldState_maybeShift_東へ進むとシフトする(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(777, chunkW, chunkH, k, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	// プレイヤーを東チャンクへ踏み込ませる（localX >= 2*chunkW=60）
	world.Components.GridElement.Get(player).X = 2 * chunkW
	// 安定点の前提: Player フェーズ・継続アクティビティなし（OnStart 直後はこの状態）
	require.Equal(t, gc.TurnPhasePlayer, query.GetTurnState(world).Phase)

	require.NoError(t, st.maybeShift(world))

	assert.Equal(t, 1, st.band.EastIndex(), "東シフトで eastIndex が進む")
	assert.Equal(t, chunkW, world.Components.GridElement.Get(player).X, "プレイヤーは中央へ戻る")
}

func TestOverworldState_maybeShift_複数チャンク跨ぎで連続シフト(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(777, chunkW, chunkH, k, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)

	// 中央から2チャンク以上東（帯外）に飛ばす。1回のシフトでは中央に収まらない
	world.Components.GridElement.Get(player).X = 100
	require.NoError(t, st.maybeShift(world))

	assert.Equal(t, 2, st.band.EastIndex(), "収まるまで連続シフトして eastIndex=2")
	px := world.Components.GridElement.Get(player).X
	assert.GreaterOrEqual(t, px, consts.Tile(k/2)*chunkW, "プレイヤーは中央チャンク内に収まる")
	assert.Less(t, px, consts.Tile(k/2+1)*chunkW, "プレイヤーは中央チャンク内に収まる")
}

// TestOverworldState_遺跡遷移で帯を退避復元する は OnPause/OnResume（B2）を検証する。
// 遺跡進入(OnPause)で帯タイルを退避し、帰還(OnResume)で決定的に再生成しつつ
// プレイヤー位置・探索済みを復元することを固定する。
func TestOverworldState_遺跡遷移で帯を退避復元する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(999, chunkW, chunkH, k, mapplanner.PlannerTypeOverworldField)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	world.Components.GridElement.Get(player).X = 45
	world.Components.GridElement.Get(player).Y = 10
	query.GetDungeon(world).ExploredTiles[gc.GridElement{X: 45, Y: 10}] = true

	before := countGridEntities(world)

	// 遺跡へ進入: 帯タイル退避
	require.NoError(t, st.OnPause(world))
	assert.Less(t, countGridEntities(world), before, "帯タイルが退避（削除）される")
	assert.True(t, world.ECS.Alive(player), "プレイヤーは残る")
	require.NotNil(t, st.savedPlayerPos)

	// 遺跡が探索済みをリセットし、プレイヤーを遺跡開始位置へ動かした状況を模す
	query.GetDungeon(world).ExploredTiles = map[gc.GridElement]bool{}
	world.Components.GridElement.Get(player).X = 5
	world.Components.GridElement.Get(player).Y = 5

	// 遺跡から帰還: 帯再構築＋復元
	require.NoError(t, st.OnResume(world))

	// 帯が再生成され各スロットが埋まる
	slotCounts := make([]int, k)
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		x := world.Components.GridElement.Get(q.Entity()).X
		if x >= 0 && x < chunkW*k {
			slotCounts[int(x/chunkW)]++
		}
	}
	for i, c := range slotCounts {
		assert.NotZero(t, c, "スロット%d が再生成される", i)
	}

	// プレイヤー位置・探索済みが復元される
	pg := world.Components.GridElement.Get(player)
	assert.Equal(t, consts.Tile(45), pg.X, "プレイヤーX が復元される")
	assert.Equal(t, consts.Tile(10), pg.Y, "プレイヤーY が復元される")
	assert.True(t, query.GetDungeon(world).ExploredTiles[gc.GridElement{X: 45, Y: 10}], "探索済みが復元される")
}

// TestOverworldState_maybeShift_開始点より西へはシフトしない は、eastIndex=0 で西へ移動しても
// 開始点より西へシフトしない（eastIndex を負にしない）ことを固定する（bot レビュー #3）。
func TestOverworldState_maybeShift_開始点より西へはシフトしない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(777, chunkW, chunkH, k, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))
	require.Equal(t, 0, st.band.EastIndex(), "前提: 開始時 eastIndex=0")

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	// 中央チャンクより西へ（localX < centerSlot*chunkW = 30）
	world.Components.GridElement.Get(player).X = 10
	require.True(t, st.band.ShouldShiftWest(10), "前提: 西シフト条件は満たす")

	require.NoError(t, st.maybeShift(world))
	assert.Equal(t, 0, st.band.EastIndex(), "開始点より西へはシフトしない（eastIndex は負にならない）")
}

// countGridEntities は GridElement を持つエンティティ数を返す。
func countGridEntities(world w.World) int {
	count := 0
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		count++
	}
	return count
}

func TestOverworldState_maybeShift_中央では動かない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(777, chunkW, chunkH, 3, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	require.NoError(t, st.maybeShift(world))
	assert.Equal(t, 0, st.band.EastIndex(), "中央チャンク内ではシフトしない")
}
