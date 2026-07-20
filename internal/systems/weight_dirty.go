package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
	// WeightDirtyマーカーが付いたエンティティを収集する。
	var changedEntities []ecs.Entity
	weightDirtyQuery := query.ActiveFilter1[gc.WeightDirty](world).Query()
	for weightDirtyQuery.Next() {
		changedEntities = append(changedEntities, weightDirtyQuery.Entity())
	}

	// 変動のあったエンティティのフラグをクリアして重量を再計算する
	for _, entity := range changedEntities {
		world.Components.WeightDirty.Remove(entity)
		query.UpdateWeightCapacity(world, entity)
	}

	return nil
}
