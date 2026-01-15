package systems

import (
	w "github.com/kijimaD/ruins/lib/world"
	"github.com/kijimaD/ruins/lib/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// InventoryChangedSystem はインベントリ変動のダーティフラグが立ったら、所持重量を再計算する
type InventoryChangedSystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装する
func (sys InventoryChangedSystem) String() string {
	return "InventoryChangedSystem"
}

// ShouldRun はインベントリ変動フラグをチェックし、フラグをクリアする
// ShouldRunner interfaceを実装する
func (sys *InventoryChangedSystem) ShouldRun(world w.World) bool {
	running := false
	world.Manager.Join(
		world.Components.InventoryChanged,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		running = true
		entity.RemoveComponent(world.Components.InventoryChanged)
	}))
	return running
}

// Update はインベントリ変動フラグをチェックし、必要に応じて所持重量を再計算する
// w.Updater interfaceを実装する
func (sys *InventoryChangedSystem) Update(world w.World) error {
	if !sys.ShouldRun(world) {
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
