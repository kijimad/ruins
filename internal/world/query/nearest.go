package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FindNearestEntity は条件を満たす最寄りのエンティティを全 GridElement から探す。
// selfは検索対象から除外されるエンティティで、Deadエンティティも自動的に除外する。
// matchはターゲット候補を絞り込む述語。見つからない場合はnilを返す。
// キャラクター探索のホットパスは FindNearestCharacter を使うこと。本関数は床・壁タイルも走査する。
func FindNearestEntity(world w.World, self ecs.Entity, from *gc.GridElement, match func(ecs.Entity) bool) (*ecs.Entity, *gc.GridElement, int) {
	var nearestEntity *ecs.Entity
	var nearestGrid *gc.GridElement
	nearestDist := -1

	nearestQuery := ecs.NewFilter1[gc.GridElement](world.ECS).Query()
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
		dist := geometry.ChebyshevDistance(from.Coord, grid.Coord)
		if nearestDist < 0 || dist < nearestDist {
			e := entity
			nearestEntity = &e
			nearestGrid = grid
			nearestDist = dist
		}
	}

	return nearestEntity, nearestGrid, nearestDist
}

// FindNearestCharacter は条件を満たす最寄りのキャラクターを探す。
//
// 床・壁・void も GridElement を持つエンティティのため、全 GridElement を走査するとマップ全タイル
// を毎回舐めることになる。空間インデックスのキャラクター位置を候補にして
// O(タイル数+キャラ数) → O(キャラ数) に削減する。AI のターゲット探索はすべて
// キャラクター対象なので、この関数を使う。
//
// 同距離は entity.ID() の小さい方を選び、マップ反復順に依存しない決定的な結果を返して
// 固定 seed の再現性を保つ。位置は live な GridElement を読むため、インデックスキーが多少
// ずれても正しい。インデックスが使えないときは全走査にフォールバックする。
func FindNearestCharacter(world w.World, self ecs.Entity, from *gc.GridElement, match func(ecs.Entity) bool) (*ecs.Entity, *gc.GridElement, int) {
	si := GetSpatialIndex(world)
	if si == nil || si.Characters == nil {
		return FindNearestEntity(world, self, from, match)
	}

	var nearest ecs.Entity
	var nearestGrid *gc.GridElement
	nearestDist := -1
	found := false

	for _, entity := range si.Characters {
		if entity == self || !world.ECS.Alive(entity) || world.Components.Dead.Has(entity) {
			continue
		}
		if !match(entity) {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		dist := geometry.ChebyshevDistance(from.Coord, grid.Coord)
		if !found || dist < nearestDist || (dist == nearestDist && entity.ID() < nearest.ID()) {
			nearest = entity
			nearestGrid = grid
			nearestDist = dist
			found = true
		}
	}

	if !found {
		return nil, nil, -1
	}
	return &nearest, nearestGrid, nearestDist
}
