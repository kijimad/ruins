package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FindNearestEntity は条件を満たす最寄りのエンティティを探す。
// selfは検索対象から除外されるエンティティで、Deadエンティティも自動的に除外する。
// matchはターゲット候補を絞り込む述語。見つからない場合はnilを返す
func FindNearestEntity(world w.World, self ecs.Entity, from *gc.GridElement, match func(ecs.Entity) bool) (*ecs.Entity, *gc.GridElement, int) {
	var nearestEntity *ecs.Entity
	var nearestGrid *gc.GridElement
	nearestDist := -1

	nearestQuery := ecs.NewFilter1[gc.GridElement](world.World).Query()
	for nearestQuery.Next() {
		entity := nearestQuery.Entity()
		if entity == self {
			continue
		}
		if world.Components.Dead.Has(entity) {
			continue
		}
		if !match(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		dist := geometry.ChebyshevDistance(int(from.X), int(from.Y), int(grid.X), int(grid.Y))
		if nearestDist < 0 || dist < nearestDist {
			e := entity
			nearestEntity = &e
			nearestGrid = grid
			nearestDist = dist
		}
	}

	return nearestEntity, nearestGrid, nearestDist
}
