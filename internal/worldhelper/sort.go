package worldhelper

import (
	"sort"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SortEntities はエンティティリストをソートする汎用関数
// Nameコンポーネントを持つエンティティを名前順でソートする
// Nameコンポーネントを持っていないエンティティはスキップされる
func SortEntities(world w.World, entities []ecs.Entity) []ecs.Entity {
	if len(entities) == 0 {
		return entities
	}

	// ソート用の構造体
	type entityWithName struct {
		entity ecs.Entity
		name   string
	}

	// Nameコンポーネントを持つエンティティだけを抽出
	withNames := make([]entityWithName, 0, len(entities))
	for _, entity := range entities {
		if entity.HasComponent(world.Components.Name) {
			nameComp := world.Components.Name.Get(entity)
			if nameComp != nil {
				name := nameComp.(*gc.Name)
				withNames = append(withNames, entityWithName{
					entity: entity,
					name:   name.Name,
				})
			}
		}
	}

	// 名前順でソート
	sort.Slice(withNames, func(i, j int) bool {
		return withNames[i].name < withNames[j].name
	})

	// ソート結果を返す
	result := make([]ecs.Entity, len(withNames))
	for i, item := range withNames {
		result[i] = item.entity
	}

	return result
}
