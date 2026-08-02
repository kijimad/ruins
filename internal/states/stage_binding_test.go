package states

import (
	"testing"

	"github.com/kijimaD/ruins/internal/activity"
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/kijimaD/ruins/internal/world/stage"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlacedItemBindsToCurrentStage は、プレイ中にアイテムを置くと、その現物が
// 現ステージへ束縛され、ステージを離れると一緒に退避されることを実際の設置経路で検証する。
//
// 設置は MoveToField を通り、所有者からの移送は設置時に現ステージへ束縛される。これで置いた物は
// その場のクエリ、たとえば総重量に即座に乗り、置いた階に残り、戻れば現物が復元される。
// swapTo 冒頭の Bind は取りこぼした未束縛エンティティを回収する保険として残る。
func TestPlacedItemBindsToCurrentStage(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 5, Y: 5}, "Ash")
	require.NoError(t, err)

	stageA := gc.NewDungeonStage("テスト遺跡", 1)
	stageB := gc.NewDungeonStage("テスト遺跡", 2)
	query.GetDungeon(world).CurrentStage = stageA

	// バックパックのアイテムを実際の設置経路 DropBehavior で足元へ置く
	item, err := lifecycle.SpawnBackpackItem(world, "木刀", 1)
	require.NoError(t, err)
	_, err = activity.Execute(&activity.DropBehavior{
		Target:      item,
		Destination: gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 6, Y: 6}},
	}, player, world)
	require.NoError(t, err)

	// 置いた直後にフィールド座標を持ち、その場で現ステージAへ束縛される
	require.True(t, world.Components.GridElement.Has(item), "置いたアイテムはフィールド座標を持つ")
	require.True(t, world.Components.LocationOnField.Has(item), "置いたアイテムはフィールド上にある")
	require.True(t, world.Components.StageBound.Has(item), "置いた直後に現ステージへ束縛される")
	assert.Equal(t, stageA, world.Components.StageBound.Get(item).Key, "束縛先は置いた階A")

	// 別ステージBへ移ると、A を離れるので置いたアイテムも一緒に退避される
	var stageBEntity ecs.Entity
	require.NoError(t, stage.SwapTo(world, stageB, func(world w.World, key gc.StageKey) error {
		stageBEntity = world.ECS.NewEntity()
		world.Components.StageBound.Add(stageBEntity, &gc.StageBound{Key: key})
		return nil
	}))

	require.True(t, world.Components.StageBound.Has(item), "置いたアイテムは現ステージへ束縛される")
	assert.Equal(t, stageA, world.Components.StageBound.Get(item).Key, "置いた階Aに束縛される")
	assert.True(t, world.Components.Suspended.Has(item), "階Aを離れると置いたアイテムも退避される")

	// A へ戻ると置いたアイテムが再稼働し、現物が残っている
	require.NoError(t, stage.SwapTo(world, stageA, func(_ w.World, _ gc.StageKey) error {
		return nil // A は訪問済みなので generate は呼ばれない
	}))
	assert.False(t, world.Components.Suspended.Has(item), "階Aへ戻ると置いたアイテムは再稼働する")
	assert.True(t, world.ECS.Alive(item), "置いたアイテムの現物が残っている")
	assert.Equal(t, stageA, world.Components.StageBound.Get(item).Key, "置いた階Aへの束縛は保たれる")

	// 離れた階Bのエンティティは破棄されず、退避されたまま共存している
	assert.True(t, world.ECS.Alive(stageBEntity), "離れた階Bの現物は破棄されず残る")
	assert.True(t, world.Components.Suspended.Has(stageBEntity), "離れた階Bは退避されたまま")
}
