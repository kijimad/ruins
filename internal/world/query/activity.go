package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetActivity は指定されたエンティティの現在のアクティビティを取得する。
// 死亡・未保持のエンティティにはnilを返す
func GetActivity(world w.World, entity ecs.Entity) *gc.Activity {
	if !world.ECS.Alive(entity) || !world.Components.Activity.Has(entity) {
		return nil
	}
	return world.Components.Activity.Get(entity)
}

// HasActivity は指定されたエンティティがアクティビティを実行中かを返す
func HasActivity(world w.World, entity ecs.Entity) bool {
	activity := GetActivity(world, entity)
	return activity != nil && activity.State == gc.ActivityStateRunning
}

// SetActivity はエンティティにアクティビティを設定する。既存があれば上書きする
func SetActivity(world w.World, entity ecs.Entity, activity *gc.Activity) {
	gc.Upsert(world.Components.Activity, entity, activity)
}

// RemoveActivity はエンティティからアクティビティを削除する
func RemoveActivity(world w.World, entity ecs.Entity) {
	if world.Components.Activity.Has(entity) {
		world.Components.Activity.Remove(entity)
	}
}
