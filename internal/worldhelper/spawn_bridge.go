package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/maptemplate"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SpawnBridge は出口操作エンティティを生成する
// 出口は階層間の遷移ポイントに配置され、次階層への移動相互作用を提供する
// 5の倍数の階層の場合、街広場へのワープInteractionを設定する
func SpawnBridge(
	world w.World,
	exitID maptemplate.ExitID,
	x gc.Tile,
	y gc.Tile,
	currentDepth int,
) (ecs.Entity, error) {
	exitName := fmt.Sprintf("出口(%s)", exitID)

	// 5の倍数の階層の場合、街広場へのワープInteractionを設定
	var interactionData gc.InteractionData
	if currentDepth%5 == 0 && currentDepth > 0 {
		interactionData = gc.PlazaWarpInteraction{}
	} else {
		// 通常の出口Interaction(次階層へ)
		interactionData = gc.BridgeInteraction{
			BridgeID: exitID,
		}
	}

	// EntitySpecを直接構築
	entitySpec := gc.EntitySpec{
		Name: &gc.Name{Name: exitName},
		Description: &gc.Description{
			Description: fmt.Sprintf("次の階層へと続く%s", exitName),
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
		return ecs.Entity(0), fmt.Errorf("出口エンティティの生成に失敗しました")
	}

	return entitiesSlice[0], nil
}
