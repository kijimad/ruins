package states

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
)

var (
	stageA      = gc.StageKey{Kind: gc.StageKindDungeon, Depth: 1}
	stageB      = gc.StageKey{Kind: gc.StageKindDungeon, Depth: 2}
	stageAbsent = gc.StageKey{Kind: gc.StageKindRuin, Ruin: "未訪問", Depth: 1}
)

// addStageEntity は指定ステージに属するエンティティを1つ作る
func addStageEntity(t *testing.T, world w.World, key gc.StageKey) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.StageMember.Add(e, &gc.StageMember{Key: key})
	return e
}

func TestSuspendResumeStage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a1 := addStageEntity(t, world, stageA)
	a2 := addStageEntity(t, world, stageA)
	b1 := addStageEntity(t, world, stageB)
	other := world.ECS.NewEntity() // StageMember なし。Player 相当

	suspendStage(world, stageA)
	assert.True(t, world.Components.Suspended.Has(a1))
	assert.True(t, world.Components.Suspended.Has(a2))
	assert.False(t, world.Components.Suspended.Has(b1), "別ステージは退避しない")
	assert.False(t, world.Components.Suspended.Has(other), "StageMember なしは退避しない")

	// 二重付与しても壊れない
	suspendStage(world, stageA)
	assert.True(t, world.Components.Suspended.Has(a1))

	resumeStage(world, stageA)
	assert.False(t, world.Components.Suspended.Has(a1))
	assert.False(t, world.Components.Suspended.Has(a2))
}

func TestStageExists(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	addStageEntity(t, world, stageA)
	assert.True(t, stageExists(world, stageA))
	assert.False(t, stageExists(world, stageAbsent), "未訪問ステージは存在しない")
}

func TestPurgeStage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a1 := addStageEntity(t, world, stageA)
	b1 := addStageEntity(t, world, stageB)
	other := world.ECS.NewEntity()

	purgeStage(world, stageA)
	assert.False(t, world.ECS.Alive(a1), "離脱ステージのエンティティは除去される")
	assert.True(t, world.ECS.Alive(b1), "別ステージは残る")
	assert.True(t, world.ECS.Alive(other), "StageMember なしは残る")
	assert.False(t, stageExists(world, stageA))
}

func TestResetExploredTiles(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}: true,
	}

	resetExploredTiles(world)
	assert.Empty(t, query.GetDungeon(world).ExploredTiles, "入り直しで探索履歴は空になる")
}

func TestSwapTo(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = stageA
	a1 := addStageEntity(t, world, stageA) // 現ステージA のエンティティ、稼働中

	genCalls := 0
	generate := func(world w.World, key gc.StageKey) {
		genCalls++
		addStageEntity(t, world, key)
	}

	// A → B。B は未訪問なので生成し、A は退避する
	swapTo(world, stageB, generate)
	assert.True(t, world.Components.Suspended.Has(a1), "離れた A は退避される")
	assert.Equal(t, 1, genCalls, "未訪問の B は生成される")
	assert.True(t, stageExists(world, stageB))
	assert.Equal(t, stageB, query.GetDungeon(world).CurrentStage)

	// B → A。A は訪問済みなので再稼働し、生成しない
	swapTo(world, stageA, generate)
	assert.False(t, world.Components.Suspended.Has(a1), "戻った A は再稼働される")
	assert.Equal(t, 1, genCalls, "訪問済みの A は再生成しない")
	assert.Equal(t, stageA, query.GetDungeon(world).CurrentStage)
}

func TestSwapTo_探索履歴をリセットする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.CurrentStage = stageA
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}: true,
	}

	swapTo(world, stageB, func(world w.World, key gc.StageKey) {
		addStageEntity(t, world, key)
	})
	assert.Empty(t, query.GetDungeon(world).ExploredTiles, "swap で探索履歴は空になる")
}
