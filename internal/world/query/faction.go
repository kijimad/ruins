package query

import (
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
