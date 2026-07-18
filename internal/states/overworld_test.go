package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	gs "github.com/kijimaD/ruins/internal/systems"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOverworldState_ロード復元で視界が再計算され真っ暗にならない は、セーブ→ロードで
// VisibleTiles が空へ戻る（serde が json:"-" で除外）とき、OnStart が視界の強制再計算を
// 促して初回フレームで真っ暗にならないことを固定する。
//
// VisionSystem は Depth/DefinitionName が変わらないと内部キャッシュを無効化しない。
// オーバーワールドは常に Depth=0・DefinitionName="" なのでフロア変化が起きず、stale な
// isInitialized で視界再計算がスキップされ、可視タイルが空のまま黒画面になっていた。
func TestOverworldState_ロード復元で視界が再計算され真っ暗にならない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(777, chunkW, chunkH, 3, mapplanner.PlannerTypeOverworldField)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	d := query.GetDungeon(world)
	vs := gs.NewVisionSystem()

	// 初回の視界計算で可視タイルが埋まる（プレイヤーは中央、周囲は開けた原野）
	require.NoError(t, vs.Update(world))
	require.NotEmpty(t, d.VisibleTiles, "前提: 初回で視界が埋まる")

	// ロードを模す: serde は VisibleTiles を空へ初期化する（json:"-"）。
	// VisionSystem インスタンスは world.Updaters に居座り isInitialized/lastPlayer が残る
	d.VisibleTiles = map[gc.GridElement]bool{}

	// ロード復帰の OnStart（sb.Active ブランチ）。プレイヤーは動かさない
	require.NoError(t, st.OnStart(world))

	// stale な同じ VisionSystem で再計算しても視界が戻る（真っ暗回帰防止）
	require.NoError(t, vs.Update(world))
	assert.NotEmpty(t, d.VisibleTiles, "ロード後も視界が再計算されて真っ暗にならない")
}

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

	assert.Equal(t, 1, int(st.band.EastIndex()), "東シフトで eastIndex が進む")
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

	assert.Equal(t, 2, int(st.band.EastIndex()), "収まるまで連続シフトして eastIndex=2")
	px := world.Components.GridElement.Get(player).X
	assert.GreaterOrEqual(t, px, consts.Tile(k/2)*chunkW, "プレイヤーは中央チャンク内に収まる")
	assert.Less(t, px, consts.Tile(k/2+1)*chunkW, "プレイヤーは中央チャンク内に収まる")
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
	require.Equal(t, 0, int(st.band.EastIndex()), "前提: 開始時 eastIndex=0")

	player, err := query.GetPlayerEntity(world)
	require.NoError(t, err)
	// 中央チャンクより西へ（localX < centerSlot*chunkW = 30）
	world.Components.GridElement.Get(player).X = 10
	require.True(t, st.band.ShouldShiftWest(10), "前提: 西シフト条件は満たす")

	require.NoError(t, st.maybeShift(world))
	assert.Equal(t, 0, int(st.band.EastIndex()), "開始点より西へはシフトしない（eastIndex は負にならない）")
}

// TestOverworldState_オーバーレイ進入で帯タイルを消さない は、射撃/観察/ダンジョンメニュー等の
// オーバーレイ（TransPush → OnPause）に入ったときに帯タイルが消えない（黒画面にならない）ことを固定する。
func TestOverworldState_オーバーレイ進入で帯タイルを消さない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(777, chunkW, chunkH, 3, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
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
// 隊員の位置が動かないことを固定する（MovePlayerToPosition の隊員再配置に巻き込まれない）。
func TestOverworldState_オーバーレイ往復で隊員位置が変わらない(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	factory := NewOverworldState(777, chunkW, chunkH, 3, mapplanner.PlannerTypeSmallRoom)
	state, err := factory()
	require.NoError(t, err)
	st, ok := state.(*OverworldState)
	require.True(t, ok)
	require.NoError(t, st.OnStart(world))

	// 隊員を1体作る（SquadMember + FactionAlly + GridElement）
	const memberX, memberY consts.Tile = 20, 12
	member := world.Components.GridElement.NewEntity(&gc.GridElement{X: memberX, Y: memberY})
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
	assert.Equal(t, 0, int(st.band.EastIndex()), "中央チャンク内ではシフトしない")
}
