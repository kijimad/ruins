package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// Relation はエンティティ間の派閥関係を表す
type Relation string

// 派閥間の関係性
const (
	RelationHostile  Relation = "hostile"
	RelationFriendly Relation = "friendly"
	RelationNeutral  Relation = "neutral"
)

// IsEnemy はエンティティが敵性派閥かを返す
func IsEnemy(world w.World, e ecs.Entity) bool {
	return world.Components.FactionEnemy.Has(e)
}

// IsAlly はエンティティが味方派閥かを返す
func IsAlly(world w.World, e ecs.Entity) bool {
	return world.Components.FactionAlly.Has(e)
}

// IsNeutral はエンティティが中立派閥かを返す
func IsNeutral(world w.World, e ecs.Entity) bool {
	return world.Components.FactionNeutral.Has(e)
}

// FactionRelation は2つのエンティティ間の派閥関係を返す。
// FactionAlly同士はFriendly、FactionEnemy同士もFriendly、
// FactionAllyとFactionEnemyはHostile、それ以外はNeutral
func FactionRelation(world w.World, a, b ecs.Entity) Relation {
	aEnemy := IsEnemy(world, a)
	bEnemy := IsEnemy(world, b)
	aAlly := IsAlly(world, a)
	bAlly := IsAlly(world, b)

	if aEnemy && bAlly || aAlly && bEnemy {
		return RelationHostile
	}
	if aEnemy && bEnemy || aAlly && bAlly {
		return RelationFriendly
	}
	return RelationNeutral
}

// IsAreaSafe はアクターの周囲に敵対エンティティがいないかを返す。休息・睡眠・読書など継続
// アクティビティが毎ターン安全を確かめるのに使う。座標を持たなければ判定できず危険とみなす。
func IsAreaSafe(world w.World, actor ecs.Entity) bool {
	if !world.Components.GridElement.Has(actor) {
		return false
	}
	gridElement := world.Components.GridElement.Get(actor)
	actorX, actorY := int(gridElement.X), int(gridElement.Y)

	safeRadius := 1
	hasHostile := false

	areaQuery := ActiveFilter1[gc.GridElement](world).Query()
	for areaQuery.Next() {
		entity := areaQuery.Entity()
		if hasHostile {
			continue
		}
		if FactionRelation(world, actor, entity) != RelationHostile {
			continue
		}
		grid := world.Components.GridElement.Get(entity)
		dx, dy := int(grid.X)-actorX, int(grid.Y)-actorY
		if dx >= -safeRadius && dx <= safeRadius && dy >= -safeRadius && dy <= safeRadius {
			hasHostile = true
		}
	}

	return !hasHostile
}
