package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// TransferItem はアイテムの位置を変更する
func TransferItem(world w.World, itemEntity ecs.Entity, fromLocation, toLocation gc.ItemLocationType) {
	// 現在の位置コンポーネントを削除
	switch fromLocation {
	case gc.ItemLocationInPlayerBackpack:
		if itemEntity.HasComponent(world.Components.ItemLocationInPlayerBackpack) {
			itemEntity.RemoveComponent(world.Components.ItemLocationInPlayerBackpack)
		}
	case gc.ItemLocationOnField:
		if itemEntity.HasComponent(world.Components.ItemLocationOnField) {
			itemEntity.RemoveComponent(world.Components.ItemLocationOnField)
		}
	}

	// 新しい位置コンポーネントを追加
	switch toLocation {
	case gc.ItemLocationInPlayerBackpack:
		itemEntity.AddComponent(world.Components.ItemLocationInPlayerBackpack, &gc.LocationInPlayerBackpack{})
	case gc.ItemLocationOnField:
		itemEntity.AddComponent(world.Components.ItemLocationOnField, &gc.LocationOnField{})
	}
}

// GetInventoryItems はバックパック内のアイテム一覧を取得する
func GetInventoryItems(world w.World) []ecs.Entity {
	var items []ecs.Entity

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		items = append(items, entity)
	}))

	return items
}

// GetInventoryStackables はバックパック内のスタック可能アイテム一覧を取得する
func GetInventoryStackables(world w.World) []ecs.Entity {
	var stackables []ecs.Entity

	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInPlayerBackpack,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		stackables = append(stackables, entity)
	}))

	return stackables
}

// FindStackableInInventory は名前でバックパック内のStackableアイテムを検索する
func FindStackableInInventory(world w.World, name string) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInPlayerBackpack,
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

// FindItemInInventory は名前でバックパック内のアイテムを検索する
func FindItemInInventory(world w.World, itemName string) (ecs.Entity, bool) {
	var foundEntity ecs.Entity
	var found bool

	world.Manager.Join(
		world.Components.Item,
		world.Components.ItemLocationInPlayerBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		if found {
			return
		}
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == itemName {
			foundEntity = entity
			found = true
		}
	}))

	return foundEntity, found
}

// ChangeItemCount は対象アイテムの個数を変更する。Stackable/非Stackableに関わらず使用できる。
// 使用、売却、破棄、拾得など、個数を変更する全ての用途で使用する。
// 個数が0以下になった場合はエンティティを削除する。
func ChangeItemCount(world w.World, itemEntity ecs.Entity, delta int) error {
	if delta == 0 {
		return fmt.Errorf("delta must not be zero")
	}

	if !itemEntity.HasComponent(world.Components.Item) {
		return fmt.Errorf("entity does not have Item component")
	}

	item := world.Components.Item.Get(itemEntity).(*gc.Item)
	newCount := item.Count + delta

	// 減少の場合、結果がマイナスになるならエラー
	if newCount < 0 {
		return fmt.Errorf("アイテム数が不足しています: 現在=%d, 変更=%d, 結果=%d", item.Count, delta, newCount)
	}

	item.Count = newCount

	// 個数が0になったらエンティティを削除
	if item.Count == 0 {
		world.Manager.DeleteEntity(itemEntity)
	}

	// インベントリ変動フラグを立てる
	// TODO(kijima): 移動ヘルパーで書くようにしたい
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(playerEntity ecs.Entity) {
		playerEntity.AddComponent(world.Components.InventoryChanged, &gc.InventoryChanged{})
	}))

	return nil
}
