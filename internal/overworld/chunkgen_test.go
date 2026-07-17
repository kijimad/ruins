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

func TestChunkSeed_決定的(t *testing.T) {
	t.Parallel()

	first := overworld.ChunkSeed(42, 5)
	again := overworld.ChunkSeed(42, 5)
	assert.Equal(t, first, again, "同じ入力なら同じ seed（決定的）")
	assert.NotEqual(t, overworld.ChunkSeed(42, 1), overworld.ChunkSeed(42, 2), "隣接インデックスで seed が変わる")
	assert.NotEqual(t, overworld.ChunkSeed(1, 5), overworld.ChunkSeed(2, 5), "runSeed が変われば seed も変わる")
}

func TestNewChunkGen_オフセット配置(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	gen := overworld.NewChunkGen(world, 123, chunkW, chunkH, mapplanner.PlannerTypeSmallRoom)

	require.NoError(t, gen(2, 60)) // chunkIndex=2 を offsetX=60 へ

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

func TestNewChunkGen_決定的レイアウト(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20

	collect := func() []gc.GridElement {
		world := testutil.InitTestWorld(t)
		gen := overworld.NewChunkGen(world, 999, chunkW, chunkH, mapplanner.PlannerTypeSmallRoom)
		require.NoError(t, gen(7, 0))

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
	assert.Equal(t, a, b, "同じ (runSeed, chunkIndex) は同じ壁配置＝決定的に再生成できる")
	assert.NotEmpty(t, a, "壁が存在する（生成が空でない）")
}

// TestShiftEast_実チャンク生成との統合 は Band と実 ChunkGen を繋いで
// 「実際に東へ1回シフトして東端を実生成し、帯全域が埋まったまま」を固定する。
func TestShiftEast_実チャンク生成との統合(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)
	const chunkW, chunkH consts.Tile = 30, 20
	const k = 3
	query.GetDungeon(world).Level = gc.Level{TileWidth: chunkW * k, TileHeight: chunkH}

	gen := overworld.NewChunkGen(world, 555, chunkW, chunkH, mapplanner.PlannerTypeSmallRoom)
	// 初期帯: K チャンクを各スロットへ生成
	for i := range k {
		require.NoError(t, gen(i, consts.Tile(i)*chunkW))
	}
	// プレイヤーを中央チャンク東端に置く（localX=2*chunkW → 東シフト条件）
	player, err := lifecycle.SpawnPlayer(world, int(2*chunkW), int(chunkH/2), "Ash")
	require.NoError(t, err)

	band := worldstream.NewBand(chunkW, k)
	require.True(t, band.ShouldShiftEast(world.Components.GridElement.Get(player).X))
	require.NoError(t, band.ShiftEast(world, gen))

	assert.Equal(t, 1, band.EastIndex(), "東へ1チャンク進む")
	assert.Equal(t, chunkW, world.Components.GridElement.Get(player).X, "プレイヤーは中央へ戻る")

	// 帯を3スロットに分け、各スロットにタイルが存在する（破棄＋生成＋リベース後も全域が埋まる）
	slotCounts := make([]int, k)
	q := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
	for q.Next() {
		x := world.Components.GridElement.Get(q.Entity()).X
		if x < 0 || x >= chunkW*k {
			continue
		}
		slotCounts[int(x/chunkW)]++
	}
	for i, c := range slotCounts {
		assert.NotZero(t, c, "スロット%d にタイルが存在する（帯全域が埋まっている）", i)
	}
}
