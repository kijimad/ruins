package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// DismissSquadMember は隊員を解雇する。エンティティを削除する
func DismissSquadMember(world w.World, member ecs.Entity) error {
	if !world.Components.SquadMember.Has(member) {
		return fmt.Errorf("エンティティは隊員ではありません")
	}
	world.World.RemoveEntity(member)
	return nil
}

// GetAI は隊員のSquadAIコンポーネントを返す。コンポーネントがない場合はエラーを返す
func GetAI(world w.World, member ecs.Entity) (*gc.SquadAI, error) {
	comp := world.Components.SquadAI.Get(member)
	if comp == nil {
		return nil, fmt.Errorf("エンティティにAIがありません")
	}
	return comp, nil
}
