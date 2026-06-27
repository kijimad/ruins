package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/engine/entities"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// SpawnProp は置物を生成する
func SpawnProp(world w.World, propName string, x consts.Tile, y consts.Tile) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, propName)
	if err != nil {
		return consts.InvalidEntity, err
	}

	entitySpec.GridElement = &gc.GridElement{X: x, Y: y}
	loc := gc.LocationTypeOnField
	entitySpec.LocationType = &loc

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	ents, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	return ents[len(ents)-1], nil
}

// SpawnDoor は扉を生成する
func SpawnDoor(world w.World, x consts.Tile, y consts.Tile, orientation gc.DoorOrientation) (ecs.Entity, error) {
	var spriteKey string
	if orientation == gc.DoorOrientationHorizontal {
		spriteKey = "door_horizontal_closed"
	} else {
		spriteKey = "door_vertical_closed"
	}

	loc := gc.LocationTypeOnField
	entitySpec := gc.EntitySpec{
		Name:        &gc.Name{Name: "扉"},
		Description: &gc.Description{Description: "開閉できる扉"},
		GridElement: &gc.GridElement{X: x, Y: y},
		SpriteRender: &gc.SpriteRender{
			SpriteSheetName: "field",
			SpriteKey:       spriteKey,
			Depth:           gc.DepthNumTaller,
		},
		BlockPass:    &gc.BlockPass{},
		BlockView:    &gc.BlockView{},
		LocationType: &loc,
		Door: &gc.Door{
			IsOpen:      false,
			Orientation: orientation,
		},
		Interactable: &gc.Interactable{Interactions: []gc.InteractionData{gc.DoorInteraction{}}},
	}

	componentList := entities.ComponentList[gc.EntitySpec]{}
	componentList.Entities = append(componentList.Entities, entitySpec)
	ents, err := entities.AddEntities(world, componentList)
	if err != nil {
		return consts.InvalidEntity, err
	}
	if len(ents) == 0 {
		return consts.InvalidEntity, fmt.Errorf("エンティティが生成されませんでした")
	}
	return ents[len(ents)-1], nil
}

// DeleteDoorLockTriggers はDoorLockInteractionを持つエンティティを全削除する
func DeleteDoorLockTriggers(world w.World) {
	var toDelete []ecs.Entity
	world.Manager.Join(world.Components.Interactable).Visit(ecs.Visit(func(triggerEntity ecs.Entity) {
		interactable := world.Components.Interactable.Get(triggerEntity).(*gc.Interactable)
		for _, interaction := range interactable.Interactions {
			if _, ok := interaction.(gc.DoorLockInteraction); ok {
				toDelete = append(toDelete, triggerEntity)
				return
			}
		}
	}))
	for _, entity := range toDelete {
		world.Manager.DeleteEntity(entity)
	}
}
