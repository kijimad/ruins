package mapspawner

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	mapplanner "github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"

	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
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
	for _i, tile := range metaPlan.Tiles {
		i := gc.TileIdx(_i)
		pos := metaPlan.Level.IndexToCoord(i)
		tileX, tileY := pos.X+offsetX, pos.Y+offsetY

		tileEntity, err := spawnTile(world, metaPlan, tile, i, tileX, tileY)
		if err != nil {
			return fmt.Errorf("タイルエンティティ生成エラー (%d, %d): %w", int(pos.X), int(pos.Y), err)
		}

		// TileRaw の環境情報を TileTemperature に設定する
		if world.Components.TileTemperature.Has(tileEntity) {
			tileTemp := world.Components.TileTemperature.Get(tileEntity)
			tileTemp.Shelter = gc.ShelterType(tile.Shelter)
			tileTemp.Water = gc.WaterType(tile.Water)
			tileTemp.Foliage = gc.FoliageType(tile.Foliage)
		}
	}
	return nil
}

// tileSpec は1種類のタイルをどう実体化するかの仕様。
// プランナーが出力する論理名 tile.Name をキーに引く。
type tileSpec struct {
	// spawnName は生成するスプライト名。多くは論理名と同じだが wall→dwall のように異なるものもある
	spawnName string
	// autotile は周囲を見てオートタイル添字を計算するか。void のように単一絵柄のタイルは false
	autotile bool
}

var (
	// 歩行可能タイル
	passableTileSpecs = map[string]tileSpec{
		consts.TileNameDirt:    {spawnName: consts.TileNameDirt, autotile: true},
		consts.TileNameFloor:   {spawnName: consts.TileNameFloor, autotile: true},
		consts.TileNameBridgeA: {spawnName: consts.TileNameBridgeA, autotile: true},
		consts.TileNameBridgeB: {spawnName: consts.TileNameBridgeB, autotile: true},
		consts.TileNameBridgeC: {spawnName: consts.TileNameBridgeC, autotile: true},
		consts.TileNameBridgeD: {spawnName: consts.TileNameBridgeD, autotile: true},
	}
	// 通行不可タイル
	blockedTileSpecs = map[string]tileSpec{
		consts.TileNameWall: {spawnName: consts.TileNameDWall, autotile: true},
		consts.TileNameVoid: {spawnName: consts.TileNameVoid, autotile: false},
	}
)

// spawnTile は1タイルを生成する。
// 通行可否で仕様表を選び、論理名 tile.Name で仕様を引いて実体化する。
func spawnTile(world w.World, metaPlan *mapplanner.MetaPlan, tile oapi.Tile, i gc.TileIdx, tileX, tileY consts.Tile) (ecs.Entity, error) {
	specs := passableTileSpecs
	category := "歩行可能"
	if tile.BlockPass {
		specs = blockedTileSpecs
		category = "通行不可"
	}

	spec, ok := specs[tile.Name]
	if !ok {
		return gc.InvalidEntity, fmt.Errorf("未対応の%sタイル名: %s (%d, %d)", category, tile.Name, int(tileX), int(tileY))
	}

	// オートタイル添字は論理名 tile.Name で計算する。生成スプライト名 spec.spawnName とは別物。
	var indexPtr *int
	if spec.autotile {
		index := int(metaPlan.CalculateAutoTileIndex(i, tile.Name))
		indexPtr = &index
	}
	return lifecycle.SpawnTile(world, spec.spawnName, tileX, tileY, indexPtr)
}

// spawnNPCs はNPCを生成する
func spawnNPCs(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, npc := range metaPlan.NPCs {
		member, err := raw.FindMember(world.Resources.RawMaster, npc.Name)
		if err != nil {
			return fmt.Errorf("NPC '%s' が見つかりません", npc.Name)
		}

		x, y := int(npc.X)+int(offsetX), int(npc.Y)+int(offsetY)
		if member.FactionType != nil && string(*member.FactionType) == gc.FactionNeutralName {
			_, err := lifecycle.SpawnNeutralNPC(world, consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, npc.Name)
			if err != nil {
				return fmt.Errorf("中立NPC生成エラー (%d, %d): %w", x, y, err)
			}
		} else {
			var opts []lifecycle.SpawnEnemyOption
			if member.IsBoss {
				opts = append(opts, lifecycle.WithBoss())
			}
			_, err := lifecycle.SpawnEnemy(world, consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}, npc.Name, opts...)
			if err != nil {
				return fmt.Errorf("敵NPC生成エラー (%d, %d): %w", x, y, err)
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
			return fmt.Errorf("アイテムの個数が不正です (%d, %d): count=%d", item.X, item.Y, item.Count)
		}
		_, err := lifecycle.SpawnFieldItem(world, item.Name, tileX, tileY, item.Count)
		if err != nil {
			return fmt.Errorf("アイテム生成エラー (%d, %d): %w", item.X, item.Y, err)
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
			return fmt.Errorf("props取得エラー (%s): %w", prop.Name, err)
		}

		propEntity, err := lifecycle.SpawnProp(world, prop.Name, tileX, tileY)
		if err != nil {
			return fmt.Errorf("props生成エラー (%d, %d): %w", prop.X, prop.Y, err)
		}

		// Door componentがあれば向きを設定して閉じた状態で初期化
		if propRaw.Door != nil {
			doorComp := world.Components.Door.Get(propEntity)
			doorComp.Orientation = detectPropDoorOrientation(metaPlan, int(prop.X), int(prop.Y))
			if err := lifecycle.CloseDoor(world, propEntity); err != nil {
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
func spawnDoors(world w.World, metaPlan *mapplanner.MetaPlan, offsetX, offsetY consts.Tile) error {
	for _, door := range metaPlan.Doors {
		tileX, tileY := door.X+offsetX, door.Y+offsetY
		_, err := lifecycle.SpawnDoor(world, tileX, tileY, door.Orientation)
		if err != nil {
			return fmt.Errorf("ドア生成エラー (%d, %d): %w", door.X, door.Y, err)
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
			return fmt.Errorf("NextPortal生成エラー (%d, %d): %w", portal.X, portal.Y, err)
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
	if d := query.GetDungeon(world); d != nil {
		depth = d.CurrentStage.Depth
	}

	for range lootCount {
		itemName, err := raw.SelectItemByWeight(*metaPlan.RawMaster, itemTable, metaPlan.RNG, depth)
		if err != nil {
			return fmt.Errorf("アイテム抽選エラー: %w", err)
		}
		if itemName == "" {
			continue
		}

		if _, err := lifecycle.SpawnStorageItem(world, itemName, 1, storageEntity); err != nil {
			return fmt.Errorf("アイテム '%s' の生成に失敗: %w", itemName, err)
		}
	}

	return nil
}
