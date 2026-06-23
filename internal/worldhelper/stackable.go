package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

type mergeLocation int

const (
	mergeInBackpack mergeLocation = iota
	mergeInStorage
)

// mergeStackableItems は指定ロケーション内の同一Owner配下にある同名Stackableアイテムを1つに統合する
func mergeStackableItems(world w.World, itemName string, loc mergeLocation, owner ecs.Entity) error {
	var locationComp ecs.DataComponent
	switch loc {
	case mergeInBackpack:
		locationComp = world.Components.LocationInBackpack
	case mergeInStorage:
		locationComp = world.Components.LocationInStorage
	}

	var stackableItems []ecs.Entity
	world.Manager.Join(
		world.Components.Stackable,
		locationComp,
		world.Components.Name,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		name := world.Components.Name.Get(entity).(*gc.Name)
		if name.Name != itemName {
			return
		}
		// Ownerが一致するもののみ統合対象にする
		switch l := locationComp.Get(entity).(type) {
		case *gc.LocationInBackpack:
			if l.Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
		case *gc.LocationInStorage:
			if l.Owner == owner {
				stackableItems = append(stackableItems, entity)
			}
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
		mergeCount := GetEntityCount(world, itemToMerge)

		if err := ChangeItemCount(world, targetEntity, mergeCount); err != nil {
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
	_, err := SpawnBackpackItem(world, name, amount)
	return err
}
