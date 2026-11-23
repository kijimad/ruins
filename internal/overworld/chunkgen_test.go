package overworld_test

import (
	"sort"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChunkGen_オフセット配置(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 24, 24
	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)

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
	const chunkW, chunkH consts.Tile = 24, 24

	// プレイヤーはステージをまたぐ訪問者で束縛されない
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "ash")
	require.NoError(t, err)

	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
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

	const chunkW, chunkH consts.Tile = 24, 24

	collect := func() []gc.GridElement {
		world := testutil.InitTestWorld(t)
		gen := overworld.NewChunkGen(world, 999, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
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

// TestShiftNorth_実チャンク生成との統合 は Band と実 ChunkGen を繋いで
// 「実際に北へ1回シフトして北端を実生成し、帯全域が埋まったまま」を固定する。
func TestShiftNorth_実チャンク生成との統合(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	const rows = 3
	const cols = 1
	world := testutil.InitTestWorld(t, testutil.WithStageLevel(gc.Level{TileWidth: chunkW, TileHeight: chunkH * rows}))

	gen := overworld.NewChunkGen(world, 555, chunkW, chunkH, cols, mapplanner.PlannerTypeOverworldField)
	// 初期帯: rows チャンクを各行スロットへ生成
	for cy := range consts.Chunk(rows) {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{Y: cy}, 0, cy.Tiles(chunkH)))
	}
	// プレイヤーを中央行の北端に置く（localY=0 → 北シフト条件）
	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: chunkW / 2, Y: 0}, "ash")
	require.NoError(t, err)

	band := worldstream.NewBand(chunkW, chunkH, cols, rows)
	require.True(t, band.ShouldShiftNorth(world.Components.GridElement.Get(player).Y))
	require.NoError(t, band.ShiftNorth(world, gen))

	assert.Equal(t, 1, int(band.NorthIndex()), "北へ1チャンク進む")
	assert.Equal(t, chunkH, world.Components.GridElement.Get(player).Y, "プレイヤーは中央へ戻る")

	// 帯を3行スロットに分け、各スロットにタイルが存在する（破棄＋生成＋リベース後も全域が埋まる）
	slotCounts := make([]int, rows)
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		y := world.Components.GridElement.Get(q.Entity()).Y
		if y < 0 || y >= chunkH*rows {
			continue
		}
		slotCounts[int(y/chunkH)]++
	}
	for i, c := range slotCounts {
		assert.NotZero(t, c, "スロット%d にタイルが存在する（帯全域が埋まっている）", i)
	}
}

// TestChunkType_小集落はリージョンにちょうど1つ は、小集落がリージョン方式の当選チャンクにだけ
// 分類されることを固定する。集落のチャンクに施設を置くことは settlementFeature のテストが固定する。
func TestChunkType_小集落はリージョンに1つまで(t *testing.T) {
	t.Parallel()

	const regionSpan = 5 // 集落の地物の Spacing
	const cols consts.Chunk = 9
	raws := testutil.InitTestWorld(t).Resources.RawMaster
	counts := map[consts.Chunk]int{}
	for y := consts.Chunk(-40); y < 40; y++ {
		for x := range cols {
			if overworld.IsSettlementChunk(raws, 123, consts.Coord[consts.Chunk]{X: x, Y: y}, cols) {
				counts[consts.FloorDiv(y, regionSpan)]++
			}
		}
	}
	require.NotEmpty(t, counts, "前提: 集落がある")
	for r, n := range counts {
		assert.Equal(t, 1, n, "リージョン %d の小集落", r)
	}
}

// TestNewChunkGen_生成は時間に依存しない は、GameTime が進んでいても同じ (runSeed, 座標) から
// 同じチャンクが生成されることを固定する。昼夜や気温など時間依存の効果は実行時オーバーレイの責務で、
// 生成は座標純関数を保つ。
func TestNewChunkGen_生成は時間に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	c := consts.Coord[consts.Chunk]{X: 3}

	collect := func(advanceTurns consts.Turn) ([]gc.GridElement, []string) {
		world := testutil.InitTestWorld(t)
		query.GetGameTime(world).TotalTurns += advanceTurns
		gen := overworld.NewChunkGen(world, 999, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
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
		var names []string
		nq := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
		for nq.Next() {
			names = append(names, string(world.Components.Name.Get(nq.Entity()).Name))
		}
		sort.Strings(names)
		return walls, names
	}

	wallsA, namesA := collect(0)
	wallsB, namesB := collect(99999)
	assert.Equal(t, wallsA, wallsB, "時間が進んでも壁配置は同一")
	assert.Equal(t, namesA, namesB, "時間が進んでも置かれる物は同一")
}
