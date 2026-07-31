package query

import (
	"cmp"
	"slices"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SortEntities はエンティティリストをソートする汎用関数
// Nameコンポーネントを持つエンティティを名前順でソートする
// Nameコンポーネントを持っていないエンティティはスキップされる
func SortEntities(world w.World, entities []ecs.Entity) []ecs.Entity {
	if len(entities) == 0 {
		return entities
	}

	named := make([]ecs.Entity, 0, len(entities))
	for _, entity := range entities {
		if world.Components.Name.Has(entity) {
			named = append(named, entity)
		}
	}

	slices.SortStableFunc(named, func(a, b ecs.Entity) int {
		return cmp.Compare(world.Components.Name.Get(a).Name, world.Components.Name.Get(b).Name)
	})

	return named
}
