package worldhelper

import (
	"fmt"

	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/raw"
	ecs "github.com/x-hgg-x/goecs/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
)

// SpawnProp は置物を生成する
func SpawnProp(world w.World, propName string, x gc.Tile, y gc.Tile) (ecs.Entity, error) {
	// RawMasterから置物の設定を生成
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	entitySpec, err := rawMaster.NewPropSpec(propName)
	if err != nil {
		return ecs.Entity(0), err
	}

	// 位置情報を設定
	entitySpec.GridElement = &gc.GridElement{X: x, Y: y}

	// エンティティを生成
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	entities, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	return entities[len(entities)-1], nil
}

// SpawnDoor はドアを生成する
func SpawnDoor(world w.World, x gc.Tile, y gc.Tile, orientation gc.DoorOrientation) (ecs.Entity, error) {
	// スプライトキーを決定（閉じたドア）
	var spriteKey string
	if orientation == gc.DoorOrientationHorizontal {
		spriteKey = "door_horizontal_closed"
	} else {
		spriteKey = "door_vertical_closed"
	}

	// EntitySpecを構築
	entitySpec := gc.EntitySpec{
		Name:        &gc.Name{Name: "ドア"},
		Description: &gc.Description{Description: "開閉できるドア"},
		GridElement: &gc.GridElement{X: x, Y: y},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "field",
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumTaller,
		},
		BlockPass: &gc.BlockPass{}, // 閉じているので通行不可
		BlockView: &gc.BlockView{}, // 閉じているので視線を遮る
		Door: &gc.Door{
			IsOpen:      false,
			Orientation: orientation,
		},
		Interactable: &gc.Interactable{Data: gc.DoorInteraction{}},
	}

	// エンティティを生成
	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	ents, err := entities.AddEntities(world, componentList)
	if err != nil {
		return ecs.Entity(0), err
	}
	if len(ents) == 0 {
		return ecs.Entity(0), fmt.Errorf("エンティティが生成されませんでした")
	}
	return ents[len(ents)-1], nil
}
