package lifecycle

import (
	"fmt"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ChangeStackableCount は指定した名前のStackableアイテムの数量を変更する
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
