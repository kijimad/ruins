package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoInteractionSystem_NoGridElement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// GridElementなしのプレイヤーを作成
	player := world.ECS.NewEntity()
	world.Components.Player.Add(player, &gc.Player{})

	// システム実行（エラーなしで完了するべき）
	sys := &AutoInteractionSystem{}
	err := sys.Update(world)
	assert.NoError(t, err, "GridElementがなくてもエラーにならない")
}

func TestAutoInteractionSystem_OutOfRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 範囲外にフィールドアイテムを置く。プレイヤーから距離が2以上
	item, err := lifecycle.SpawnFieldItem(world, "healing_potion", 15, 15, 1)
	require.NoError(t, err)

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// 範囲外のトリガーは処理されない
	assert.True(t, world.Components.Interactable.Has(item),
		"範囲外のトリガーは処理されないべき")
}

// TestAutoInteractionSystem_ManualWay はManual方式のトリガーが自動実行されないことを確認
func TestAutoInteractionSystem_ManualWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// フィールドアイテムはInteractionItem=Manual方式。プレイヤーと同じタイルに置く
	item, err := lifecycle.SpawnFieldItem(world, "healing_potion", 10, 10, 1)
	require.NoError(t, err)

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// Manualトリガーは実行されず、残っているべき
	assert.True(t, world.Components.Interactable.Has(item),
		"Manualトリガーは自動実行されないべき")
	assert.True(t, world.Components.Consumable.Has(item),
		"Manualトリガーは自動実行されないので削除されないべき")
}

// TestAutoInteractionSystem_OnCollisionWay はOnCollision方式のトリガーが自動実行されないことを確認
func TestAutoInteractionSystem_OnCollisionWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// OnCollision方式の扉をプレイヤーと隣接して置く
	door, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: 11, Y: 10}, gc.DoorOrientationHorizontal)
	require.NoError(t, err)

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// OnCollisionトリガーは実行されず、扉は閉じたままのはず
	doorComp := world.Components.Door.Get(door)
	assert.False(t, doorComp.IsOpen, "OnCollisionトリガーは自動実行されないべき")
}

// TestAutoInteractionSystem_InvalidRange は無効なActivationRangeを持つトリガーがスキップされることを確認
func TestAutoInteractionSystem_InvalidRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)

	// 未知の種類（平坦化によりゼロ値=無効なConfigになる）のトリガーを作成
	triggerEntity := world.ECS.NewEntity()
	world.Components.GridElement.Add(triggerEntity, &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: 10, Y: 10}})
	world.Components.Interactable.Add(triggerEntity, &gc.Interactable{
		Interactions: []gc.InteractionKind{gc.InteractionKind("UNKNOWN")},
	})
	world.Components.Consumable.Add(triggerEntity, &gc.Consumable{})

	// システム実行（エラーは返さず、警告ログを出してスキップする）
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world), "無効なトリガーはスキップされ、エラーは返さない")

	// トリガーは実行されず、残っているべき
	assert.True(t, world.Components.Interactable.Has(triggerEntity),
		"無効なConfigのトリガーはスキップされるべき")
	assert.True(t, world.Components.Consumable.Has(triggerEntity),
		"無効なConfigのトリガーは削除されないべき")
}
