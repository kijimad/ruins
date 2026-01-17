package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MergeInventoryItem はバックパック内の指定された名前のStackableアイテムをすべて1つに統合する
func MergeInventoryItem(world w.World, itemName string) error {
	// 同名のStackableアイテムをすべて取得
	var stackableItems []ecs.Entity
	world.Manager.Join(
		world.Components.Stackable,
		world.Components.ItemLocationInBackpack,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name == itemName {
			stackableItems = append(stackableItems, entity)
		}
	}))

	// 0個または1個の場合は統合不要
	if len(stackableItems) <= 1 {
		return nil
	}

	// 最初のアイテムに統合する
	targetEntity := stackableItems[0]
	for i := 1; i < len(stackableItems); i++ {
		itemToMerge := stackableItems[i]
		itemComp := world.Components.Item.Get(itemToMerge).(*gc.Item)

		if err := ChangeItemCount(world, targetEntity, itemComp.Count); err != nil {
			return fmt.Errorf("数量統合エラー: %w", err)
		}

		// 統合元のアイテムエンティティを削除
		world.Manager.DeleteEntity(itemToMerge)
	}

	return nil
}

// ChangeStackableCount は指定した名前のStackableアイテムの数量を変更する
// amount が正の場合は増加、負の場合は減少する
func ChangeStackableCount(world w.World, name string, amount int) error {
	if amount == 0 {
		return fmt.Errorf("amount must not be zero")
	}

	// 既存のアイテムを検索する
	entity, found := FindStackableInInventory(world, name)
	if found {
		return ChangeItemCount(world, entity, amount)
	}

	// 存在しない場合
	if amount < 0 {
		// 減らす操作で存在しない場合はエラー
		return fmt.Errorf("stackable item not found: %s", name)
	}

	// 増やす操作で存在しない場合は新規作成する
	_, err := SpawnItem(world, name, amount, gc.ItemLocationInBackpack)
	return err
}
