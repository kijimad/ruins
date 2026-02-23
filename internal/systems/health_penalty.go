package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// HealthPenaltySystem は健康状態が変化したときに EquipmentChanged を発行するシステム
// TemperatureSystem の後に実行する必要がある
type HealthPenaltySystem struct{}

// String はシステム名を返す
func (sys HealthPenaltySystem) String() string {
	return "HealthPenaltySystem"
}

// Update は健康ペナルティが変化したエンティティに EquipmentChanged を発行する
func (sys *HealthPenaltySystem) Update(world w.World) error {
	world.Manager.Join(
		world.Components.HealthStatus,
		world.Components.Attributes,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		hs := world.Components.HealthStatus.Get(entity).(*gc.HealthStatus)

		// ペナルティが変化した場合のみ EquipmentChanged を発行する
		if hs.HasModifierChanged() {
			entity.AddComponent(world.Components.EquipmentChanged, &gc.EquipmentChanged{})
		}
	}))

	return nil
}
