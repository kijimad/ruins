package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/lib/components"
	w "github.com/kijimaD/ruins/lib/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// MergeStackableIntoInventory は既存のバックパック内Stackableアイテムと統合するか新規追加する
// Stackableコンポーネントを持つ場合は既存と数量統合、それ以外は個別アイテムとして追加
func MergeStackableIntoInventory(world w.World, newItemEntity ecs.Entity, itemName string) error {
	// Stackableコンポーネントがない場合は何もしない（個別アイテムとして扱う）
	if !newItemEntity.HasComponent(world.Components.Stackable) {
		return nil
	}

	// 既存の同名Stackableアイテムを探してマージ
	existingEntity, found := FindStackableInInventory(world, itemName)
	if found && existingEntity != newItemEntity {
		// 新しいアイテムの数量を既存のアイテムに追加する
		existingItemComp := world.Components.Item.Get(existingEntity).(*gc.Item)
		newItemComp := world.Components.Item.Get(newItemEntity).(*gc.Item)

		// 数量を統合
		existingItemComp.Count += newItemComp.Count

		// 新しいアイテムエンティティを削除
		world.Manager.DeleteEntity(newItemEntity)
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
