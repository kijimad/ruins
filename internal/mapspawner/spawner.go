package mapspawner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/worldhelper"
	ecs "github.com/x-hgg-x/goecs/v2"
)

// Spawn はMetaPlanからレベルを生成する
// タイル、NPC、Props、ワープポータル情報から効率的にエンティティを生成する
func Spawn(world w.World, metaPlan *mapplanner.MetaPlan) (gc.Level, error) {
	level := gc.Level{
		TileWidth:  metaPlan.Level.TileWidth,
		TileHeight: metaPlan.Level.TileHeight,
	}

	if err := spawnTiles(world, metaPlan); err != nil {
		return gc.Level{}, err
	}
	if err := spawnNPCs(world, metaPlan); err != nil {
		return gc.Level{}, err
	}
	if err := spawnItems(world, metaPlan); err != nil {
		return gc.Level{}, err
	}
	if err := spawnProps(world, metaPlan); err != nil {
		return gc.Level{}, err
	}
	if err := spawnDoors(world, metaPlan); err != nil {
		return gc.Level{}, err
	}
	if err := spawnPortals(world, metaPlan); err != nil {
		return gc.Level{}, err
	}

	return level, nil
}

// spawnTiles はタイルからエンティティを生成する
func spawnTiles(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _i, tile := range metaPlan.Tiles {
		i := gc.TileIdx(_i)
		x, y := metaPlan.Level.XYTileCoord(i)
		tileX, tileY := consts.Tile(x), consts.Tile(y)

		tileEntity, err := spawnTile(world, metaPlan, tile, i, tileX, tileY)
		if err != nil {
			return fmt.Errorf("タイルエンティティ生成エラー (%d, %d): %w", int(x), int(y), err)
		}

		// TileRaw の環境情報を TileTemperature に設定する
		if tileTemp, ok := world.Components.TileTemperature.Get(tileEntity).(*gc.TileTemperature); ok && tileTemp != nil {
			tileTemp.Shelter = gc.ShelterType(tile.Shelter)
			tileTemp.Water = gc.WaterType(tile.Water)
			tileTemp.Foliage = gc.FoliageType(tile.Foliage)
		}
	}
	return nil
}

// spawnTile は1タイルを生成する
func spawnTile(world w.World, metaPlan *mapplanner.MetaPlan, tile oapi.Tile, i gc.TileIdx, tileX, tileY consts.Tile) (ecs.Entity, error) {
	// TODO: タイル名直判定だと忘れやすいので直したい
	if !tile.BlockPass {
		switch tile.Name {
		case "dirt":
			index := int(metaPlan.CalculateAutoTileIndex(i, "dirt"))
			return worldhelper.SpawnTile(world, "dirt", tileX, tileY, &index)
		case "floor":
			index := int(metaPlan.CalculateAutoTileIndex(i, "floor"))
			return worldhelper.SpawnTile(world, "floor", tileX, tileY, &index)
		case "bridge_a", "bridge_b", "bridge_c", "bridge_d":
			index := int(metaPlan.CalculateAutoTileIndex(i, tile.Name))
			return worldhelper.SpawnTile(world, tile.Name, tileX, tileY, &index)
		default:
			return consts.InvalidEntity, fmt.Errorf("未対応の歩行可能タイル名: %s (%d, %d)", tile.Name, int(tileX), int(tileY))
		}
	}

	switch tile.Name {
	case "wall":
		index := int(metaPlan.CalculateAutoTileIndex(i, "wall"))
		return worldhelper.SpawnTile(world, "dwall", tileX, tileY, &index)
	case "void":
		return worldhelper.SpawnTile(world, "void", tileX, tileY, nil)
	default:
		return consts.InvalidEntity, fmt.Errorf("未対応の通行不可タイル名: %s (%d, %d)", tile.Name, int(tileX), int(tileY))
	}
}

// spawnNPCs はNPCを生成する
func spawnNPCs(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _, npc := range metaPlan.NPCs {
		member, err := raw.FindMember(world.Resources.RawMaster, npc.Name)
		if err != nil {
			return fmt.Errorf("NPC '%s' が見つかりません", npc.Name)
		}

		if member.FactionType != nil && string(*member.FactionType) == gc.FactionNeutral.String() {
			_, err := worldhelper.SpawnNeutralNPC(world, npc.X, npc.Y, npc.Name)
			if err != nil {
				return fmt.Errorf("中立NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		} else {
			var opts []worldhelper.SpawnEnemyOption
			if member.IsBoss {
				opts = append(opts, worldhelper.WithBoss())
			}
			_, err := worldhelper.SpawnEnemy(world, npc.X, npc.Y, npc.Name, opts...)
			if err != nil {
				return fmt.Errorf("敵NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		}
	}
	return nil
}

// spawnItems はアイテムを生成する
func spawnItems(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _, item := range metaPlan.Items {
		tileX, tileY := consts.Tile(item.X), consts.Tile(item.Y)
		if item.Count <= 0 {
			return fmt.Errorf("アイテムの個数が不正です (%d, %d): count=%d", item.X, item.Y, item.Count)
		}
		_, err := worldhelper.SpawnFieldItem(world, item.Name, tileX, tileY, item.Count)
		if err != nil {
			return fmt.Errorf("アイテム生成エラー (%d, %d): %w", item.X, item.Y, err)
		}
	}
	return nil
}

// spawnProps はPropsを生成する
func spawnProps(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _, prop := range metaPlan.Props {
		tileX, tileY := consts.Tile(prop.X), consts.Tile(prop.Y)

		propRaw, err := raw.GetProp(*metaPlan.RawMaster, prop.Name)
		if err != nil {
			return fmt.Errorf("props取得エラー (%s): %w", prop.Name, err)
		}

		propEntity, err := worldhelper.SpawnProp(world, prop.Name, tileX, tileY)
		if err != nil {
			return fmt.Errorf("props生成エラー (%d, %d): %w", prop.X, prop.Y, err)
		}

		// Door componentがあれば向きを設定して閉じた状態で初期化
		if propRaw.Door != nil {
			orientation := detectDoorOrientation(metaPlan, prop.X, prop.Y)
			doorComp := world.Components.Door.Get(propEntity).(*gc.Door)
			doorComp.Orientation = orientation
			if err := worldhelper.CloseDoor(world, propEntity); err != nil {
				return fmt.Errorf("扉初期化エラー (%d, %d): %w", prop.X, prop.Y, err)
			}
		}

		// Storage propにルートアイテムを格納する
		if propRaw.Storage != nil && propRaw.Storage.LootTableName != nil && *propRaw.Storage.LootTableName != "" {
			if err := populateStorageLoot(world, metaPlan, propEntity, propRaw); err != nil {
				return fmt.Errorf("収納アイテム生成エラー (%d, %d): %w", prop.X, prop.Y, err)
			}
		}
	}
	return nil
}

// spawnDoors はドアを生成する
func spawnDoors(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _, door := range metaPlan.Doors {
		tileX, tileY := consts.Tile(door.X), consts.Tile(door.Y)
		orientation := detectDoorOrientation(metaPlan, door.X, door.Y)
		_, err := worldhelper.SpawnDoor(world, tileX, tileY, orientation)
		if err != nil {
			return fmt.Errorf("ドア生成エラー (%d, %d): %w", door.X, door.Y, err)
		}
	}
	return nil
}

// spawnPortals はポータルを生成する
func spawnPortals(world w.World, metaPlan *mapplanner.MetaPlan) error {
	for _, portal := range metaPlan.NextPortals {
		tileX, tileY := consts.Tile(portal.X), consts.Tile(portal.Y)
		_, err := worldhelper.SpawnProp(world, "warp_next", tileX, tileY)
		if err != nil {
			return fmt.Errorf("NextPortal生成エラー (%d, %d): %w", portal.X, portal.Y, err)
		}
	}

	for _, portal := range metaPlan.EscapePortals {
		tileX, tileY := consts.Tile(portal.X), consts.Tile(portal.Y)
		_, err := worldhelper.SpawnProp(world, "warp_escape", tileX, tileY)
		if err != nil {
			return fmt.Errorf("EscapePortal生成エラー (%d, %d): %w", portal.X, portal.Y, err)
		}
	}
	return nil
}

// detectDoorOrientation は隣接タイルから扉の向きを判定する
// 左右が壁の場合は縦向き、上下が壁の場合は横向き
func detectDoorOrientation(metaPlan *mapplanner.MetaPlan, x, y int) gc.DoorOrientation {
	width := int(metaPlan.Level.TileWidth)
	height := int(metaPlan.Level.TileHeight)

	// 範囲外チェック
	if x <= 0 || x >= width-1 || y <= 0 || y >= height-1 {
		// デフォルトは横向き
		return gc.DoorOrientationHorizontal
	}

	// 隣接タイルを取得
	leftIdx := y*width + (x - 1)
	rightIdx := y*width + (x + 1)
	topIdx := (y-1)*width + x
	bottomIdx := (y+1)*width + x

	leftTile := metaPlan.Tiles[leftIdx]
	rightTile := metaPlan.Tiles[rightIdx]
	topTile := metaPlan.Tiles[topIdx]
	bottomTile := metaPlan.Tiles[bottomIdx]

	// 左右が壁（通行不可）の場合は縦向き
	if leftTile.BlockPass && rightTile.BlockPass {
		return gc.DoorOrientationVertical
	}

	// 上下が壁（通行不可）の場合は横向き
	if topTile.BlockPass && bottomTile.BlockPass {
		return gc.DoorOrientationHorizontal
	}

	// どちらでもない場合はデフォルト（横向き）
	return gc.DoorOrientationHorizontal
}

// populateStorageLoot は収納propにルートテーブルからアイテムを格納する
func populateStorageLoot(world w.World, metaPlan *mapplanner.MetaPlan, storageEntity ecs.Entity, propRaw oapi.Prop) error {
	tableName := *propRaw.Storage.LootTableName
	itemTable, err := raw.GetItemTable(*metaPlan.RawMaster, tableName)
	if err != nil {
		return fmt.Errorf("ItemTable '%s' の取得に失敗: %w", tableName, err)
	}

	// ルート数を決定する
	countMin := 1
	countMax := 1
	if propRaw.Storage.LootCountMin != nil {
		countMin = int(*propRaw.Storage.LootCountMin)
	}
	if propRaw.Storage.LootCountMax != nil {
		countMax = int(*propRaw.Storage.LootCountMax)
	}
	if countMin > countMax {
		countMin = countMax
	}

	lootCount := countMin
	if countMax > countMin {
		lootCount = countMin + metaPlan.RNG.IntN(countMax-countMin+1)
	}

	// ダンジョン深度を取得する。未設定の場合は深度1として扱う
	depth := 1
	if d := worldhelper.GetDungeon(world); d != nil {
		depth = d.Depth
	}

	for i := 0; i < lootCount; i++ {
		itemName, err := raw.SelectItemByWeight(*metaPlan.RawMaster, itemTable, metaPlan.RNG, depth)
		if err != nil {
			return fmt.Errorf("アイテム抽選エラー: %w", err)
		}
		if itemName == "" {
			continue
		}

		if _, err := worldhelper.SpawnStorageItem(world, itemName, 1, storageEntity); err != nil {
			return fmt.Errorf("アイテム '%s' の生成に失敗: %w", itemName, err)
		}
	}

	return nil
}
