package worldhelper

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// GetCurrentActivity は指定されたエンティティの現在のアクティビティを取得する
func GetCurrentActivity(world w.World, entity ecs.Entity) *gc.CurrentActivity {
	comp := world.Components.CurrentActivity.Get(entity)
	if comp == nil {
		return nil
	}
	return comp.(*gc.CurrentActivity)
}

// HasActivity は指定されたエンティティがアクティビティを実行中かを返す
func HasActivity(world w.World, entity ecs.Entity) bool {
	activity := GetCurrentActivity(world, entity)
	return activity != nil && activity.State == gc.ActivityStateRunning
}

// SetCurrentActivity はエンティティにアクティビティを設定する
func SetCurrentActivity(world w.World, entity ecs.Entity, activity *gc.CurrentActivity) {
	if world.Components.CurrentActivity.Get(entity) != nil {
		// 既存のアクティビティを上書き
		entity.RemoveComponent(world.Components.CurrentActivity)
	}
	entity.AddComponent(world.Components.CurrentActivity, activity)
}

// RemoveCurrentActivity はエンティティからアクティビティを削除する
func RemoveCurrentActivity(world w.World, entity ecs.Entity) {
	if world.Components.CurrentActivity.Get(entity) != nil {
		entity.RemoveComponent(world.Components.CurrentActivity)
	}
}
