package mapplanner

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
)

// アイテム配置用の定数
const (
	// アイテム配置関連
	baseItemCount     = 15 // アイテム配置の基本数
	randomItemCount   = 9  // アイテム配置のランダム追加数（0-8の範囲）
	itemIncreaseDepth = 5  // アイテム数増加の深度しきい値

	// 配置処理関連
	maxItemPlacementAttempts = 200 // アイテム配置処理の最大試行回数
)

// ItemSpec はアイテム配置仕様を表す
type ItemSpec struct {
	Coord
	Name string // アイテム名
}

// ItemPlanner はアイテム配置を担当するプランナー
type ItemPlanner struct {
	world       w.World
	plannerType PlannerType
}

// NewItemPlanner はアイテムプランナーを作成する
func NewItemPlanner(world w.World, plannerType PlannerType) *ItemPlanner {
	return &ItemPlanner{
		world:       world,
		plannerType: plannerType,
	}
}

// PlanMeta はアイテム配置情報をMetaPlanに追加する
func (i *ItemPlanner) PlanMeta(planData *MetaPlan) error {
	if len(i.plannerType.ItemEntries) == 0 {
		return nil // エントリがない場合は何もしない
	}

	// Itemsフィールドが存在しない場合は初期化
	if planData.Items == nil {
		planData.Items = []ItemSpec{}
	}

	depth := i.world.Resources.Dungeon.Depth

	// アイテムの配置数（階層の深度に応じて調整）
	itemCount := baseItemCount + planData.RNG.IntN(randomItemCount)
	if depth > itemIncreaseDepth {
		itemCount++ // 深い階層ではアイテム数を増加
	}

	// アイテムを配置
	for j := 0; j < itemCount; j++ {
		itemName, err := selectByWeight(i.plannerType.ItemEntries, planData.RNG)
		if err != nil {
			return err
		}
		if itemName != "" {
			if err := i.addItem(planData, itemName); err != nil {
				return err
			}
		}
	}

	return nil
}

// selectByWeight はエントリから重み付き抽選で名前を選択する
func selectByWeight(entries []SpawnEntry, rng *rand.Rand) (string, error) {
	items := make([]raw.WeightedItem, len(entries))
	for i, entry := range entries {
		items[i] = raw.WeightedItem{Value: entry.Name, Weight: entry.Weight}
	}
	return raw.SelectByWeight(items, rng)
}

// addItem は単一のアイテムをMetaPlanに追加する
func (i *ItemPlanner) addItem(planData *MetaPlan, itemName string) error {
	failCount := 0

	for {
		if failCount > maxItemPlacementAttempts {
			return fmt.Errorf("アイテム配置の試行回数が上限に達しました。アイテム: %s", itemName)
		}

		// ランダムな位置を選択
		x := gc.Tile(planData.RNG.IntN(int(planData.Level.TileWidth)))
		y := gc.Tile(planData.RNG.IntN(int(planData.Level.TileHeight)))

		// スポーン可能な位置かチェック
		if !i.isValidItemPosition(planData, x, y) {
			failCount++
			continue
		}

		// MetaPlanにアイテムを追加
		planData.Items = append(planData.Items, ItemSpec{
			Coord: Coord{X: int(x), Y: int(y)},
			Name:  itemName,
		})

		return nil
	}
}

// isValidItemPosition はアイテム配置に適した位置かチェックする
func (i *ItemPlanner) isValidItemPosition(planData *MetaPlan, x, y gc.Tile) bool {
	tileIdx := planData.Level.XYTileIndex(x, y)
	if int(tileIdx) >= len(planData.Tiles) {
		return false
	}

	tile := planData.Tiles[tileIdx]
	// 歩行可能なタイルに配置可能
	return !tile.BlockPass
}
