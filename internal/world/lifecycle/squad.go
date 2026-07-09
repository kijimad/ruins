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

// GetAI は隊員のAIコンポーネントを返す。コンポーネントがない場合はエラーを返す
func GetAI(world w.World, member ecs.Entity) (*gc.AI, error) {
	ai, ok := world.Components.AI.TryGet(member)
	if !ok {
		return nil, fmt.Errorf("エンティティにAIがありません")
	}
	return ai, nil
}
