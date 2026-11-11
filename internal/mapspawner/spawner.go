package mapspawner

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
)

var (
	// errUnsupportedTile は生成の仕様 spawn を持たないタイルを生成しようとしたときに返す
	errUnsupportedTile = errors.New("unsupported tile id")
	// errInvalidItemCount はアイテムの個数が0以下のときに返す
	errInvalidItemCount = errors.New("invalid item count")
	// errInvalidLootCount は収納propのルート数の表記が不正なときに返す
	errInvalidLootCount = errors.New("invalid lootCount notation")
	// errDrawLoot はルートテーブルからの抽選に失敗したときに返す
	errDrawLoot = errors.New("failed to draw item")
)

// Spawn はMetaPlanからレベルを生成する
// タイル、NPC、Props、ワープポータル情報から効率的にエンティティを生成する
func Spawn(world w.World, metaPlan *mapplanner.MetaPlan) (gc.Level, error) {
	return SpawnAt(world, metaPlan, 0, 0)
}

// SpawnAt は MetaPlan を offsetX, offsetY タイルずらして生成する。
// オーバーワールドで、チャンクを帯の東スラブなど任意位置へ配置するために使う。
// オフセットはエンティティ座標にのみ加算し、オートタイルや扉向きの判定は
// プラン内ローカル座標、すなわち metaPlan.Tiles のインデックスで行うため影響しない。
// 現状 offsetY は常に 0 で南北ストリーミングしない帯だが、将来の 2D 配置・対称性のため引数に残す。
func SpawnAt(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) (gc.Level, error) {
	level := gc.Level{
		TileWidth:  metaPlan.Level.TileWidth,
		TileHeight: metaPlan.Level.TileHeight,
	}

	if err := spawnTiles(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}
	if err := spawnNPCs(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}
	if err := spawnItems(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}
	if err := spawnProps(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}
	if err := spawnDoors(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}
	if err := spawnPortals(world, metaPlan, offsetX, offsetY); err != nil {
		return gc.Level{}, err
	}

	return level, nil
}

// spawnTiles はタイルからエンティティを生成する
func spawnTiles(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _i := range metaPlan.Tiles {
		tile := &metaPlan.Tiles[_i]
		i := gc.TileIdx(_i)
		pos := metaPlan.Level.IndexToCoord(i)
		tileX, tileY := pos.X+offsetX, pos.Y+offsetY

		tileEntity, err := spawnTile(world, metaPlan, *tile, i, tileX, tileY)
		if err != nil {
			return fmt.Errorf("failed to spawn tile entity (%d, %d): %w", int(pos.X), int(pos.Y), err)
		}

		if metaPlan.Environments != nil && world.Components.TileEnvironment.Has(tileEntity) {
			*world.Components.TileEnvironment.Get(tileEntity) = metaPlan.Environments[i]
		}
	}
	return nil
}

// spawnTile は1タイルを生成する。タイルの行の spawn が生成するタイルとオートタイルの有無を持ち、
// spawn を持たないタイルは計画に置けないので error にする。
func spawnTile(world w.World, metaPlan *mapplanner.MetaPlan, tile oapi.Tile, i gc.TileIdx, tileX, tileY consts.Tile) (ecs.Entity, error) {
	if tile.Spawn == nil {
		category := "walkable"
		if tile.BlockPass {
			category = "impassable"
		}
		return gc.InvalidEntity, fmt.Errorf("%w: %s %s (%d, %d)", errUnsupportedTile, category, tile.Id, int(tileX), int(tileY))
	}
	spawnName := tile.Id
	if tile.Spawn.Tile != nil {
		spawnName = *tile.Spawn.Tile
	}

	// オートタイル添字は tile.Id で計算する。生成するタイル spawnName とは別物。
	var indexPtr *int
	if tile.Spawn.Autotile {
		index := int(metaPlan.CalculateAutoTileIndex(i, tile.Id))
		indexPtr = &index
	}
	return lifecycle.SpawnTile(world, spawnName, tileX, tileY, indexPtr)
}

// spawnNPCs はNPCを生成する
func spawnNPCs(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, npc := range metaPlan.NPCs {
		member, err := raw.FindMember(world.Resources.RawMaster, npc.Name)
		if err != nil {
			return fmt.Errorf("NPC %q: %w", npc.Name, err)
		}

		x, y := int(npc.X)+int(offsetX), int(npc.Y)+int(offsetY)
		if member.FactionType != nil && *member.FactionType == oapi.FactionNeutral {
			_, err := lifecycle.SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, npc.Name)
			if err != nil {
				return fmt.Errorf("failed to spawn neutral NPC (%d, %d): %w", x, y, err)
			}
		} else {
			_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, npc.Name)
			if err != nil {
				return fmt.Errorf("failed to spawn enemy NPC (%d, %d): %w", x, y, err)
			}
		}
	}
	return nil
}

// spawnItems はアイテムを生成する
func spawnItems(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, item := range metaPlan.Items {
		tileX, tileY := item.X+offsetX, item.Y+offsetY
		if item.Count <= 0 {
			return fmt.Errorf("%w (%d, %d): count=%d", errInvalidItemCount, item.X, item.Y, item.Count)
		}
		_, err := lifecycle.SpawnFieldItem(world, item.Name, tileX, tileY, item.Count)
		if err != nil {
			return fmt.Errorf("failed to spawn item (%d, %d): %w", item.X, item.Y, err)
		}
	}
	return nil
}

// spawnProps はPropsを生成する
func spawnProps(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, prop := range metaPlan.Props {
		tileX, tileY := prop.X+offsetX, prop.Y+offsetY

		propRaw, err := raw.GetProp(*metaPlan.RawMaster, prop.Name)
		if err != nil {
			return fmt.Errorf("failed to get props (%s): %w", prop.Name, err)
		}

		propEntity, err := lifecycle.SpawnProp(world, prop.Name, tileX, tileY)
		if err != nil {
			return fmt.Errorf("failed to spawn props (%d, %d): %w", prop.X, prop.Y, err)
		}

		// Door componentがあれば向きを設定して閉じた状態で初期化
		if propRaw.Door != nil {
			doorComp := world.Components.Door.Get(propEntity)
			doorComp.Orientation = detectPropDoorOrientation(metaPlan, int(prop.X), int(prop.Y))
			if err := lifecycle.CloseDoor(world, propEntity); err != nil {
				return fmt.Errorf("failed to initialize door (%d, %d): %w", prop.X, prop.Y, err)
			}
		}

		// Storage propにルートアイテムを格納する
		if propRaw.Storage != nil && propRaw.Storage.LootTableId != nil && *propRaw.Storage.LootTableId != "" {
			if err := populateStorageLoot(world, metaPlan, propEntity, propRaw); err != nil {
				return fmt.Errorf("failed to spawn container item (%d, %d): %w", prop.X, prop.Y, err)
			}
		}
	}
	return nil
}

// spawnDoors はドアを生成する
func spawnDoors(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, door := range metaPlan.Doors {
		tileX, tileY := door.X+offsetX, door.Y+offsetY
		_, err := lifecycle.SpawnDoor(world, consts.Coord[consts.Tile]{X: tileX, Y: tileY}, door.Orientation)
		if err != nil {
			return fmt.Errorf("failed to spawn door (%d, %d): %w", door.X, door.Y, err)
		}
	}
	return nil
}

// spawnPortals はポータルを生成する
func spawnPortals(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, portal := range metaPlan.NextPortals {
		tileX, tileY := portal.X+offsetX, portal.Y+offsetY
		_, err := lifecycle.SpawnProp(world, "warp_next", tileX, tileY)
		if err != nil {
			return fmt.Errorf("failed to spawn NextPortal (%d, %d): %w", portal.X, portal.Y, err)
		}
	}

	return nil
}

// detectPropDoorOrientation はpropsの扉の向きを隣接タイルから判定する。
// DoorSpecを持たないprops扉専用で、左右が壁なら縦向き、それ以外は横向きを返す
func detectPropDoorOrientation(metaPlan *mapplanner.MetaPlan, x, y int) gc.DoorOrientation {
	width := int(metaPlan.Level.TileWidth)
	height := int(metaPlan.Level.TileHeight)

	if x <= 0 || x >= width-1 || y <= 0 || y >= height-1 {
		return gc.DoorOrientationHorizontal
	}

	idx := y*width + x
	if metaPlan.Tiles[idx-1].BlockPass && metaPlan.Tiles[idx+1].BlockPass {
		return gc.DoorOrientationVertical
	}

	return gc.DoorOrientationHorizontal
}

// populateStorageLoot は収納propにルートテーブルからアイテムを格納する
func populateStorageLoot(world w.World, metaPlan *mapplanner.MetaPlan, storageEntity ecs.Entity, propRaw oapi.Prop) error {
	tableName := *propRaw.Storage.LootTableId
	itemTable, err := raw.GetItemTable(*metaPlan.RawMaster, tableName)
	if err != nil {
		return fmt.Errorf("failed to get ItemTable '%s': %w", tableName, err)
	}

	// ルート数はダイス表記で決める。省略時は1個
	lootDice := consts.Dice{Base: 1, Sides: 1}
	if propRaw.Storage.LootCount != nil {
		d, err := consts.ParseDice(*propRaw.Storage.LootCount)
		if err != nil {
			return fmt.Errorf("%w for container %q: %w", errInvalidLootCount, propRaw.Name, err)
		}
		lootDice = d
	}
	lootCount := lootDice.Roll(metaPlan.RNG)

	// 危険度は生成中フロアのプランから取る。
	danger := metaPlan.Danger

	for range lootCount {
		draws, err := raw.DrawFromItemTable(*metaPlan.RawMaster, itemTable, metaPlan.RNG, int(danger))
		if err != nil {
			return fmt.Errorf("%w: %w", errDrawLoot, err)
		}
		for _, d := range draws {
			if _, err := lifecycle.SpawnStorageItem(world, d.Name, d.Count, storageEntity); err != nil {
				return fmt.Errorf("failed to spawn item '%s': %w", d.Name, err)
			}
		}
	}

	return nil
}
