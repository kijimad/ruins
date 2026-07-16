package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// FindNearestEntity は条件を満たす最寄りのエンティティを全 GridElement から探す（汎用・全走査）。
// selfは検索対象から除外されるエンティティで、Deadエンティティも自動的に除外する。
// matchはターゲット候補を絞り込む述語。見つからない場合はnilを返す。
// キャラクター探索のホットパスは FindNearestCharacter を使うこと（本関数は床・壁タイルも走査する）。
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

// FindNearestCharacter は条件を満たす最寄りのキャラクター（Player/AI/隊員）を探す。
//
// 床・壁・void も GridElement を持つエンティティのため、全 GridElement を走査するとマップ全タイル
// （数千個）を毎回舐めることになる。空間インデックスの Characters（キャラクター位置）を候補にして
// O(タイル数+キャラ数) → O(キャラ数) に削減する。AI のターゲット探索（敵・味方・仲間）はすべて
// キャラクター対象なので、この関数を使う。
//
// タイ（同距離）は entity.ID() の小さい方を選び、マップ反復順に依存しない決定的な結果を返す
// （固定 seed の再現性を保つ）。位置は live な GridElement を読むため、インデックスキーが多少
// ずれても正しい。空間インデックス未構築時は FindNearestEntity（全走査）にフォールバックする
// （match がキャラクター以外を弾く前提）。
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
		dist := geometry.ChebyshevDistance(int(from.X), int(from.Y), int(grid.X), int(grid.Y))
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
