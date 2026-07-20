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
