package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SquadMembers は生存している全隊員を返す
func SquadMembers(world w.World) []ecs.Entity {
	var members []ecs.Entity
	world.Manager.Join(
		world.Components.SquadMember,
		world.Components.FactionAlly,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if entity.HasComponent(world.Components.Dead) {
			return
		}
		members = append(members, entity)
	}))
	return members
}

// SquadMemberCount は生存隊員数を返す
func SquadMemberCount(world w.World) int {
	return len(SquadMembers(world))
}

// SquadMemberAt は指定座標にいる隊員を返す。
// 見つからなければ ok=false を返す
func SquadMemberAt(world w.World, x, y int) (ecs.Entity, bool) {
	for _, member := range SquadMembers(world) {
		grid := world.Components.GridElement.MustGet(member)
		if int(grid.X) == x && int(grid.Y) == y {
			return member, true
		}
	}
	return ecs.Entity(0), false
}

// GetAI は隊員のAIコンポーネントを返す。
// コンポーネントがない場合はnilを返す
func GetAI(world w.World, member ecs.Entity) *gc.AI {
	ai, ok := world.Components.AI.TryGet(member)
	if !ok {
		return nil
	}
	return ai
}

// IsSquadMember はエンティティが隊員かどうかを返す
func IsSquadMember(world w.World, entity ecs.Entity) bool {
	return entity.HasComponent(world.Components.SquadMember)
}
