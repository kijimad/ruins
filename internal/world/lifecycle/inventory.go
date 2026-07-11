package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
		world.World.RemoveEntity(entity)
	} else if world.Components.Stackable.Has(entity) {
		world.Components.Stackable.Get(entity).Count = newCount
	}

	// インベントリ変動フラグを立てる。
	var players []ecs.Entity
	playerQuery := ecs.NewFilter1[gc.Player](world.World).Query()
	for playerQuery.Next() {
		players = append(players, playerQuery.Entity())
	}
	for _, playerEntity := range players {
		ensureMarker(world, world.Components.WeightDirty, playerEntity, &gc.WeightDirty{})
	}

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
