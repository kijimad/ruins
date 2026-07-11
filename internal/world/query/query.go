package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// Player はプレイヤーエンティティをVisitする。
// f はエンティティ生成などの構造変更を行うことがあるため、
// クエリを閉じてから呼び出す。反復中はワールドがロックされる
func Player(world w.World, f func(entity ecs.Entity)) {
	var players []ecs.Entity
	playerQuery := ecs.NewFilter2[gc.Player, gc.FactionAllyData](world.World).Query()
	for playerQuery.Next() {
		players = append(players, playerQuery.Entity())
	}
	for _, entity := range players {
		f(entity)
	}
}

// GetPlayerEntity はプレイヤーエンティティを返す
// プレイヤーが0個または2個以上の場合はエラーを返す
func GetPlayerEntity(world w.World) (ecs.Entity, error) {
	var entities []ecs.Entity
	playerQuery := ecs.NewFilter1[gc.Player](world.World).Query()
	for playerQuery.Next() {
		entity := playerQuery.Entity()
		entities = append(entities, entity)
	}

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
	if !world.Components.LocationOnField.Has(entity) {
		return false
	}
	if world.Components.Prop.Has(entity) {
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
	entitiesQuery := ecs.NewFilter1[gc.GridElement](world.World).Query()
	for entitiesQuery.Next() {
		entity := entitiesQuery.Entity()
		grid := world.Components.GridElement.Get(entity)
		if grid.X == x && grid.Y == y {
			entities = append(entities, entity)
		}
	}
	return entities
}
