package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// RequestStateChange はステート遷移イベントエンティティを作成する。既にイベントが存在する場合はエラーを返す
func RequestStateChange(world w.World, event gc.StateChangeRequest) error {
	var existing *gc.StateChangeRequest
	world.Manager.Join(world.Components.StateChangeRequest).Visit(ecs.Visit(func(entity ecs.Entity) {
		existing = world.Components.StateChangeRequest.Get(entity)
	}))
	if existing != nil {
		return fmt.Errorf("リクエストがすでに設定されています: %s → %s を設定しようとしました",
			existing.Kind, event.Kind)
	}
	entity := world.World.NewEntity()
	world.Components.StateChangeRequest.Add(entity, &event)
	return nil
}

// ConsumeStateChange はステート遷移イベントを読み取り、エンティティを削除する。
// イベントが無い場合は nil を返す
func ConsumeStateChange(world w.World) *gc.StateChangeRequest {
	var event *gc.StateChangeRequest
	var eventEntity ecs.Entity
	world.Manager.Join(world.Components.StateChangeRequest).Visit(ecs.Visit(func(entity ecs.Entity) {
		event = world.Components.StateChangeRequest.Get(entity)
		eventEntity = entity
	}))
	if event == nil {
		return nil
	}
	world.World.RemoveEntity(eventEntity)
	return event
}
