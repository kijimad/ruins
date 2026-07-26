package overworld_test

import (
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

	const chunkW, chunkH consts.Tile = 30, 20
	world := testutil.InitTestWorld(t)
	gen := overworld.NewChunkGen(world, 500, chunkW, chunkH, 1, mapplanner.PlannerTypeOverworldField)
	for i := range 12 {
		require.NoError(t, gen(consts.Coord[consts.Chunk]{X: consts.Chunk(i)}, consts.Tile(i)*chunkW, 0))
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
