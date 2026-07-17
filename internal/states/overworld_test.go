package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/testutil"
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
