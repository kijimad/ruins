package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/engine/entities"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// CalculateBridgeSeed は橋固有のシード値を計算する
func CalculateBridgeSeed(baseDepth int, gameSeed uint64, bridgeID string) uint64 {
	// 橋ごとのオフセット
	offsets := map[string]uint64{
		"A": 1000,
		"B": 2000,
		"C": 3000,
		"D": 4000,
	}

	baseSeed := uint64(baseDepth) + gameSeed
	offset, ok := offsets[bridgeID]
	if !ok {
		// 未知のbridgeIDの場合はデフォルトオフセット
		offset = 9999
	}

	return baseSeed + offset
}

// SpawnBridge は橋エンティティを生成する
// 橋エンティティは橋の端に配置され、橋を渡る相互作用を提供する
func SpawnBridge(
	world w.World,
	bridgeID string,
	x gc.Tile,
	y gc.Tile,
	currentDepth int,
	gameSeed uint64,
) (ecs.Entity, error) {
	// NextFloorSeed を計算
	nextFloorSeed := CalculateBridgeSeed(currentDepth, gameSeed, bridgeID)

	// 橋名を構築（例: "橋A"）
	bridgeName := fmt.Sprintf("橋%s", bridgeID)

	// EntitySpecを直接構築
	entitySpec := gc.EntitySpec{
		Name: &gc.Name{Name: bridgeName},
		Description: &gc.Description{
			Description: fmt.Sprintf("次の階層へと続く%s", bridgeName),
		},
		GridElement: &gc.GridElement{X: x, Y: y},
		Interactable: &gc.Interactable{
			Data: gc.BridgeInteraction{
				BridgeID:      bridgeID,
				NextFloorSeed: nextFloorSeed,
			},
		},
	}

	// エンティティを追加
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)

	entitiesSlice, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(entitiesSlice) == 0 {
		return ecs.Entity(0), fmt.Errorf("橋エンティティの生成に失敗しました")
	}

	return entitiesSlice[0], nil
}
