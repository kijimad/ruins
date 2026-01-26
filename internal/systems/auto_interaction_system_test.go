package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// テスト用のトリガー型定義

// AutoAdjacentTrigger は自動発動するテスト用トリガー（Adjacent）
type AutoAdjacentTrigger struct{}

// Config はTriggerDataインターフェースの実装
func (t AutoAdjacentTrigger) Config() gc.InteractionConfig {
	return gc.InteractionConfig{
		ActivationRange: gc.ActivationRangeAdjacent,
		ActivationWay:   gc.ActivationWayAuto,
	}
}

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

// TestAutoInteractionSystem_AutoWay はAuto方式のトリガーが自動実行されることを確認
func TestAutoInteractionSystem_AutoWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// TestTriggerInteractionを使用
	executed := false
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// トリガーが実行されたことを確認
	assert.True(t, executed, "Autoトリガーが実行される")

	// トリガーエンティティは削除されない
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"トリガーは削除されない")
}

// TestAutoInteractionSystem_ManualWay はManual方式のトリガーが自動実行されないことを確認
func TestAutoInteractionSystem_ManualWay(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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

	// OnCollisionトリガーは実行されず、ドアは閉じたままのはず
	doorComp := world.Components.Door.Get(triggerEntity).(*gc.Door)
	assert.False(t, doorComp.IsOpen, "OnCollisionトリガーは自動実行されないべき")
}

// TestAutoInteractionSystem_OutOfRange は範囲外のAutoトリガーが実行されないことを確認
func TestAutoInteractionSystem_OutOfRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// Auto方式のトリガーを作成（プレイヤーから遠い位置）
	executed := false
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 50, Y: 50}) // 遠い位置
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// 範囲外なので実行されない
	assert.False(t, executed, "範囲外のAutoトリガーは実行されない")

	// トリガーは削除されない
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"範囲外のAutoトリガーは削除されない")
}

// TestAutoInteractionSystem_AdjacentRange は隣接範囲のAutoトリガーが実行されることを確認
func TestAutoInteractionSystem_AdjacentRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// Auto方式 + Adjacent範囲のトリガーを作成（プレイヤーに隣接）
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 11, Y: 10}) // 隣接
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: AutoAdjacentTrigger{},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// 隣接範囲内なので実行されるが削除されない
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"隣接範囲のAutoトリガーは実行されるが削除されない")
}

// TestAutoInteractionSystem_NoPlayer はプレイヤーがいない場合にエラーを返すことを確認
func TestAutoInteractionSystem_NoPlayer(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成しない

	// Auto方式のトリガーを作成
	executed := false
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.Error(t, sys.Update(world), "プレイヤーがいない場合はエラーを返すべき")
}

// TestAutoInteractionSystem_MultipleAutoTriggers は複数のAutoトリガーが同時に実行されることを確認
func TestAutoInteractionSystem_MultipleAutoTriggers(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

	// 複数のAutoトリガーを作成
	executed1 := false
	trigger1 := world.Manager.NewEntity()
	trigger1.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	trigger1.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed1},
	})

	executed2 := false
	trigger2 := world.Manager.NewEntity()
	trigger2.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	trigger2.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed2},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	require.NoError(t, sys.Update(world))

	// 両方のトリガーが実行される
	assert.True(t, executed1, "1つ目のAutoトリガーが実行される")
	assert.True(t, executed2, "2つ目のAutoトリガーが実行される")

	// 両方のトリガーが削除されない
	assert.True(t, trigger1.HasComponent(world.Components.Interactable),
		"1つ目のAutoトリガーは削除されない")
	assert.True(t, trigger2.HasComponent(world.Components.Interactable),
		"2つ目のAutoトリガーは削除されない")
}

// TestAutoInteractionSystem_PlayerNoGridElement はプレイヤーにGridElementがない場合の動作確認
func TestAutoInteractionSystem_PlayerNoGridElement(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成（GridElementなし）
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	// GridElementを追加しない

	// Autoトリガーを作成
	executed := false
	triggerEntity := world.Manager.NewEntity()
	triggerEntity.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})
	triggerEntity.AddComponent(world.Components.Interactable, &gc.Interactable{
		Data: gc.TestTriggerInteraction{Executed: &executed},
	})

	// システム実行
	sys := &AutoInteractionSystem{}
	// GridElementがない場合はnilを返して処理を中断する
	assert.NoError(t, sys.Update(world), "プレイヤーにGridElementがない場合はエラーなしで終了すべき")

	// トリガーは実行されない
	assert.False(t, executed, "プレイヤーにGridElementがない場合、トリガーは実行されない")
	assert.True(t, triggerEntity.HasComponent(world.Components.Interactable),
		"トリガーは削除されない")
}

// TestAutoInteractionSystem_InvalidRange は無効なActivationRangeを持つトリガーがスキップされることを確認
func TestAutoInteractionSystem_InvalidRange(t *testing.T) {
	t.Parallel()

	world := testutil.InitTestWorld(t)

	// プレイヤーを作成
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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
	player := world.Manager.NewEntity()
	player.AddComponent(world.Components.Player, &gc.Player{})
	player.AddComponent(world.Components.GridElement, &gc.GridElement{X: 10, Y: 10})

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
