package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/maptemplate"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SpawnBridge は橋の作用エンティティを生成する
// 橋エンティティは橋の端に配置され、橋を渡る相互作用を提供する
// 5の倍数の階層かつ出口橋の場合、街広場へのワープInteractionを設定する
func SpawnBridge(
	world w.World,
	bridgeID maptemplate.BridgeID,
	x gc.Tile,
	y gc.Tile,
	currentDepth int,
) (ecs.Entity, error) {
	// 橋名を構築（例: "出口橋"）
	bridgeName := fmt.Sprintf("橋%s", bridgeID)

	// 5の倍数の階層かつ出口橋の場合、街広場へのワープInteractionを設定
	var interactionData gc.InteractionData
	if currentDepth%5 == 0 && currentDepth > 0 && bridgeID.IsExit() {
		interactionData = gc.PlazaWarpInteraction{}
	} else {
		// 通常の橋Interaction（次階層へ）
		interactionData = gc.BridgeInteraction{
			BridgeID: bridgeID,
		}
	}

	// EntitySpecを直接構築
	entitySpec := gc.EntitySpec{
		Name: &gc.Name{Name: bridgeName},
		Description: &gc.Description{
			Description: fmt.Sprintf("次の階層へと続く%s", bridgeName),
		},
		GridElement: &gc.GridElement{X: x, Y: y},
		Interactable: &gc.Interactable{
			Data: interactionData,
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
