package lifecycle

import (
	"fmt"
	"math/rand/v2"

	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// merchantStockItems は商人が初期在庫として並べるアイテムと個数。1個1エンティティで個数ぶん
// 生成し、同一スタックは店頭で1行に束ねられる。消耗品は複数個持たせ、スタック売買を試せるようにする
var merchantStockItems = []struct {
	Name  string
	Count int
}{
	{Name: "wooden_sword", Count: 1},
	{Name: "handgun", Count: 1},
	{Name: "western_armor", Count: 1},
	{Name: "work_helmet", Count: 1},
	{Name: "leather_boots", Count: 1},
	{Name: "healing_potion", Count: 5},
	{Name: "bread", Count: 3},
	{Name: "army_shooting_manual", Count: 1},
}

// PopulateMerchantStock は商人の品揃えを決める。アイテムを商人所有の LocationInStorage で持たせる。
// 商人生成時に一度だけ呼ぶ
func PopulateMerchantStock(world w.World, merchant ecs.Entity, _ *rand.Rand) error {
	for _, item := range merchantStockItems {
		if _, err := SpawnStorageItem(world, item.Name, item.Count, merchant); err != nil {
			return fmt.Errorf("failed to stock item %s: %w", item.Name, err)
		}
	}

	return nil
}
