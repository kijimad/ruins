package overworld_test

import (
	"slices"
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/overworld"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewChunkGen_遺跡入口が帯全域に決定的に配置される は、遺跡入口リージョン1つぶんの
// チャンク群に入口がちょうど1つ現れ、進入先の定義名を持ち帯へ束縛されることを固定する。
func TestNewChunkGen_遺跡入口が帯全域に決定的に配置される(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 500, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	// 遺跡入口は Y 方向のリージョンに並ぶので、リージョン3つぶんを Y 方向に生成する
	for i := range 12 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 0, consts.Tile(i)*chunkH))
	}

	count := 0
	q := ecs.NewFilter1[gc.DungeonEntrance](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		count++
		assert.NotEmpty(t, world.Components.DungeonEntrance.Get(e).DefinitionName, "進入先の定義名を持つ")
		require.True(t, world.Components.StageBound.Has(e), "入口は帯へ束縛される")
	}
	// 12チャンクは Spacing 4 のリージョン3つぶんで、各リージョンにちょうど1つずつ現れる
	assert.Equal(t, 3, count, "リージョンごとに入口はちょうど1つ")
}

// TestNewChunkGen_全てのダンジョンの入口の重みが0なら遺跡入口を置かない は、引けるダンジョンが
// 無いとき入口の地物が何も置かずに生成を続けることを固定する。
func TestNewChunkGen_全てのダンジョンの入口の重みが0なら遺跡入口を置かない(t *testing.T) {
	t.Parallel()

	const chunkW, chunkH consts.Tile = 24, 24
	world := testutil.InitTestWorld(t)
	dungeons := slices.Clone(*world.Resources.RawMaster.Dungeons)
	for i := range dungeons {
		dungeons[i].EntranceWeight = 0
	}
	world.Resources.RawMaster.Dungeons = &dungeons
	gen := overworld.NewChunkGen(world, 500, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 12 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{Y: consts.Chunk(i)}, 0, consts.Tile(i)*chunkH))
	}

	count := 0
	q := ecs.NewFilter1[gc.DungeonEntrance](world.ECS).Query()
	for q.Next() {
		count++
	}
	assert.Equal(t, 0, count, "引けるダンジョンが無いので入口は0")
}
