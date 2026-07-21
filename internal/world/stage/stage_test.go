package stage

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	stageA      = gc.NewDungeonStage(1)
	stageB      = gc.NewDungeonStage(2)
	stageAbsent = gc.NewNamedDungeonStage("未訪問", 1)
)

// addStageEntity は指定ステージに束縛されたエンティティを1つ作る
func addStageEntity(t *testing.T, world w.World, key gc.StageKey) ecs.Entity {
	t.Helper()
	e := world.ECS.NewEntity()
	world.Components.StageBound.Add(e, &gc.StageBound{Key: key})
	return e
}

func TestSuspendResume(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a1 := addStageEntity(t, world, stageA)
	a2 := addStageEntity(t, world, stageA)
	b1 := addStageEntity(t, world, stageB)
	other := world.ECS.NewEntity() // StageBound なし。Player 相当

	suspend(world, stageA)
	assert.True(t, world.Components.Suspended.Has(a1))
	assert.True(t, world.Components.Suspended.Has(a2))
	assert.False(t, world.Components.Suspended.Has(b1), "別ステージは退避しない")
	assert.False(t, world.Components.Suspended.Has(other), "StageBound なしは退避しない")

	// 二重付与しても壊れない
	suspend(world, stageA)
	assert.True(t, world.Components.Suspended.Has(a1))

	resume(world, stageA)
	assert.False(t, world.Components.Suspended.Has(a1))
	assert.False(t, world.Components.Suspended.Has(a2))
}

func TestExists(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	addStageEntity(t, world, stageA)
	assert.True(t, exists(world, stageA))
	assert.False(t, exists(world, stageAbsent), "未訪問ステージは存在しない")
}

func TestPurge(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	a1 := addStageEntity(t, world, stageA)
	b1 := addStageEntity(t, world, stageB)
	other := world.ECS.NewEntity()

	Purge(world, stageA)
	assert.False(t, world.ECS.Alive(a1), "離脱ステージのエンティティは除去される")
	assert.True(t, world.ECS.Alive(b1), "別ステージは残る")
	assert.True(t, world.ECS.Alive(other), "StageBound なしは残る")
	assert.False(t, exists(world, stageA))
}

func TestResetExploredTiles(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 5, Y: 5}}: true,
	}

	ResetExploredTiles(world)
	assert.Empty(t, query.GetDungeon(world).ExploredTiles, "入り直しで探索履歴は空になる")
}

func TestBind(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 生成物相当。GridElement を持ち StageBound なし
	tile := world.ECS.NewEntity()
	world.Components.GridElement.Add(tile, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}})
	enemy := world.ECS.NewEntity()
	world.Components.GridElement.Add(enemy, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 2, Y: 2}})

	// Player はステージをまたいで生きるので束縛しない
	player := world.ECS.NewEntity()
	world.Components.GridElement.Add(player, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}})
	world.Components.Player.Add(player, &gc.Player{})

	// 既に別ステージに束縛されたエンティティは上書きしない
	existing := world.ECS.NewEntity()
	world.Components.GridElement.Add(existing, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 4, Y: 4}})
	world.Components.StageBound.Add(existing, &gc.StageBound{Key: stageB})

	Bind(world, stageA)

	assert.Equal(t, stageA, world.Components.StageBound.Get(tile).Key, "生成タイルは現ステージに束縛される")
	assert.Equal(t, stageA, world.Components.StageBound.Get(enemy).Key, "生成した敵は現ステージに束縛される")
	assert.False(t, world.Components.StageBound.Has(player), "Player は StageBound を持たない")
	assert.Equal(t, stageB, world.Components.StageBound.Get(existing).Key, "既存の束縛は上書きしない")
}

func TestSwapTo(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = stageA
	a1 := addStageEntity(t, world, stageA) // 現ステージA のエンティティ、稼働中

	genCalls := 0
	generate := func(world w.World, key gc.StageKey) error {
		genCalls++
		addStageEntity(t, world, key)
		return nil
	}

	// A → B。B は未訪問なので生成し、A は退避する
	require.NoError(t, SwapTo(world, stageB, generate))
	assert.True(t, world.Components.Suspended.Has(a1), "離れた A は退避される")
	assert.Equal(t, 1, genCalls, "未訪問の B は生成される")
	assert.True(t, exists(world, stageB))
	assert.Equal(t, stageB, query.GetDungeon(world).CurrentStage)

	// B → A。A は訪問済みなので再稼働し、生成しない
	require.NoError(t, SwapTo(world, stageA, generate))
	assert.False(t, world.Components.Suspended.Has(a1), "戻った A は再稼働される")
	assert.Equal(t, 1, genCalls, "訪問済みの A は再生成しない")
	assert.Equal(t, stageA, query.GetDungeon(world).CurrentStage)
}

func TestSwapTo_未タグの湧きエンティティを現ステージへ回収する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.CurrentStage = stageA

	// プレイ中に湧いた未束縛のフィールドエンティティ。ドロップや置いたアイテム相当
	drop := world.ECS.NewEntity()
	world.Components.GridElement.Add(drop, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 3, Y: 3}})

	require.NoError(t, SwapTo(world, stageB, func(world w.World, key gc.StageKey) error {
		addStageEntity(t, world, key)
		return nil
	}))

	// 湧きは現ステージAへ回収され、現階と共に退避される。次ステージへ漏れない
	require.True(t, world.Components.StageBound.Has(drop), "未束縛の湧きは StageBound を得る")
	assert.Equal(t, stageA, world.Components.StageBound.Get(drop).Key, "現ステージAに回収される")
	assert.True(t, world.Components.Suspended.Has(drop), "回収された湧きは現階と共に退避される")
}

func TestSwapTo_生成失敗時は現ステージを壊さない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.CurrentStage = stageA
	a1 := addStageEntity(t, world, stageA)
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}: true,
	}

	// 未訪問 B への生成が失敗する
	err := SwapTo(world, stageB, func(_ w.World, _ gc.StageKey) error {
		return assert.AnError
	})
	require.Error(t, err)

	// 現ステージは壊れない。A は退避されず、CurrentStage も探索履歴も維持される
	assert.False(t, world.Components.Suspended.Has(a1), "生成失敗時に現ステージA は退避されない")
	assert.Equal(t, stageA, d.CurrentStage, "生成失敗時に CurrentStage は動かない")
	assert.NotEmpty(t, d.ExploredTiles, "生成失敗時に探索履歴は消えない")
}

func TestSwapTo_座標索引を無効化する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	query.GetDungeon(world).CurrentStage = stageA
	// 現ステージのフィールド寸法は StageMeta が持つ。索引構築が寸法を引けるよう用意する
	testutil.SetStageLevel(world, gc.Level{TileWidth: 50, TileHeight: 50})

	// 索引を一度構築しておく
	query.GetSpatialIndex(world)
	si := world.Components.SpatialIndex.Get(world.Resources.SingletonEntity)
	require.True(t, si.Built, "前提: 索引は構築済み")

	require.NoError(t, SwapTo(world, stageB, func(world w.World, key gc.StageKey) error {
		addStageEntity(t, world, key)
		return nil
	}))

	si2 := world.Components.SpatialIndex.Get(world.Resources.SingletonEntity)
	assert.False(t, si2.Built, "swap 後は索引が無効化され、次アクセスで現ステージ用に再構築される")
}

func TestSwapTo_探索履歴をリセットする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)
	d := query.GetDungeon(world)
	d.CurrentStage = stageA
	d.ExploredTiles = map[gc.GridElement]bool{
		{Coord: consts.Coord[consts.Tile]{X: 1, Y: 1}}: true,
	}

	require.NoError(t, SwapTo(world, stageB, func(world w.World, key gc.StageKey) error {
		addStageEntity(t, world, key)
		return nil
	}))
	assert.Empty(t, query.GetDungeon(world).ExploredTiles, "swap で探索履歴は空になる")
}
