package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FindNearestEntity は条件を満たす最寄りのエンティティを探す。
// matchはターゲット候補を絞り込む述語で、Deadエンティティは自動的に除外する。
// 見つからない場合はnilを返す
func FindNearestEntity(world w.World, from *gc.GridElement, match func(ecs.Entity) bool) (*ecs.Entity, *gc.GridElement, int) {
	var nearestEntity *ecs.Entity
	var nearestGrid *gc.GridElement
	nearestDist := -1

	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		if !match(entity) {
			return
		}
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		dist := geometry.ChebyshevDistance(int(from.X), int(from.Y), int(grid.X), int(grid.Y))
		if nearestDist < 0 || dist < nearestDist {
			e := entity
			nearestEntity = &e
			nearestGrid = grid
			nearestDist = dist
		}
	}))

	return nearestEntity, nearestGrid, nearestDist
}
