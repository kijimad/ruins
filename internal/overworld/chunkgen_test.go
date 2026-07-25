package overworld_test

import (
	"sort"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkSeed2D_決定的(t *testing.T) {
	t.Parallel()

	first := overworld.ChunkSeed2D(42, 5, 2)
	again := overworld.ChunkSeed2D(42, 5, 2)
	assert.Equal(t, first, again, "同じ入力なら同じ seed（決定的）")
	assert.NotEqual(t, overworld.ChunkSeed2D(1, 5, 2), overworld.ChunkSeed2D(2, 5, 2), "runSeed が変われば seed も変わる")
}

func TestChunkSeed2D_転置と隣接で散る(t *testing.T) {
	t.Parallel()

	assert.NotEqual(t, overworld.ChunkSeed2D(42, 1, 2), overworld.ChunkSeed2D(42, 2, 1), "転置した座標は別の seed になる")

	// 負を含む近傍グリッド全域で seed が衝突しないことを確認する。
	// シードの世代間互換は保証しないため、特定の値との一致は検証しない
	seen := map[uint64]consts.Coord[consts.Chunk]{}
	for cy := consts.Chunk(-8); cy <= 8; cy++ {
		for cx := consts.Chunk(-8); cx <= 8; cx++ {
			s := overworld.ChunkSeed2D(42, cx, cy)
			prev, dup := seen[s]
			assert.False(t, dup, "(%d,%d) の seed が (%d,%d) と衝突しない", cx, cy, prev.X, prev.Y)
			seen[s] = consts.Coord[consts.Chunk]{X: cx, Y: cy}
		}
	}
}

func TestNewChunkGen_オフセット配置(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)

	require.NoError(t, gen(consts.Coord[consts.Chunk]{X: 2}, 60, 0)) // X=2 を offsetX=60 へ

	query := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	count := 0
	for query.Next() {
		g := world.Components.GridElement.Get(query.Entity())
		assert.GreaterOrEqual(t, g.X, consts.Tile(60), "オフセット以上")
		assert.Less(t, g.X, consts.Tile(60)+chunkW, "オフセット+幅未満")
		count++
	}
	assert.GreaterOrEqual(t, count, int(chunkW*chunkH), "全タイルぶん配置される")
}

func TestNewChunkGen_生成物をオーバーワールドへ束縛する(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20

	// プレイヤーはステージをまたぐ訪問者で束縛されない
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, gen(consts.Coord[consts.Chunk]{}, 0, 0))

	overworldKey := gc.NewOverworldStage()

	// 生成したチャンクのフィールドエンティティは全てオーバーワールドへ束縛される。
	// これが無いと遺跡へ入るとき帯を退避できない
	boundCount := 0
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if e == player {
			continue
		}
		require.True(t, world.Components.StageBound.Has(e), "生成物は StageBound を持つ")
		assert.Equal(t, overworldKey, world.Components.StageBound.Get(e).Key, "オーバーワールドへ束縛される")
		boundCount++
	}
	assert.Positive(t, boundCount, "束縛された生成物がある")
	assert.False(t, world.Components.StageBound.Has(player), "Player は StageBound を持たない")
}

func TestNewChunkGen_決定的レイアウト(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20

	collect := func() []gc.GridElement {
		world := testutil.InitTestWorld(t)
		gen := overworld.NewChunkGen(world, 999, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: 7}, 0, 0))

		var walls []gc.GridElement
		q := ecs.NewFilter2[gc.GridElement, gc.BlockPass](world.ECS).Query()
		for q.Next() {
			walls = append(walls, *world.Components.GridElement.Get(q.Entity()))
		}
		sort.Slice(walls, func(i, j int) bool {
			if walls[i].X != walls[j].X {
				return walls[i].X < walls[j].X
			}
			return walls[i].Y < walls[j].Y
		})
		return walls
	}

	a := collect()
	b := collect()
	assert.Equal(t, a, b, "同じ (runSeed, 座標) は同じ壁配置＝決定的に再生成できる")
	assert.NotEmpty(t, a, "壁が存在する（生成が空でない）")
}

// TestShiftEast_実チャンク生成との統合 は Band と実 ChunkGen を繋いで
// 「実際に東へ1回シフトして東端を実生成し、帯全域が埋まったまま」を固定する。
func TestShiftEast_実チャンク生成との統合(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const cols = 3
	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: chunkW * cols, TileHeight: chunkH}))

	gen := overworld.NewChunkGen(world, 555, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
	// 初期帯: cols チャンクを各スロットへ生成
	for i := range cols {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}
	// プレイヤーを中央チャンク東端に置く（localX=2*chunkW → 東シフト条件）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 2 * chunkW, Y: chunkH / 2}, "Ash")
	require.NoError(t, err)

	band := worldstream.NewBand(chunkW, chunkH, cols, 1)
	require.True(t, band.ShouldShiftEast(world.Components.GridElement.Get(player).X))
	require.NoError(t, band.ShiftEast(world, gen))

	assert.Equal(t, 1, int(band.EastIndex()), "東へ1チャンク進む")
	assert.Equal(t, chunkW, world.Components.GridElement.Get(player).X, "プレイヤーは中央へ戻る")

	// 帯を3スロットに分け、各スロットにタイルが存在する（破棄＋生成＋リベース後も全域が埋まる）
	slotCounts := make([]int, cols)
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		x := world.Components.GridElement.Get(q.Entity()).X
		if x < 0 || x >= chunkW*cols {
			continue
		}
		slotCounts[int(x/chunkW)]++
	}
	for i, c := range slotCounts {
		assert.NotZero(t, c, "スロット%d にタイルが存在する（帯全域が埋まっている）", i)
	}
}

// merchantName は小集落の店NPC名。テスト間で共有する。
const merchantName = "商人"

// settlementBucket は 商人 が立つチャンクスロットと、商人がいるかを返す。
func settlementBucket(world w.World) (int, bool) {
	const chunkW consts.Tile = 30
	q := ecs.NewFilter1[gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.Name.Get(e).Name != merchantName {
			continue
		}
		x := world.Components.GridElement.Get(e).X
		q.Close()
		return int(x / chunkW), true
	}
	return 0, false
}

// TestNewChunkGen_小集落はリージョンにちょうど1つ生成される は、地物レイヤが
// リージョン方式の当選チャンクにだけ小集落を置くことを固定する。
func TestNewChunkGen_小集落はリージョンにちょうど1つ生成される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const regionSpan = 8 // settlementPlacement の Spacing。1リージョンぶんを生成する
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
	for i := range regionSpan {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}

	count := 0
	q := ecs.NewFilter1[gc.Name](world.ECS).Query()
	for q.Next() {
		if world.Components.Name.Get(q.Entity()).Name == merchantName {
			count++
		}
	}
	assert.Equal(t, 1, count, "1リージョンに小集落はちょうど1つ")
}

// TestNewChunkGen_外れチャンクには小集落が出ない は、リージョン抽選に外れたチャンクには
// 小集落が置かれないことを固定する。開始特例は無く、当選チャンクだけが集落になる。
func TestNewChunkGen_外れチャンクには小集落が出ない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	// まず当選チャンクを探し、外れチャンクを1つ確定させる
	scout := testutil.InitTestWorld(t)
	genScout := overworld.NewChunkGen(scout, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
	for i := range 8 {
		require.NoError(t, genScout(consts.Coord[consts.Chunk]{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}
	winner, ok := settlementBucket(scout)
	require.True(t, ok, "前提: 当選チャンクが存在する")
	loser := consts.Coord[consts.Chunk]{X: consts.Chunk((winner + 1) % 8)}

	// 外れチャンク単体では小集落が出ない
	plain := testutil.InitTestWorld(t)
	genPlain := overworld.NewChunkGen(plain, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
	require.NoError(t, genPlain(loser, 0, 0))
	_, plainOK := settlementBucket(plain)
	assert.False(t, plainOK, "外れチャンクに小集落は出ない")
}

// TestNewChunkGen_生成は時間に依存しない は、GameTime が進んでいても同じ (runSeed, 座標) から
// 同じチャンクが生成されることを固定する。前線など時間依存の効果は実行時オーバーレイの責務で、
// 生成は座標純関数を保つ。
func TestNewChunkGen_生成は時間に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	c := consts.Coord[consts.Chunk]{X: 3}

	collect := func(advanceTurns consts.Turn) ([]gc.GridElement, int, bool) {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns += advanceTurns
		gen := overworld.NewChunkGen(world, 999, chunkW, chunkH, 1, mapplanner.PlannerTypeSmallRoom)
		require.NoError(t, gen(c, 0, 0))

		var walls []gc.GridElement
		q := ecs.NewFilter2[gc.GridElement, gc.BlockPass](world.ECS).Query()
		for q.Next() {
			walls = append(walls, *world.Components.GridElement.Get(q.Entity()))
		}
		sort.Slice(walls, func(i, j int) bool {
			if walls[i].X != walls[j].X {
				return walls[i].X < walls[j].X
			}
			return walls[i].Y < walls[j].Y
		})
		slot, ok := settlementBucket(world)
		return walls, slot, ok
	}

	wallsA, slotA, okA := collect(0)
	wallsB, slotB, okB := collect(99999)
	assert.Equal(t, wallsA, wallsB, "時間が進んでも壁配置は同一")
	assert.Equal(t, okA, okB, "時間が進んでも小集落の有無は同一")
	assert.Equal(t, slotA, slotB, "時間が進んでも小集落のスロットは同一")
}
