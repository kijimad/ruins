package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/worldhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のトリガー型定義

// InvalidAutoRangeTrigger は無効なActivationRangeを持つ自動発動トリガー（テスト用）
type InvalidAutoRangeTrigger struct{}

// Config はTriggerDataインターフェースの実装
func (t InvalidAutoRangeTrigger) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRange("INVALID_RANGE"),
		ActivationWay:   gc.ActivationWayAuto,
	}
}

// InvalidAutoWayTrigger は無効なActivationWayを持つトリガー（テスト用）
type InvalidAutoWayTrigger struct{}

// Config はTriggerDataインターフェースの実装
func (t InvalidAutoWayTrigger) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRangeSameTile,
		ActivationWay:   gc.ActivationWay("INVALID_WAY"),
	}
}

func TestAutoInteractionSystem_NoGridElement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// GridElementなしのプレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})

	// システム実行（エラーなしで完了するべき）
	sys := &AutoInteractionSystem{}
	err := sys.Update(world)
	assert.NoError(t, err, "GridElementがなくてもエラーにならない")
}

func TestAutoInteractionSystem_OutOfRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// 範囲外にあるトリガーを作成（距離が2以上）
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 15, Y: 15})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.ItemInteraction{},
	})
	triggerEntity.AddComponent(world.Components.Consumable, &gc.Consumable{})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// 範囲外のトリガーは処理されない
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"範囲外のトリガーは処理されないべき")
}

// TestAutoInteractionSystem_ManualWay はManual方式のトリガーが自動実行されないことを確認
func TestAutoInteractionSystem_ManualWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// Manual方式のトリガーを作成（プレイヤーと同じタイル）
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.ItemInteraction{}, // Manual 方式
	})
	triggerEntity.AddComponent(world.Components.Consumable, &gc.Consumable{})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// Manualトリガーは実行されず、残っているべき
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"Manualトリガーは自動実行されないべき")
	assert.True(t, triggerEntity.HasComponent(world.Components.Consumable),
		"Manualトリガーは自動実行されないので削除されないべき")
}

// TestAutoInteractionSystem_OnCollisionWay はOnCollision方式のトリガーが自動実行されないことを確認
func TestAutoInteractionSystem_OnCollisionWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// OnCollision方式のトリガーを作成（プレイヤーと隣接）
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.DoorInteraction{}, // OnCollision 方式
	})
	triggerEntity.AddComponent(world.Components.Door, &gc.Door{IsOpen: false, Orientation: gc.DoorOrientationHorizontal})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// OnCollisionトリガーは実行されず、扉は閉じたままのはず
	doorComp := world.Components.Door.Get(triggerEntity).(*gc.Door)
	assert.False(t, doorComp.IsOpen, "OnCollisionトリガーは自動実行されないべき")
}

// TestAutoInteractionSystem_InvalidRange は無効なActivationRangeを持つトリガーがスキップされることを確認
func TestAutoInteractionSystem_InvalidRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// 無効なActivationRangeを持つトリガーを作成
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: InvalidAutoRangeTrigger{},
	})
	triggerEntity.AddComponent(world.Components.Consumable, &gc.Consumable{})

	// システム実行（エラーは返さず、警告ログを出してスキップする）
	sys := &AutoInteractionSystem{}
	assert.NoError(t, sys.Update(world), "無効なトリガーはスキップされ、エラーは返さない")

	// トリガーは実行されず、残っているべき
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"無効なActivationRangeのトリガーはスキップされるべき")
	assert.True(t, triggerEntity.HasComponent(world.Components.Consumable),
		"無効なActivationRangeのトリガーは削除されないべき")
}

// TestAutoInteractionSystem_InvalidWay は無効なActivationWayを持つトリガーがスキップされることを確認
func TestAutoInteractionSystem_InvalidWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	_, err := worldhelper.SpawnPlayer(world, 10, 10, "Ash")
	require.NoError(t, err)

	// 無効なActivationWayを持つトリガーを作成
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: InvalidAutoWayTrigger{},
	})
	triggerEntity.AddComponent(world.Components.Consumable, &gc.Consumable{})

	// システム実行（エラーは返さず、警告ログを出してスキップする）
	sys := &AutoInteractionSystem{}
	assert.NoError(t, sys.Update(world), "無効なトリガーはスキップされ、エラーは返さない")

	// トリガーは実行されず、残っているべき
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"無効なActivationWayのトリガーはスキップされるべき")
	assert.True(t, triggerEntity.HasComponent(world.Components.Consumable),
		"無効なActivationWayのトリガーは削除されないべき")
}
