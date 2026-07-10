package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetActivity は指定されたエンティティの現在のアクティビティを取得する
func GetActivity(world w.World, entity ecs.Entity) *gc.Activity {
	activity, ok := world.Components.Activity.TryGet(entity)
	if !ok {
		return nil
	}
	return activity
}

// HasActivity は指定されたエンティティがアクティビティを実行中かを返す
func HasActivity(world w.World, entity ecs.Entity) bool {
	activity := GetActivity(world, entity)
	return activity != nil && activity.State == gc.ActivityStateRunning
}

// SetActivity はエンティティにアクティビティを設定する
func SetActivity(world w.World, entity ecs.Entity, activity *gc.Activity) {
	if entity.HasComponent(world.Components.Activity) {
		// 既存のアクティビティを上書き
		entity.RemoveComponent(world.Components.Activity)
	}
	world.Components.Activity.Add(entity, activity)
}

// RemoveActivity はエンティティからアクティビティを削除する
func RemoveActivity(world w.World, entity ecs.Entity) {
	if entity.HasComponent(world.Components.Activity) {
		entity.RemoveComponent(world.Components.Activity)
	}
}
