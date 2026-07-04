package query

import (
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Relation はエンティティ間の派閥関係を表す
type Relation string

// 派閥間の関係性
const (
	RelationHostile  Relation = "hostile"
	RelationFriendly Relation = "friendly"
	RelationNeutral  Relation = "neutral"
)

// FactionRelation は2つのエンティティ間の派閥関係を返す。
// FactionAlly同士はFriendly、FactionEnemy同士もFriendly、
// FactionAllyとFactionEnemyはHostile、それ以外はNeutral
func FactionRelation(world w.World, a, b ecs.Entity) Relation {
	aEnemy := a.HasComponent(world.Components.FactionEnemy)
	bEnemy := b.HasComponent(world.Components.FactionEnemy)
	aAlly := a.HasComponent(world.Components.FactionAlly)
	bAlly := b.HasComponent(world.Components.FactionAlly)

	if aEnemy && bAlly || aAlly && bEnemy {
		return RelationHostile
	}
	if aEnemy && bEnemy || aAlly && bAlly {
		return RelationFriendly
	}
	return RelationNeutral
}
