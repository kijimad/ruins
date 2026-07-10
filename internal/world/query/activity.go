package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetActivity は指定されたエンティティの現在のアクティビティを取得する
func GetActivity(world w.World, entity ecs.Entity) *gc.Activity {
	comp := world.Components.Activity.Get(entity)
	if comp == nil {
		return nil
	}
	return comp.(*gc.Activity)
}

// HasActivity は指定されたエンティティがアクティビティを実行中かを返す
func HasActivity(world w.World, entity ecs.Entity) bool {
	activity := GetActivity(world, entity)
	return activity != nil && activity.State == gc.ActivityStateRunning
}

// SetActivity はエンティティにアクティビティを設定する
func SetActivity(world w.World, entity ecs.Entity, activity *gc.Activity) {
	if world.Components.Activity.Get(entity) != nil {
		// 既存のアクティビティを上書き
		world.Components.Activity.Remove(entity)
	}
	world.Components.Activity.Add(entity, activity)
}

// RemoveActivity はエンティティからアクティビティを削除する
func RemoveActivity(world w.World, entity ecs.Entity) {
	if world.Components.Activity.Get(entity) != nil {
		world.Components.Activity.Remove(entity)
	}
}
