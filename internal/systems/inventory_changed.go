package systems

import (
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InventoryChangedSystem はインベントリ変動のダーティフラグが立ったら、所持重量を再計算する
type InventoryChangedSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装する
func (sys InventoryChangedSystem) String() string {
	return "InventoryChangedSystem"
}

// ShouldRun はインベントリ変動フラグをチェックする（フラグ削除は Update で行う）
// ShouldRunner interfaceを実装する
func (sys *InventoryChangedSystem) ShouldRun(world w.World) bool {
	running := false
	world.Manager.Join(
		world.Components.InventoryChanged,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		running = true
	}))
	return running
}

// Update はインベントリ変動フラグをチェックし、必要に応じて所持重量を再計算する
// w.Updater interfaceを実装する
func (sys *InventoryChangedSystem) Update(world w.World) error {
	// フラグをチェックしてクリアする
	hasChanged := false
	world.Manager.Join(
		world.Components.InventoryChanged,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		hasChanged = true
		entity.RemoveComponent(world.Components.InventoryChanged)
	}))

	if !hasChanged {
		return nil
	}

	// プレイヤーの所持重量を再計算
	world.Manager.Join(
		world.Components.Player,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		worldhelper.UpdateCarryingWeight(world, entity)
	}))

	return nil
}
