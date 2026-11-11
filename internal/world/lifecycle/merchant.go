package lifecycle

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// PopulateStock は member の行の在庫を、その member 所有の LocationInStorage で持たせる。
// 在庫を持たない member には何もしない。生成時に一度だけ呼ぶ
func PopulateStock(world w.World, owner ecs.Entity, name string) error {
	member, err := raw.FindMember(world.Resources.RawMaster, name)
	if err != nil {
		return err
	}
	for _, item := range raw.PtrSlice(member.Stock) {
		if _, err := SpawnStorageItem(world, item.Name, item.Count, owner); err != nil {
			return fmt.Errorf("failed to stock item %s: %w", item.Name, err)
		}
	}
	return nil
}
