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
	"github.com/kijimaD/ruins/internal/worldstream"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// snapshotWorld は全タイルの SpriteKey とソート済みの敵座標を集める。
func snapshotWorld(world w.World) (map[gc.GridElement]string, []consts.Coord[consts.Tile]) {
	tiles := map[gc.GridElement]string{}
	tq := ecs.NewFilter3[gc.GridElement, gc.SpriteRender, gc.Tile](world.ECS).Query()
	for tq.Next() {
		e := tq.Entity()
		tiles[*world.Components.GridElement.Get(e)] = world.Components.SpriteRender.Get(e).SpriteKey
	}
	var enemies []consts.Coord[consts.Tile]
	nq := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for nq.Next() {
		e := nq.Entity()
		if world.Components.Name.Get(e).Name == "火の玉" {
			enemies = append(enemies, world.Components.GridElement.Get(e).Coord)
		}
	}
	sort.Slice(enemies, func(i, j int) bool {
		if enemies[i].X != enemies[j].X {
			return enemies[i].X < enemies[j].X
		}
		return enemies[i].Y < enemies[j].Y
	})
	return tiles, enemies
}

// TestNewChunkGen_市街地の断片は生成順に依存しない は、市街地をまたぐチャンク群を
// 西→東と東→西で生成しても、全タイルと全敵が一致することを固定する。断片が citySeed の
// 一括導出から描かれる不変条件の検証で、帯ストリーミングの再訪に耐える根拠になる。
func TestNewChunkGen_市街地の断片は生成順に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const window = 16 // 市街地リージョン1つぶん
	const runSeed uint64 = 77

	build := func(reverse bool) (map[gc.GridElement]string, []consts.Coord[consts.Tile]) {
		world := testutil.InitTestWorld(t)
		gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, 1, worldstream.ChunkCoord{X: -1}, mapplanner.PlannerTypeOverworldField)
		for i := range window {
			x := i
			if reverse {
				x = window - 1 - i
			}
			require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(x)}, consts.Tile(x)*chunkW, 0))
		}
		return snapshotWorld(world)
	}

	tilesA, enemiesA := build(false)
	tilesB, enemiesB := build(true)
	assert.Equal(t, tilesA, tilesB, "全タイルの SpriteKey は生成順に依存しない")
	assert.Equal(t, enemiesA, enemiesB, "敵の配置は生成順に依存しない")
	assert.NotEmpty(t, enemiesA, "リージョン内に市街地の敵が存在する")
}

// TestNewChunkGen_市街地に敵が湧き帯へ束縛される は、市街地の敵がオーバーワールド帯へ
// 束縛され、遺跡進入時に帯とともに退避される前提を固定する。
func TestNewChunkGen_市街地に敵が湧き帯へ束縛される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 77, chunkW, chunkH, 1, worldstream.ChunkCoord{X: -1}, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}

	found := 0
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.Name.Get(e).Name != "火の玉" {
			continue
		}
		found++
		require.True(t, world.Components.StageBound.Has(e), "市街地の敵は StageBound を持つ")
		assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "オーバーワールド帯へ束縛される")
	}
	assert.Positive(t, found, "市街地の敵が存在する")
}

// TestNewChunkGen_開始チャンクを含む市街地はスキップされる は、開始チャンクが市街地の
// 足あとに重なった場合に市街地が丸ごと生成されず、開始点が安全に保たれることを固定する。
func TestNewChunkGen_開始チャンクを含む市街地はスキップされる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const runSeed uint64 = 77

	// まず市街地の位置を敵の湧いたチャンクから特定する
	scout := testutil.InitTestWorld(t)
	genScout := overworld.NewChunkGen(scout, runSeed, chunkW, chunkH, 1, worldstream.ChunkCoord{X: -1}, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, genScout(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}
	_, enemies := snapshotWorld(scout)
	require.NotEmpty(t, enemies, "前提: 市街地が存在する")
	cityChunk := worldstream.ChunkCoord{X: consts.Chunk(enemies[0].X / chunkW)}

	// 市街地の断片を開始チャンクに指定すると、市街地ごとスキップされ敵が出ない
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, 1, cityChunk, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}
	_, after := snapshotWorld(world)
	assert.Empty(t, after, "開始チャンクを含む市街地はスキップされ敵が湧かない")
}
