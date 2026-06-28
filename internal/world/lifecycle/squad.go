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

// SetSquadPolicy は隊員のポリシーを変更する
func SetSquadPolicy(world w.World, member ecs.Entity, policy gc.SquadPolicy) error {
	if !member.HasComponent(world.Components.SquadPolicy) {
		return fmt.Errorf("エンティティにSquadPolicyがありません")
	}
	current := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	*current = policy
	return nil
}

// SetPositionPolicy は隊員の位置ポリシーを変更する
func SetPositionPolicy(world w.World, member ecs.Entity, policy gc.PositionPolicy) error {
	if !member.HasComponent(world.Components.SquadPolicy) {
		return fmt.Errorf("エンティティにSquadPolicyがありません")
	}
	current := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	current.Position = policy
	return nil
}

// SetSquadMemberActive は隊員の同行/待機状態を切り替える
func SetSquadMemberActive(world w.World, member ecs.Entity, active bool) error {
	if !member.HasComponent(world.Components.SquadMember) {
		return fmt.Errorf("エンティティは隊員ではありません")
	}
	sm := world.Components.SquadMember.Get(member).(*gc.SquadMember)
	sm.Active = active
	return nil
}

// SetCombatPolicy は隊員の戦闘ポリシーを変更する
func SetCombatPolicy(world w.World, member ecs.Entity, policy gc.CombatPolicy) error {
	if !member.HasComponent(world.Components.SquadPolicy) {
		return fmt.Errorf("エンティティにSquadPolicyがありません")
	}
	current := world.Components.SquadPolicy.Get(member).(*gc.SquadPolicy)
	current.Combat = policy
	return nil
}
