package mapplanner

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// アイテム配置用の定数
const (
	// アイテム配置関連
	baseItemCount     = 8 // アイテム配置の基本数
	randomItemCount   = 5 // アイテム配置のランダム追加数（0-4の範囲）
	itemIncreaseDepth = 5 // アイテム数増加の深度しきい値

	// 配置処理関連
	maxItemPlacementAttempts = 200 // アイテム配置処理の最大試行回数
)

// ItemSpec はアイテム配置仕様を表す
type ItemSpec struct {
	consts.Coord[int]
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

	depth := worldhelper.GetDungeon(i.world).Depth

	// アイテムの配置数（階層の深度に応じて調整）
	itemCount := baseItemCount + planData.RNG.IntN(randomItemCount)
	if depth > itemIncreaseDepth {
		itemCount++ // 深い階層ではアイテム数を増加
	}

	// アイテムを配置
	for j := 0; j < itemCount; j++ {
		entry, err := selectSpawnEntry(i.plannerType.ItemEntries, planData.RNG)
		if err != nil {
			return err
		}
		if entry.Name != "" {
			if err := i.addItem(planData, entry.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

// addItem は単一のアイテムをMetaPlanに追加する。
// 部屋がある場合は部屋内を優先し、見つからなければマップ全体にフォールバックする
func (i *ItemPlanner) addItem(planData *MetaPlan, itemName string) error {
	var selectors []positionSelector
	if room, _, ok := planData.selectRoom(); ok {
		selectors = append(selectors, inRoomSelector(room, maxRoomAttempts))
	}
	selectors = append(selectors, onMapSelector(maxItemPlacementAttempts))

	x, y, err := findPosition(planData, i.world, selectors...)
	if err != nil {
		return fmt.Errorf("%w: アイテム: %s", err, itemName)
	}

	planData.Items = append(planData.Items, ItemSpec{
		Coord: consts.Coord[int]{X: int(x), Y: int(y)},
		Name:  itemName,
	})
	return nil
}
