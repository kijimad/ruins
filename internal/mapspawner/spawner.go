package mapspawner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/resources"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
)

// Spawn はMetaPlanからレベルを生成する
// タイル、NPC、Props、ワープポータル情報から効率的にエンティティを生成する
func Spawn(world w.World, metaPlan *mapplanner.MetaPlan) (resources.Level, error) {
	level := resources.Level{
		TileWidth:  metaPlan.Level.TileWidth,
		TileHeight: metaPlan.Level.TileHeight,
	}

	// タイルからエンティティを直接生成
	for _i, tile := range metaPlan.Tiles {
		i := resources.TileIdx(_i)
		x, y := metaPlan.Level.XYTileCoord(i)
		tileX, tileY := gc.Tile(x), gc.Tile(y)

		var err error

		if tile.Walkable {
			// すべての歩行可能タイルは16オートタイルシステムを使用
			switch tile.Name {
			case "Dirt":
				index := int(metaPlan.CalculateAutoTileIndex(i, "Dirt"))
				_, err = worldhelper.SpawnTile(world, "Dirt", tileX, tileY, &index)
			case "Floor":
				index := int(metaPlan.CalculateAutoTileIndex(i, "Floor"))
				_, err = worldhelper.SpawnTile(world, "Floor", tileX, tileY, &index)
			default:
				// 未知のタイル名はエラーとして処理
				return resources.Level{}, fmt.Errorf("未対応の歩行可能タイル名: %s (%d, %d)", tile.Name, int(x), int(y))
			}
		} else {
			// 隣接に床がある場合のみ壁エンティティを生成
			if metaPlan.AdjacentAnyFloor(i) {
				// 壁タイルも16タイルオートタイルを使用
				index := int(metaPlan.CalculateAutoTileIndex(i, "Wall"))
				_, err = worldhelper.SpawnTile(world, "Wall", tileX, tileY, &index)
			}
		}

		if err != nil {
			return resources.Level{}, fmt.Errorf("タイルエンティティ生成エラー (%d, %d): %w", int(x), int(y), err)
		}
	}

	// ワープポータルを生成
	for _, portal := range metaPlan.WarpPortals {
		tileX, tileY := gc.Tile(portal.X), gc.Tile(portal.Y)

		var propName string
		switch portal.Type {
		case mapplanner.WarpPortalNext:
			propName = "warp_next"
		case mapplanner.WarpPortalEscape:
			propName = "warp_escape"
		}

		_, err := worldhelper.SpawnProp(world, propName, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("ワープポータル生成エラー (%d, %d): %w", portal.X, portal.Y, err)
		}
	}

	// NPCを生成
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	for _, npc := range metaPlan.NPCs {
		// NPCが中立かどうかを判断
		memberIdx, ok := rawMaster.MemberIndex[npc.NPCType]
		if !ok {
			return resources.Level{}, fmt.Errorf("NPC '%s' が見つかりません", npc.NPCType)
		}
		member := rawMaster.Raws.Members[memberIdx]

		// 中立NPCの場合
		if member.FactionType == gc.FactionNeutral.String() {
			_, err := worldhelper.SpawnNeutralNPC(world, npc.X, npc.Y, npc.NPCType)
			if err != nil {
				return resources.Level{}, fmt.Errorf("中立NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		} else {
			// 敵NPCの場合
			_, err := worldhelper.SpawnEnemy(world, npc.X, npc.Y, npc.NPCType)
			if err != nil {
				return resources.Level{}, fmt.Errorf("敵NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		}
	}

	// アイテムを生成
	for _, item := range metaPlan.Items {
		tileX, tileY := gc.Tile(item.X), gc.Tile(item.Y)
		_, err := worldhelper.SpawnFieldItem(world, item.ItemName, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("アイテム生成エラー (%d, %d): %w", item.X, item.Y, err)
		}
	}

	// Propsを生成
	for _, prop := range metaPlan.Props {
		tileX, tileY := gc.Tile(prop.X), gc.Tile(prop.Y)
		_, err := worldhelper.SpawnProp(world, prop.PropKey, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("props生成エラー (%d, %d): %w", prop.X, prop.Y, err)
		}
	}

	// ドアを生成
	for _, door := range metaPlan.Doors {
		tileX, tileY := gc.Tile(door.X), gc.Tile(door.Y)
		_, err := worldhelper.SpawnDoor(world, tileX, tileY, door.Orientation)
		if err != nil {
			return resources.Level{}, fmt.Errorf("ドア生成エラー (%d, %d): %w", door.X, door.Y, err)
		}
	}

	return level, nil
}
