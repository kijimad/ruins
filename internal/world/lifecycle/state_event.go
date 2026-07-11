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
	existingQuery := ecs.NewFilter1[gc.StateChangeRequest](world.ECS).Query()
	for existingQuery.Next() {
		entity := existingQuery.Entity()
		existing = world.Components.StateChangeRequest.Get(entity)
	}
	if existing != nil {
		return fmt.Errorf("リクエストがすでに設定されています: %s → %s を設定しようとしました",
			existing.Kind, event.Kind)
	}
	entity := world.ECS.NewEntity()
	world.Components.StateChangeRequest.Add(entity, &event)
	return nil
}

// ConsumeStateChange はステート遷移イベントを読み取り、エンティティを削除する。
// イベントが無い場合は nil を返す
func ConsumeStateChange(world w.World) *gc.StateChangeRequest {
	var event *gc.StateChangeRequest
	var eventEntity ecs.Entity
	eventQuery := ecs.NewFilter1[gc.StateChangeRequest](world.ECS).Query()
	for eventQuery.Next() {
		entity := eventQuery.Entity()
		// Getはストレージへのポインタを返し、RemoveEntityで失効するため値をコピーする
		copied := *world.Components.StateChangeRequest.Get(entity)
		event = &copied
		eventEntity = entity
	}
	if event == nil {
		return nil
	}
	world.ECS.RemoveEntity(eventEntity)
	return event
}
