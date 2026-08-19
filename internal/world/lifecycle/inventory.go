package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// ChangeItemCount は entity が属するスタックの個数を delta だけ増減する。
// 個数は保存しないので、増加は同種を delta 個生成し、減少は同一スタックのエンティティを
// |delta| 個取り除く。1個1エンティティのため、これがスタック数の変更になる。
func ChangeItemCount(world w.World, entity ecs.Entity, delta int) error {
	if delta == 0 {
		return fmt.Errorf("delta must not be zero")
	}

	if delta > 0 {
		if err := addToStack(world, entity, delta); err != nil {
			return err
		}
	} else {
		if err := removeFromStack(world, entity, -delta); err != nil {
			return err
		}
	}

	// インベントリ変動フラグを立てる。反復中の構造変更を避けるため、
	// プレイヤーを一旦集めてからマーカーを付ける。
	var players []ecs.Entity
	playerQuery := ecs.NewFilter1[gc.Player](world.ECS).Query()
	for playerQuery.Next() {
		players = append(players, playerQuery.Entity())
	}
	for _, playerEntity := range players {
		ensureMarker(world, world.Components.WeightDirty, playerEntity, &gc.WeightDirty{})
	}

	return nil
}

// removeFromStack は entity と同一スタックのエンティティを n 個取り除く。在庫が足りなければエラー。
func removeFromStack(world w.World, entity ecs.Entity, n int) error {
	members := query.StackMembers(world, entity)
	if n > len(members) {
		return fmt.Errorf("insufficient item count: have=%d, remove=%d", len(members), n)
	}
	for i := range n {
		world.ECS.RemoveEntity(members[i])
	}
	return nil
}

// addToStack は entity と同じ位置へ同種のアイテムを n 個生成する。腐敗品は生成直後の鮮度になる。
func addToStack(world w.World, entity ecs.Entity, n int) error {
	id := world.Components.RawID.Get(entity).ID
	switch {
	case world.Components.LocationInBackpack.Has(entity):
		owner := world.Components.LocationInBackpack.Get(entity).Owner
		for range n {
			e, err := spawnItemBase(world, id)
			if err != nil {
				return err
			}
			if err := MoveToBackpack(world, e, owner); err != nil {
				return err
			}
		}
	case world.Components.LocationInStorage.Has(entity):
		owner := world.Components.LocationInStorage.Get(entity).Owner
		for range n {
			e, err := spawnItemBase(world, id)
			if err != nil {
				return err
			}
			if err := MoveToStorage(world, e, owner); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("cannot add to stack: entity has no backpack or storage location")
	}
	return nil
}

// ChangeStackCount は指定した名前のスタックアイテムの数量を変更する。
// amount が正の場合は増加、負の場合は減少する
func ChangeStackCount(world w.World, name string, amount int) error {
	if amount == 0 {
		return fmt.Errorf("amount must not be zero")
	}

	entity, found := query.FindStackInInventory(world, name)
	if found {
		return ChangeItemCount(world, entity, amount)
	}

	if amount < 0 {
		return fmt.Errorf("stackable item not found: %s", name)
	}

	_, err := SpawnBackpackItem(world, name, amount)
	return err
}
