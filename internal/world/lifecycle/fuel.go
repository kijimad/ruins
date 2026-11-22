package lifecycle

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// errNoReplaceLocation は持ち物にも収納にも床にもない物を置き換えようとしたときに返す
var errNoReplaceLocation = errors.New("cannot replace item: entity has no backpack, storage or field location")

// ReplaceItem は item を、同じ置き場所の rawID の物1個に置き換え、新しい物を返す。持ち物・収納・床のどこでも置き場所を保つ
func ReplaceItem(world w.World, item ecs.Entity, rawID string) (ecs.Entity, error) {
	next, err := spawnItemBase(world, rawID)
	if err != nil {
		return gc.InvalidEntity, err
	}
	switch {
	case world.Components.LocationInBackpack.Has(item):
		err = MoveToBackpack(world, next, world.Components.LocationInBackpack.Get(item).Owner)
	case world.Components.LocationInStorage.Has(item):
		err = MoveToStorage(world, next, world.Components.LocationInStorage.Get(item).Owner)
	case world.Components.LocationOnField.Has(item):
		MoveToField(world, next, nil)
		if world.Components.GridElement.Has(item) {
			world.Components.GridElement.Add(next, &gc.GridElement{Coord: world.Components.GridElement.Get(item).Coord})
		}
		if world.Components.StageBound.Has(item) {
			world.Components.StageBound.Add(next, &gc.StageBound{Key: world.Components.StageBound.Get(item).Key})
		}
	default:
		err = errNoReplaceLocation
	}
	if err != nil {
		world.ECS.RemoveEntity(next)
		return gc.InvalidEntity, err
	}
	RemoveItem(world, item)
	return next, nil
}

// EmptyFuel は燃料の item を、同じ置き場所の空の物に置き換える。燃料でない物には何もせず偽を返す
func EmptyFuel(world w.World, item ecs.Entity) bool {
	refill, ok := query.RefillOf(world, item)
	if !ok {
		return false
	}
	if _, err := ReplaceItem(world, item, refill.EmptiesTo); err != nil {
		// 空の物の行があることは raw の検証が保証する
		panic(fmt.Sprintf("failed to replace fuel with its empty item: %v", err))
	}
	return true
}
