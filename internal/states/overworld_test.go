package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 帯シフトの単体テストは帯セッションへ移した。internal/overworld/session_test.go を参照。
// ここでは DungeonState としてのオーバーワールド統合挙動を固める。

// TestOverworldState_ロード復元で視界が再計算され真っ暗にならない は、セーブ→ロードで
// VisibleTiles が空へ戻る（serde が json:"-" で除外）とき、OnStart が視界の強制再計算を
// 促して初回フレームで真っ暗にならないことを固定する。
func TestOverworldState_ロード復元で視界が再計算され真っ暗にならない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(mapplanner.PlannerTypeOverworldField, &overworld.NewGameParams{RunSeed: 777, ChunkW: chunkW, ChunkH: chunkH, K: 3})
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	visState := query.GetVisionState(world)
	vs := gs.NewVisionSystem()

	// 初回の視界計算で可視タイルが埋まる（プレイヤーは中央、周囲は開けた原野）
	require.NoError(t, vs.Update(world))
	require.NotEmpty(t, visState.VisibleTiles, "前提: 初回で視界が埋まる")

	// ロードを模す: serde は VisibleTiles を空へ初期化する（json:"-"）
	visState.VisibleTiles = map[gc.GridElement]bool{}

	// ロード復帰の OnStart（sb.Active ブランチ）。プレイヤーは動かさない
	require.NoError(t, st.OnStart(world))

	// stale な同じ VisionSystem で再計算しても視界が戻る（真っ暗回帰防止）
	require.NoError(t, vs.Update(world))
	assert.NotEmpty(t, visState.VisibleTiles, "ロード後も視界が再計算されて真っ暗にならない")
}

func TestOverworldState_OnStart_初期帯とプレイヤー中央(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3

	factory := NewOverworldState(mapplanner.PlannerTypeSmallRoom, &overworld.NewGameParams{RunSeed: 777, ChunkW: chunkW, ChunkH: chunkH, K: k})
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	// Level は帯全幅
	assert.Equal(t, chunkW*k, query.GetCurrentStageMeta(world).Level.TileWidth, "Levelは帯全幅")

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

	// 開始チャンクに遺跡入口が1つ置かれ、進入先の遺跡名を持つ
	entranceCount := 0
	eq := ecs.NewFilter1[gc.DungeonEntrance](world.ECS).Query()
	for eq.Next() {
		entranceCount++
		assert.NotEmpty(t, world.Components.DungeonEntrance.Get(eq.Entity()).DefinitionName, "遺跡入口は進入先を持つ")
		assert.True(t, world.Components.Interactable.Has(eq.Entity()), "遺跡入口は相互作用を持つ")
	}
	assert.Equal(t, 1, entranceCount, "開始チャンクに遺跡入口が1つ置かれる")
}

// TestOverworldState_オーバーレイ進入で帯タイルを消さない は、射撃/観察等のオーバーレイ
// （TransPush → OnPause）に入ったときに帯タイルが消えない（黒画面にならない）ことを固定する。
func TestOverworldState_オーバーレイ進入で帯タイルを消さない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(mapplanner.PlannerTypeSmallRoom, &overworld.NewGameParams{RunSeed: 777, ChunkW: chunkW, ChunkH: chunkH, K: 3})
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	before := countGridEntities(world)
	require.Positive(t, before, "前提: 帯タイルが存在する")

	// オーバーレイ進入 = OnPause
	require.NoError(t, st.OnPause(world))

	assert.Equal(t, before, countGridEntities(world),
		"オーバーレイ進入で帯タイルを消さない（黒画面バグ回帰防止）")
}

// TestOverworldState_オーバーレイ往復で隊員位置が変わらない は、オーバーレイに入って戻っても
// 隊員の位置が動かないことを固定する。
func TestOverworldState_オーバーレイ往復で隊員位置が変わらない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(mapplanner.PlannerTypeSmallRoom, &overworld.NewGameParams{RunSeed: 777, ChunkW: chunkW, ChunkH: chunkH, K: 3})
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*DungeonState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	// 隊員を1体作る（SquadMember + FactionAlly + GridElement）
	const memberX, memberY consts.Tile = 20, 12
	member := world.Components.GridElement.NewEntity(&gc.GridElement{Coord: consts.Coord[consts.Tile]{X: memberX, Y: memberY}})
	world.Components.SquadMember.Add(member, &gc.SquadMember{})
	world.Components.FactionAlly.Add(member, &gc.FactionAlly{})

	// オーバーレイ往復 = OnPause → OnResume
	require.NoError(t, st.OnPause(world))
	require.NoError(t, st.OnResume(world))

	g := world.Components.GridElement.Get(member)
	assert.Equal(t, memberX, g.X, "隊員Xは変わらない")
	assert.Equal(t, memberY, g.Y, "隊員Yは変わらない")
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
