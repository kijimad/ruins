package systems

import (
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// WeightDirtySystem はWeightDirtyマーカーが付いたエンティティの重量を再計算する。
// PlayerとStorageの両方に対応する
type WeightDirtySystem struct{}

// String はシステム名を返す
// w.Updater interfaceを実装する
func (sys WeightDirtySystem) String() string {
	return "WeightDirtySystem"
}

// Update はWeightDirtyフラグをチェックし、対象エンティティの重量を再計算する
// w.Updater interfaceを実装する
func (sys *WeightDirtySystem) Update(world w.World) error {
	// WeightDirtyマーカーが付いたエンティティを収集してフラグをクリアする
	var changedEntities []ecs.Entity
	world.Manager.Join(
		world.Components.WeightDirty,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		changedEntities = append(changedEntities, entity)
		entity.RemoveComponent(world.Components.WeightDirty)
	}))

	// 変動のあったエンティティの重量を再計算する
	for _, entity := range changedEntities {
		query.UpdateWeightCapacity(world, entity)
	}

	return nil
}
