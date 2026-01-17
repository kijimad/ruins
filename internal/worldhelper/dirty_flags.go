package worldhelper

import (
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// HasEquipmentChanged は装備変更フラグが立っているかチェックする（フラグは削除しない）
// UI更新判定などで使用する
func HasEquipmentChanged(world w.World) bool {
	hasChanged := false
	world.Manager.Join(
		world.Components.EquipmentChanged,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		hasChanged = true
	}))
	return hasChanged
}

// HasInventoryChanged はインベントリ変更フラグが立っているかチェックする（フラグは削除しない）
// UI更新判定などで使用する
func HasInventoryChanged(world w.World) bool {
	hasChanged := false
	world.Manager.Join(
		world.Components.InventoryChanged,
	).Visit(ecs.Visit(func(_ ecs.Entity) {
		hasChanged = true
	}))
	return hasChanged
}
