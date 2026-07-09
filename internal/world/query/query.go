package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Player はプレイヤーエンティティをVisitする
func Player(world w.World, f func(entity ecs.Entity)) {
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
		return consts.InvalidEntity, fmt.Errorf("プレイヤーエンティティが存在しません")
	}
	if len(entities) > 1 {
		return consts.InvalidEntity, fmt.Errorf("プレイヤーエンティティが複数存在します: %d個", len(entities))
	}

	return entities[0], nil
}

// IsPickable はエンティティが拾得可能かを判定する。
// LocationOnField を持つ非Propエンティティが対象。
// Propは設置物なので拾えない。破壊や収納経由でアイテムを取得する
func IsPickable(entity ecs.Entity, world w.World) bool {
	if !entity.HasComponent(world.Components.LocationOnField) {
		return false
	}
	if entity.HasComponent(world.Components.Prop) {
		return false
	}
	return true
}

// IsInActivationRange はプレイヤーがトリガーの発動範囲内にいるかを判定する
func IsInActivationRange(playerGrid, triggerGrid *gc.GridElement, activationRange gc.ActivationRange) bool {
	switch activationRange {
	case gc.ActivationRangeSameTile:
		return playerGrid.X == triggerGrid.X && playerGrid.Y == triggerGrid.Y
	case gc.ActivationRangeAdjacent:
		return geometry.IsAdjacent(int(playerGrid.X), int(playerGrid.Y), int(triggerGrid.X), int(triggerGrid.Y))
	default:
		return false
	}
}

// GetEntitiesAt は指定座標にあるすべてのエンティティを返す
func GetEntitiesAt(world w.World, x, y consts.Tile) []ecs.Entity {
	var entities []ecs.Entity
	world.Manager.Join(
		world.Components.GridElement,
	).Visit(ecs.Visit(func(entity ecs.Entity) {
		grid := world.Components.GridElement.MustGet(entity)
		if grid.X == x && grid.Y == y {
			entities = append(entities, entity)
		}
	}))
	return entities
}
