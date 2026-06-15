package worldhelper

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

// ChangeItemCount は対象エンティティの個数を変更する。
// Stackableならカウントを増減し、非Stackableなら delta=-1 でエンティティを削除する。
// 個数が0以下になった場合はエンティティを削除する。
func ChangeItemCount(world w.World, entity ecs.Entity, delta int) error {
	if delta == 0 {
		return fmt.Errorf("delta must not be zero")
	}

	if !entity.HasComponent(world.Components.Item) {
		return fmt.Errorf("entity does not have Item component")
	}

	currentCount := GetEntityCount(world, entity)
	newCount := currentCount + delta

	if newCount < 0 {
		return fmt.Errorf("アイテム数が不足しています: 現在=%d, 変更=%d, 結果=%d", currentCount, delta, newCount)
	}

	if newCount == 0 {
		world.Manager.DeleteEntity(entity)
	} else if entity.HasComponent(world.Components.Stackable) {
		world.Components.Stackable.Get(entity).(*gc.Stackable).Count = newCount
	}

	// インベントリ変動フラグを立てる
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(playerEntity ecs.Entity) {
		playerEntity.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
	}))

	return nil
}
