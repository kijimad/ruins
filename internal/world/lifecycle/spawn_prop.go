package lifecycle

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// OpenDoor は扉を開く
func OpenDoor(world w.World, doorEntity ecs.Entity) error {
	if !world.Components.Door.Has(doorEntity) {
		return fmt.Errorf("entity is not a door")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, true)
}

// CloseDoor は扉を閉じる
func CloseDoor(world w.World, doorEntity ecs.Entity) error {
	if !world.Components.Door.Has(doorEntity) {
		return fmt.Errorf("entity is not a door")
	}

	doorComp := world.Components.Door.Get(doorEntity)
	return updateDoorState(world, doorEntity, doorComp.Orientation, false)
}

// updateDoorState は扉の向きと開閉状態に応じて、状態を更新する
func updateDoorState(world w.World, doorEntity ecs.Entity, orientation gc.DoorOrientation, isOpen bool) error {
	doorComp := world.Components.Door.Get(doorEntity)
	doorComp.Orientation = orientation
	doorComp.IsOpen = isOpen

	// スプライトキーと高さを更新する。
	// 開いた扉は平らな低い物として扱い、高さを下げる。高さが無いのでドロップシャドウを落とさず、
	// 背の高い物の後ろに沈む。閉じた扉は高さのある障壁に戻す。
	if world.Components.SpriteRender.Has(doorEntity) {
		spriteRender := world.Components.SpriteRender.Get(doorEntity)

		if door, ok := doorRaw(world, doorEntity); ok {
			spriteRender.SpriteKey = doorSpriteKey(door, orientation, isOpen)
		}
		if isOpen {
			spriteRender.Depth = gc.DepthNumRug
		} else {
			spriteRender.Depth = gc.DepthNumTaller
		}
	}

	// BlockPass / BlockView を更新
	if isOpen {
		if world.Components.BlockPass.Has(doorEntity) {
			world.Components.BlockPass.Remove(doorEntity)
		}
		if world.Components.BlockView.Has(doorEntity) {
			world.Components.BlockView.Remove(doorEntity)
		}
	} else {
		if !world.Components.BlockPass.Has(doorEntity) {
			world.Components.BlockPass.Add(doorEntity, &gc.BlockPass{})
		}
		if !world.Components.BlockView.Has(doorEntity) {
			world.Components.BlockView.Add(doorEntity, &gc.BlockView{})
		}
	}

	// 通行可否が変わったので周囲の囲われを再計算する。扉を開けると外気が入り、閉じると屋内に戻る
	if world.Components.GridElement.Has(doorEntity) {
		coord := world.Components.GridElement.Get(doorEntity).Coord
		RecalcShelterAround(world, coord.X, coord.Y)
	}

	return nil
}

// SpawnProp は置物を生成する
func SpawnProp(world w.World, propName string, x consts.Tile, y consts.Tile) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, propName)
	if err != nil {
		return gc.InvalidEntity, err
	}

	entitySpec.GridElement = &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	entitySpec.LocationOnField = &gc.LocationOnField{}

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// SpawnDungeonEntrance は raw の dungeon_entrance prop から遺跡入口を生成する。触れて Enter で definitionName の遺跡へ入る。
// 入口はオーバーワールドの地物なので StageBound{overworld} を直接持たせ、遺跡進入時に帯と共に退避されるようにする
func SpawnDungeonEntrance(world w.World, x consts.Tile, y consts.Tile, definitionName string) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, "dungeon_entrance")
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.GridElement = &gc.GridElement{Coord: consts.Coord[consts.Tile]{X: x, Y: y}}
	entitySpec.LocationOnField = &gc.LocationOnField{}
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionDungeonEnter}}
	entitySpec.DungeonEntrance = &gc.DungeonEntrance{DefinitionName: definitionName}
	entitySpec.StageBound = &gc.StageBound{Key: gc.NewOverworldStage()}

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// SpawnDoor は raw の door prop から扉を生成し、向きに合わせて閉じた姿にする
func SpawnDoor(world w.World, pos consts.Coord[consts.Tile], orientation gc.DoorOrientation) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, "door")
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.GridElement = &gc.GridElement{Coord: pos}
	entitySpec.LocationOnField = &gc.LocationOnField{}
	entitySpec.Door.Orientation = orientation
	prop, err := raw.GetProp(world.Resources.RawMaster, "door")
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.SpriteRender.SpriteKey = doorSpriteKey(*prop.Door, orientation, false)

	return world.Components.AddEntity(world.ECS, &entitySpec), nil
}

// doorRaw は扉の prop の行から扉の見た目の定義を引く。raw の行を持たない扉は ok=false
func doorRaw(world w.World, door ecs.Entity) (oapi.DoorRaw, bool) {
	if !world.Components.RawID.Has(door) {
		return oapi.DoorRaw{}, false
	}
	prop, err := raw.GetProp(world.Resources.RawMaster, world.Components.RawID.Get(door).ID)
	if err != nil || prop.Door == nil {
		return oapi.DoorRaw{}, false
	}
	return *prop.Door, true
}

// doorSpriteKey は扉の向きと開閉に応じたスプライトキーを返す
func doorSpriteKey(door oapi.DoorRaw, orientation gc.DoorOrientation, isOpen bool) string {
	switch {
	case orientation == gc.DoorOrientationHorizontal && isOpen:
		return door.HorizontalOpen
	case isOpen:
		return door.VerticalOpen
	case orientation == gc.DoorOrientationHorizontal:
		return door.HorizontalClosed
	default:
		return door.VerticalClosed
	}
}

// SpawnCube は raw の cube prop からプレイヤーに随伴する移動拠点キューブをオーバーワールドに生成し、帯へ明示束縛する。
// 陣営を持たないので、SoloAI の戦闘の状態遷移に入らず移動方針だけで動く。自分の拠点を殴れないよう、操作は随伴の操作だけにする
func SpawnCube(world w.World, pos consts.Coord[consts.Tile]) (ecs.Entity, error) {
	entitySpec, err := raw.NewPropSpec(world.Resources.RawMaster, "cube")
	if err != nil {
		return gc.InvalidEntity, err
	}
	entitySpec.GridElement = &gc.GridElement{Coord: pos}
	entitySpec.Cube = &gc.Cube{}
	entitySpec.FuelTank = &gc.FuelTank{Max: consts.CubeFuelCapacityBase, Current: consts.CubeFuelCapacityBase}
	entitySpec.SoloAI = &gc.SoloAI{
		CombatDefault: oapi.Ignore,
		CombatCurrent: oapi.Ignore,
		Movement:      oapi.Follow,
		SubState:      gc.AIStateWaiting,
	}
	entitySpec.TurnBased = &gc.TurnBased{}
	entitySpec.LocationOnField = &gc.LocationOnField{}
	entitySpec.StageBound = &gc.StageBound{Key: gc.NewOverworldStage()}
	entitySpec.WeightCapacity = &gc.WeightCapacity{Max: consts.Milligram(consts.CubeWeightCapacityKg * consts.MilligramPerKg)}
	entitySpec.Interactable = &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionOpenCubeMenu, gc.InteractionSwap}}
	cube := world.Components.AddEntity(world.ECS, &entitySpec)
	maxAP := query.CalculateMaxActionPoints(world, cube)
	world.Components.TurnBased.Get(cube).AP = gc.IntPool{Current: maxAP, Max: maxAP}
	return cube, nil
}
