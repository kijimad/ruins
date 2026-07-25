package overworld_test

import (
	"fmt"
	"slices"
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

// isHostile は敵エンティティかを近接攻撃の相互作用とターン参加の有無で判別する。
// 敵の種類は敵テーブル抽選で変わるため、名前でなく振る舞いで見る。破壊可能な prop も
// 近接の相互作用を持つため、ターンに参加する TurnBased を併せて要求する。
func isHostile(world w.World, e ecs.Entity) bool {
	if !world.Components.TurnBased.Has(e) || !world.Components.Interactable.Has(e) {
		return false
	}
	return slices.Contains(world.Components.Interactable.Get(e).Interactions, gc.InteractionMelee)
}

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
		if isHostile(world, e) {
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

// snapshotNamed は名前を持つ全エンティティを「名前@座標」のソート済み文字列で集める。
// 内装 prop・NPC・入口を含む配置全体の決定性を比較するために使う。
func snapshotNamed(world w.World) []string {
	var named []string
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		p := world.Components.GridElement.Get(e).Coord
		named = append(named, fmt.Sprintf("%s@%d,%d", world.Components.Name.Get(e).Name, p.X, p.Y))
	}
	sort.Strings(named)
	return named
}

// TestNewChunkGen_市街地の断片は生成順に依存しない は、市街地をまたぐチャンク群を
// 西→東と東→西で生成しても、全タイルと全敵が一致することを固定する。断片が citySeed の
// 一括導出から描かれる不変条件の検証で、帯ストリーミングの再訪に耐える根拠になる。
func TestNewChunkGen_市街地の断片は生成順に依存しない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	const window = 16 // 市街地リージョン1つぶん
	const runSeed uint64 = 77

	build := func(reverse bool) (map[gc.GridElement]string, []consts.Coord[consts.Tile], []string) {
		world := testutil.InitTestWorld(t)
		gen := overworld.NewChunkGen(world, runSeed, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
		for i := range window {
			x := i
			if reverse {
				x = window - 1 - i
			}
			require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(x)}, consts.Tile(x)*chunkW, 0))
		}
		tiles, enemies := snapshotWorld(world)
		return tiles, enemies, snapshotNamed(world)
	}

	tilesA, enemiesA, namedA := build(false)
	tilesB, enemiesB, namedB := build(true)
	assert.Equal(t, tilesA, tilesB, "全タイルの SpriteKey は生成順に依存しない")
	assert.Equal(t, enemiesA, enemiesB, "敵の配置は生成順に依存しない")
	assert.Equal(t, namedA, namedB, "内装propやNPCを含む全配置が生成順に依存しない")
	assert.NotEmpty(t, enemiesA, "リージョン内に市街地の敵が存在する")
}

// TestNewChunkGen_市街地に敵が湧き帯へ束縛される は、市街地の敵がオーバーワールド帯へ
// 束縛され、遺跡進入時に帯とともに退避される前提を固定する。
func TestNewChunkGen_市街地に敵が湧き帯へ束縛される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 77, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}

	found := 0
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !isHostile(world, e) {
			continue
		}
		found++
		require.True(t, world.Components.StageBound.Has(e), "市街地の敵は StageBound を持つ")
		assert.Equal(t, gc.NewOverworldStage(), world.Components.StageBound.Get(e).Key, "オーバーワールド帯へ束縛される")
	}
	assert.Positive(t, found, "市街地の敵が存在する")
}

// TestNewChunkGen_市街地の建物に見える扉が置かれる は、各建物の開口に扉エンティティが
// 実体化されることを固定する。壁の切れ目だけだと入口が分からない退行を防ぐ。
func TestNewChunkGen_市街地の建物に見える扉が置かれる(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 30, 20
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 77, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 16 {
		require.NoError(t, gen(worldstream.ChunkCoord{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
	}

	doors := 0
	q := ecs.NewFilter2[gc.GridElement, gc.Name](world.ECS).Query()
	for q.Next() {
		if world.Components.Name.Get(q.Entity()).Name == "扉" {
			doors++
		}
	}
	assert.Positive(t, doors, "市街地の建物に扉が置かれる")
}
