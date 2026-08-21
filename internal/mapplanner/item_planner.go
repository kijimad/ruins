package mapplanner

import (
	"log"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
)

// アイテム配置用の定数
const (
	// アイテム配置関連
	baseItemCount     = 8 // アイテム配置の基本数
	randomItemCount   = 5 // アイテム配置のランダム追加数（0-4の範囲）
	itemIncreaseDepth = 5 // アイテム数増加の深度しきい値

	// 配置処理関連
	maxItemPlacementAttempts = 200 // アイテム配置処理の最大試行回数
	itemClusterRadius        = 3   // アイテムクラスタの半径（タイル数）
)

// ItemSpec はアイテム配置仕様を表す
type ItemSpec struct {
	consts.Coord[consts.Tile]
	Name  string // アイテム名
	Count int    // 個数
}

// itemGroupRef はアイテムテーブルの1エントリ。深度フィルタ後に残った参照先グループの id とテーブル重み。
// グループ中身の抽選は raw.SelectFromItemGroup が draw 時に行う。
type itemGroupRef struct {
	GroupID string
	Weight  float64
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
	sources, err := resolveItemSources(planData.RawMaster, i.plannerType.ItemTableName, i.plannerType.Depth)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return nil
	}

	if planData.Items == nil {
		planData.Items = []ItemSpec{}
	}

	total := baseItemCount + planData.RNG.IntN(randomItemCount)
	if i.plannerType.Depth > itemIncreaseDepth {
		total++
	}

	placed := 0
	failCount := 0
	for placed < total && failCount <= maxItemPlacementAttempts {
		// 参照先グループをテーブル重みで選ぶ
		source, err := raw.SelectByWeightFunc(
			sources,
			func(s itemGroupRef) float64 { return s.Weight },
			func(s itemGroupRef) itemGroupRef { return s },
			planData.RNG,
		)
		if err != nil {
			return err
		}

		// グループから1山を抽選する。distribution/collection と pack の解釈は raw が担う
		items, err := raw.SelectFromItemGroup(*planData.RawMaster, source.GroupID, planData.RNG)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			failCount++
			continue
		}

		// 部屋を選んでアンカーを配置
		room, _, roomOK := planData.selectRoom()
		var selectors []positionSelector
		if roomOK {
			selectors = append(selectors, inRoomSelector(room, maxRoomAttempts))
		}
		selectors = append(selectors, onMapSelector(maxItemPlacementAttempts))

		anchorX, anchorY, posErr := findPosition(planData, i.world, selectors...)
		if posErr != nil {
			failCount++
			continue
		}

		// 最初のアイテムをアンカーに配置
		first := items[0]
		planData.Items = append(planData.Items, ItemSpec{
			X: anchorX, Y: anchorY,
			Name:  first.Name,
			Count: first.Count,
		})
		placed += first.Count
		failCount = 0

		// 残りのアイテムをアンカー周辺に配置
		for idx := 1; idx < len(items) && placed < total; idx++ {
			var nearSelectors []positionSelector
			if roomOK {
				nearSelectors = append(nearSelectors, nearSelector(anchorX, anchorY, itemClusterRadius, room, maxRoomAttempts))
			}
			nearSelectors = append(nearSelectors, onMapSelector(maxItemPlacementAttempts))

			nx, ny, nearErr := findPosition(planData, i.world, nearSelectors...)
			if nearErr != nil {
				failCount++
				break
			}
			planData.Items = append(planData.Items, ItemSpec{
				X: nx, Y: ny,
				Name:  items[idx].Name,
				Count: items[idx].Count,
			})
			placed += items[idx].Count
			failCount = 0
		}
	}

	if failCount > maxItemPlacementAttempts {
		log.Printf("ItemPlanner: reached max attempts placing items. placed: %d/%d", placed, total)
	}
	return nil
}
