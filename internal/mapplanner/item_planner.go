package mapplanner

import (
	"fmt"
	"log"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/raw"
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
	itemClusterRadius        = 3   // アイテムクラスタの半径（タイル数）
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
	if len(i.plannerType.ItemSources) == 0 {
		return nil
	}

	if planData.Items == nil {
		planData.Items = []ItemSpec{}
	}

	depth := worldhelper.GetDungeon(i.world).Depth

	total := baseItemCount + planData.RNG.IntN(randomItemCount)
	if depth > itemIncreaseDepth {
		total++
	}

	placed := 0
	failCount := 0
	for placed < total && failCount <= maxItemPlacementAttempts {
		// ソースを重みで選択
		source, err := raw.SelectByWeightFunc(
			i.plannerType.ItemSources,
			func(s ItemSource) float64 { return s.Weight },
			func(s ItemSource) ItemSource { return s },
			planData.RNG,
		)
		if err != nil {
			return err
		}

		// ソースからアイテムを解決
		items, err := resolveItemSource(source, planData)
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
			Coord: consts.Coord[int]{X: int(anchorX), Y: int(anchorY)},
			Name:  first.Name,
		})
		placed++
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
				Coord: consts.Coord[int]{X: int(nx), Y: int(ny)},
				Name:  items[idx].Name,
			})
			placed++
			failCount = 0
		}
	}

	if failCount > maxItemPlacementAttempts {
		log.Printf("ItemPlanner: アイテム配置の試行回数が上限に達しました。配置数: %d/%d", placed, total)
	}
	return nil
}

// resolveItemSource はソースからアイテムリストを解決する
func resolveItemSource(source ItemSource, planData *MetaPlan) ([]resolvedItem, error) {
	switch source.Subtype {
	case ItemGroupCollection:
		return resolveCollection(source.Entries, planData), nil
	case ItemGroupDistribution:
		return resolveDistribution(source.Entries, planData), nil
	default:
		return nil, fmt.Errorf("未知のItemGroupSubtype: %s", source.Subtype)
	}
}

type resolvedItem struct {
	Name string
}

// resolveDistribution は重みで1つ選択し、PackSize分のアイテムを返す
func resolveDistribution(entries []SpawnEntry, planData *MetaPlan) []resolvedItem {
	if len(entries) == 0 {
		return nil
	}
	entry, err := selectSpawnEntry(entries, planData.RNG)
	if err != nil || entry.Name == "" {
		return nil
	}
	packSize := entry.PackSize(planData.RNG)
	result := make([]resolvedItem, packSize)
	for i := range result {
		result[i] = resolvedItem{Name: entry.Name}
	}
	return result
}

// resolveCollection は各エントリを確率判定（weight を 0-100 の確率として扱う）し、
// 当選したもののPackSize分を返す
func resolveCollection(entries []SpawnEntry, planData *MetaPlan) []resolvedItem {
	var result []resolvedItem
	for _, entry := range entries {
		prob := entry.Weight
		if prob <= 0 {
			continue
		}
		roll := planData.RNG.Float64() * 100
		if roll < prob {
			packSize := entry.PackSize(planData.RNG)
			for j := 0; j < packSize; j++ {
				result = append(result, resolvedItem{Name: entry.Name})
			}
		}
	}
	return result
}
