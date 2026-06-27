package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SpawnVisualEffect はエンティティの位置にエフェクト専用エンティティを生成する
func SpawnVisualEffect(target ecs.Entity, effect gc.VisualEffect, world w.World) {
	if !target.HasComponent(world.Components.GridElement) {
		return
	}

	gridElement := world.Components.GridElement.Get(target).(*gc.GridElement)

	effectEntity := world.Manager.NewEntity()
	effectEntity.AddComponent(world.Components.GridElement, &gc.GridElement{
		X: gridElement.X,
		Y: gridElement.Y,
	})
	effectEntity.AddComponent(world.Components.VisualEffect, &gc.VisualEffects{
		Effects: []gc.VisualEffect{effect},
	})
}
