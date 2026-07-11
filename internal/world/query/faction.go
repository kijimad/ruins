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

// GetFactionKind はエンティティの派閥種別を返す。派閥を持たない場合は ok=false
func GetFactionKind(world w.World, e ecs.Entity) (gc.FactionKind, bool) {
	if !world.Components.Faction.Has(e) {
		return 0, false
	}
	return world.Components.Faction.Get(e).Kind, true
}

// IsEnemy はエンティティが敵性派閥かを返す
func IsEnemy(world w.World, e ecs.Entity) bool {
	k, ok := GetFactionKind(world, e)
	return ok && k == gc.FactionEnemy
}

// IsAlly はエンティティが味方派閥かを返す
func IsAlly(world w.World, e ecs.Entity) bool {
	k, ok := GetFactionKind(world, e)
	return ok && k == gc.FactionAlly
}

// IsNeutral はエンティティが中立派閥かを返す
func IsNeutral(world w.World, e ecs.Entity) bool {
	k, ok := GetFactionKind(world, e)
	return ok && k == gc.FactionNeutral
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
