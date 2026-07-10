package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// RequestStateChange はステート遷移イベントエンティティを作成する。既にイベントが存在する場合はエラーを返す
func RequestStateChange(world w.World, event gc.StateChangeRequest) error {
	var existing gc.StateChangeRequest
	world.Manager.Join(world.Components.StateChangeRequest).Visit(ecs.Visit(func(entity ecs.Entity) {
		// StateChangeRequestはインターフェース値を格納する特殊コンポーネントのため型付きのMustGetが使えない
		if req, ok := world.Components.StateChangeRequest.Get(entity).(gc.StateChangeRequest); ok {
			existing = req
		}
	}))
	if existing != nil {
		return fmt.Errorf("リクエストがすでに設定されています: %T → %T を設定しようとしました",
			existing, event)
	}
	entity := world.Manager.NewEntity()
	entity.AddComponent(world.Components.StateChangeRequest, event)
	return nil
}

// ConsumeStateChange はステート遷移イベントを読み取り、エンティティを削除する
func ConsumeStateChange(world w.World) gc.StateChangeRequest {
	var event gc.StateChangeRequest
	var eventEntity ecs.Entity
	world.Manager.Join(world.Components.StateChangeRequest).Visit(ecs.Visit(func(entity ecs.Entity) {
		// StateChangeRequestはインターフェース値を格納する特殊コンポーネントのため型付きのMustGetが使えない
		if req, ok := world.Components.StateChangeRequest.Get(entity).(gc.StateChangeRequest); ok {
			event = req
			eventEntity = entity
		}
	}))
	if event == nil {
		return nil
	}
	world.Manager.DeleteEntity(eventEntity)
	return event
}
