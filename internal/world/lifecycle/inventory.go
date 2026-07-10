package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// ChangeItemCount は対象エンティティの個数を変更する。
// Stackableならカウントを増減し、非Stackableなら delta=-1 でエンティティを削除する。
// 個数が0以下になった場合はエンティティを削除する。
func ChangeItemCount(world w.World, entity ecs.Entity, delta int) error {
	if delta == 0 {
		return fmt.Errorf("delta must not be zero")
	}

	currentCount := query.GetEntityCount(world, entity)
	newCount := currentCount + delta

	if newCount < 0 {
		return fmt.Errorf("アイテム数が不足しています: 現在=%d, 変更=%d, 結果=%d", currentCount, delta, newCount)
	}

	if newCount == 0 {
		world.Manager.DeleteEntity(entity)
	} else if entity.HasComponent(world.Components.Stackable) {
		world.Components.Stackable.MustGet(entity).Count = newCount
	}

	// インベントリ変動フラグを立てる
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(playerEntity ecs.Entity) {
		gc.AddComponent(playerEntity, world.Components.WeightDirty, &gc.WeightDirty{})
	}))

	return nil
}

// ChangeStackableCount は指定した名前のStackableアイテムの数量を変更する。
// amount が正の場合は増加、負の場合は減少する
func ChangeStackableCount(world w.World, name string, amount int) error {
	if amount == 0 {
		return fmt.Errorf("amount must not be zero")
	}

	entity, found := query.FindStackableInInventory(world, name)
	if found {
		return ChangeItemCount(world, entity, amount)
	}

	if amount < 0 {
		return fmt.Errorf("stackable item not found: %s", name)
	}

	_, err := SpawnBackpackItem(world, name, amount)
	return err
}
