package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// FindStackableInInventory は名前でバックパック内のStackableアイテムを検索する
func FindStackableInInventory(world w.World, name string) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	world.Manager.Join(
		world.Components.Stackable,
		world.Components.LocationInBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if found {
			return
		}
		itemName := world.Components.Name.Get(entity).(*gc.Name)
		if itemName.Name == name {
			foundEntity = entity
			found = true
		}
	}))

	return foundEntity, found
}

// FindAmmoInInventory は口径タグでバックパック内の弾薬アイテムを検索する
func FindAmmoInInventory(world w.World, ammoTag string) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	world.Manager.Join(
		world.Components.Stackable,
		world.Components.LocationInBackpack,
		world.Components.Ammo,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if found {
			return
		}
		ammo := world.Components.Ammo.Get(entity).(*gc.Ammo)
		if ammo.AmmoTag == ammoTag {
			foundEntity = entity
			found = true
		}
	}))

	return foundEntity, found
}

// GetEntityCount はエンティティの個数を返す。
// Stackableであれば Stackable.Count を返し、そうでなければ1を返す。
func GetEntityCount(world w.World, entity ecs.Entity) int {
	if entity.HasComponent(world.Components.Stackable) {
		return world.Components.Stackable.Get(entity).(*gc.Stackable).Count
	}
	return 1
}

// FormatItemName はアイテムエンティティから名前と個数を取得してフォーマットする。
// 名前はNameコンポーネントから取得し、見つからない場合は "Unknown Item" を返す。
// 個数が1以下の場合は名前のみ、2以上の場合は "名前(個数)" の形式で返す
func FormatItemName(world w.World, itemEntity ecs.Entity) string {
	name := "Unknown Item"
	if nameComp := world.Components.Name.Get(itemEntity); nameComp != nil {
		n := nameComp.(*gc.Name)
		name = n.Name
	}

	count := GetEntityCount(world, itemEntity)

	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s(%d個)", name, count)
}
