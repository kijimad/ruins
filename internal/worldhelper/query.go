package worldhelper

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// QueryPlayer はプレイヤー
func QueryPlayer(world w.World, f func(entity ecs.Entity)) {
	world.Manager.Join(
		world.Components.Player,
		world.Components.FactionAlly,
	).Visit(ecs.Visit(f))
}

// GetPlayerEntity はプレイヤーエンティティを返す
// プレイヤーが0個または2個以上の場合はエラーを返す
func GetPlayerEntity(world w.World) (ecs.Entity, error) {
	var entities []ecs.Entity
	world.Manager.Join(world.Components.Player).Visit(ecs.Visit(func(entity ecs.Entity) {
		entities = append(entities, entity)
	}))

	if len(entities) == 0 {
		return 0, fmt.Errorf("プレイヤーエンティティが存在しません")
	}
	if len(entities) > 1 {
		return 0, fmt.Errorf("プレイヤーエンティティが複数存在します: %d個", len(entities))
	}

	return entities[0], nil
}

// IsPickable はエンティティが拾得可能かを判定する。
// LocationOnField を持つエンティティが対象。ただしPropはHPを持つ場合のみ拾える
func IsPickable(entity ecs.Entity, world w.World) bool {
	if !entity.HasComponent(world.Components.LocationOnField) {
		return false
	}
	if entity.HasComponent(world.Components.Prop) {
		return entity.HasComponent(world.Components.HP)
	}
	return true
}

// GetEntitiesAt は指定座標にあるすべてのエンティティを返す
func GetEntitiesAt(world w.World, x, y consts.Tile) []ecs.Entity {
	var entities []ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.Get(entity).(*gc.GridElement)
		if grid.X == x && grid.Y == y {
			entities = append(entities, entity)
		}
	}))
	return entities
}
