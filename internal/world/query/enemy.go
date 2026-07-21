package query

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	"github.com/kijimaD/ruins/internal/geometry"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// GetVisibleEnemies は視界内の敵エンティティをすべて取得して返す
func GetVisibleEnemies(world w.World) ([]ecs.Entity, error) {
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}

	if !world.Components.GridElement.Has(playerEntity) {
		return nil, fmt.Errorf("プレイヤーがGridElementを持っていません")
	}

	playerGrid := world.Components.GridElement.Get(playerEntity)

	var enemies []ecs.Entity

	// 座標が重なる退避中ステージの敵を可視判定に混ぜない。現ステージのみ対象にする
	enemiesQuery := ActiveFilter2[gc.GridElement, gc.FactionEnemy](world).Query()
	for enemiesQuery.Next() {
		entity := enemiesQuery.Entity()
		gridElement := world.Components.GridElement.Get(entity)

		if !IsInVision(world, playerGrid.Coord, gridElement.Coord) {
			continue
		}

		enemies = append(enemies, entity)
	}

	return enemies, nil
}

// IsInVision はプレイヤーから指定座標が現在見えるかをチェックする。
// リアルタイムの可視性データを使用し、暗闇のタイルは見えないと判定する
func IsInVision(world w.World, player, target consts.Coord[consts.Tile]) bool {
	distanceInPixels := geometry.Distance(float64(player.X), float64(player.Y), float64(target.X), float64(target.Y)) * float64(consts.TileSize)
	visionRadius := float64(consts.VisionRadiusTiles) * float64(consts.TileSize)

	if distanceInPixels > visionRadius {
		return false
	}

	vs := GetVisionState(world)
	if vs.VisibleTiles == nil {
		return false
	}

	return vs.VisibleTiles[gc.GridElement{Coord: target}]
}

// GetVisibleItems は視界内のアイテムエンティティをすべて取得して返す
func GetVisibleItems(world w.World) ([]ecs.Entity, error) {
	playerEntity, err := GetPlayerEntity(world)
	if err != nil {
		return nil, err
	}

	if !world.Components.GridElement.Has(playerEntity) {
		return nil, fmt.Errorf("プレイヤーがGridElementを持っていません")
	}

	playerGrid := world.Components.GridElement.Get(playerEntity)

	var items []ecs.Entity

	itemsQuery := ActiveFilter2[gc.GridElement, gc.LocationOnField](world).Query()
	for itemsQuery.Next() {
		entity := itemsQuery.Entity()
		gridElement := world.Components.GridElement.Get(entity)

		if !IsInVision(world, playerGrid.Coord, gridElement.Coord) {
			continue
		}

		items = append(items, entity)
	}

	return items, nil
}

// GetEntityName はエンティティの名前を取得する
func GetEntityName(entity ecs.Entity, world w.World) string {
	if !world.ECS.Alive(entity) || !world.Components.Name.Has(entity) {
		return "Unknown"
	}
	return world.Components.Name.Get(entity).Name
}

// AppendNameWithColor はエンティティの種類に応じて色付きで名前を追加する
func AppendNameWithColor(logger *gamelog.Logger, entity ecs.Entity, name string, world w.World) {
	switch {
	case world.Components.Player.Has(entity):
		logger.PlayerName(name)
	case world.Components.SoloAI.Has(entity) || world.Components.SquadAI.Has(entity):
		logger.NPCName(name)
	default:
		logger.Append(name)
	}
}
