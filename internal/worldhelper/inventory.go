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
