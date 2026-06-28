package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SquadMembers はリーダーに所属する生存隊員の一覧を返す
func SquadMembers(world w.World, leader ecs.Entity) []ecs.Entity {
	var members []ecs.Entity
	world.Manager.Join(
		world.Components.SquadMember,
		world.Components.FactionAlly,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		sm := world.Components.SquadMember.Get(entity).(*gc.SquadMember)
		if sm.Leader == leader {
			members = append(members, entity)
		}
	}))
	return members
}

// SquadMemberCount はリーダーに所属する生存隊員数を返す
func SquadMemberCount(world w.World, leader ecs.Entity) int {
	return len(SquadMembers(world, leader))
}

// SquadPolicy は隊員の現在のポリシーを返す。
// 隊員でない場合はデフォルト値を返す
func SquadPolicy(world w.World, member ecs.Entity) gc.SquadPolicy {
	if !member.HasComponent(world.Components.SquadPolicy) {
		return gc.DefaultSquadPolicy()
	}
	return *world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
}

// IsSquadMember はエンティティが隊員かどうかを返す
func IsSquadMember(world w.World, entity ecs.Entity) bool {
	return entity.HasComponent(world.Components.SquadMember)
}

// SquadLeader は隊員のリーダーを返す。
// 隊員でない場合はゼロ値のEntityを返す
func SquadLeader(world w.World, member ecs.Entity) ecs.Entity {
	if !member.HasComponent(world.Components.SquadMember) {
		return ecs.Entity(0)
	}
	return world.Components.SquadMember.Get(member).(*gc.SquadMember).Leader
}
