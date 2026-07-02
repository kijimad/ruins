package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// DismissSquadMember は隊員を解雇する。エンティティを削除する
func DismissSquadMember(world w.World, member ecs.Entity) error {
	if !member.HasComponent(world.Components.SquadMember) {
		return fmt.Errorf("エンティティは隊員ではありません")
	}
	world.Manager.DeleteEntity(member)
	return nil
}

// SetAIPolicy は隊員のポリシーを変更する
func SetAIPolicy(world w.World, member ecs.Entity, policy gc.AIPolicy) error {
	if !member.HasComponent(world.Components.AIPolicy) {
		return fmt.Errorf("エンティティにAIPolicyがありません")
	}
	current := world.Components.AIPolicy.Get(member).(*gc.AIPolicy)
	*current = policy
	return nil
}
