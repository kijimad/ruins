package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// SquadMembers は生存している全隊員を返す
func SquadMembers(world w.World) []ecs.Entity {
	var members []ecs.Entity
	membersQuery := ecs.NewFilter2[gc.SquadMember, gc.FactionAllyData](world.World).Query()
	for membersQuery.Next() {
		entity := membersQuery.Entity()
		if world.Components.Dead.Has(entity) {
			continue
		}
		members = append(members, entity)
	}
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
		grid := world.Components.GridElement.Get(member)
		if int(grid.X) == x && int(grid.Y) == y {
			return member, true
		}
	}
	return ecs.Entity{}, false
}

// GetSquadAI は隊員のSquadAIコンポーネントを返す。
// コンポーネントがない場合はnilを返す
func GetSquadAI(world w.World, member ecs.Entity) *gc.SquadAI {
	comp := world.Components.SquadAI.Get(member)
	if comp == nil {
		return nil
	}
	return comp
}

// IsSquadMember はエンティティが隊員かどうかを返す
func IsSquadMember(world w.World, entity ecs.Entity) bool {
	return world.Components.SquadMember.Has(entity)
}
