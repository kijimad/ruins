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
func SpawnBridge(
	world w.World,
	exitID maptemplate.ExitID,
	x gc.Tile,
	y gc.Tile,
) (ecs.Entity, error) {
	exitName := fmt.Sprintf("出口(%s)", exitID)

	interactionData := gc.BridgeInteraction{
		BridgeID: exitID,
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

// SpawnBridgeHint は橋のヒント表示エンティティを生成する
// 指定位置に到達すると、関連する橋の先の階層情報をgamelogに表示する
func SpawnBridgeHint(
	world w.World,
	exitID maptemplate.ExitID,
	x gc.Tile,
	y gc.Tile,
) (ecs.Entity, error) {
	entitySpec := gc.EntitySpec{
		GridElement: &gc.GridElement{X: x, Y: y},
		Interactable: &gc.Interactable{
			Data: gc.BridgeHintInteraction{
				ExitID: exitID,
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
		return ecs.Entity(0), fmt.Errorf("橋ヒントエンティティの生成に失敗しました")
	}

	entity := entitiesSlice[0]

	return entity, nil
}
