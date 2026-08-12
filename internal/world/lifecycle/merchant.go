package lifecycle

import (
	"fmt"
	"math/rand/v2"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// merchantStockItems は商人が初期在庫として並べるアイテム。1品1個ずつ在庫実体にする
var merchantStockItems = []string{
	"wooden_sword",
	"handgun",
	"western_armor",
	"work_helmet",
	"leather_boots",
	"healing_potion",
	"army_shooting_manual",
}

// PopulateMerchantStock は商人の品揃えを決める。アイテムを商人所有の LocationInStorage で持たせる。
// 商人生成時に一度だけ呼ぶ
func PopulateMerchantStock(world w.World, merchant ecs.Entity, _ *rand.Rand) error {
	for _, itemName := range merchantStockItems {
		if _, err := SpawnStorageItem(world, itemName, 1, merchant); err != nil {
			return fmt.Errorf("failed to stock item %s: %w", itemName, err)
		}
	}

	return nil
}
