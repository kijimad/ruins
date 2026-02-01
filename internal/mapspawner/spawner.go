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

		// TODO: タイル名直判定だと忘れやすいので直したい
		if !tile.BlockPass {
			// 歩行可能タイルを生成
			switch tile.Name {
			case "dirt":
				index := int(metaPlan.CalculateAutoTileIndex(i, "dirt"))
				_, err = worldhelper.SpawnTile(world, "dirt", tileX, tileY, &index)
			case "floor":
				index := int(metaPlan.CalculateAutoTileIndex(i, "floor"))
				_, err = worldhelper.SpawnTile(world, "floor", tileX, tileY, &index)
			case "bridge_a", "bridge_b", "bridge_c", "bridge_d":
				// 橋タイルは通常の床タイルとして生成（見た目は同じ）
				index := int(metaPlan.CalculateAutoTileIndex(i, tile.Name))
				_, err = worldhelper.SpawnTile(world, tile.Name, tileX, tileY, &index)
			default:
				// 未知のタイル名はエラーとして処理
				return resources.Level{}, fmt.Errorf("未対応の歩行可能タイル名: %s (%d, %d)", tile.Name, int(x), int(y))
			}
		} else {
			// 通行不可タイルを生成
			switch tile.Name {
			case "wall":
				// 隣接に床がある場合のみ壁エンティティを生成
				if metaPlan.AdjacentAnyFloor(i) {
					index := int(metaPlan.CalculateAutoTileIndex(i, "wall"))
					_, err = worldhelper.SpawnTile(world, "wall", tileX, tileY, &index)
				}
			case "void":
				_, err = worldhelper.SpawnTile(world, "void", tileX, tileY, nil)
			default:
				return resources.Level{}, fmt.Errorf("未対応の通行不可タイル名: %s (%d, %d)", tile.Name, int(x), int(y))
			}
		}

		if err != nil {
			return resources.Level{}, fmt.Errorf("タイルエンティティ生成エラー (%d, %d): %w", int(x), int(y), err)
		}
	}

	// NPCを生成する
	rawMaster := world.Resources.RawMaster.(*raw.Master)
	for _, npc := range metaPlan.NPCs {
		// NPCが中立かどうかを判断
		memberIdx, ok := rawMaster.MemberIndex[npc.Name]
		if !ok {
			return resources.Level{}, fmt.Errorf("NPC '%s' が見つかりません", npc.Name)
		}
		member := rawMaster.Raws.Members[memberIdx]

		if member.FactionType == gc.FactionNeutral.String() {
			// 中立NPCの場合
			_, err := worldhelper.SpawnNeutralNPC(world, npc.X, npc.Y, npc.Name)
			if err != nil {
				return resources.Level{}, fmt.Errorf("中立NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		} else {
			// 敵NPCの場合
			_, err := worldhelper.SpawnEnemy(world, npc.X, npc.Y, npc.Name)
			if err != nil {
				return resources.Level{}, fmt.Errorf("敵NPC生成エラー (%d, %d): %w", npc.X, npc.Y, err)
			}
		}
	}

	// アイテムを生成する
	for _, item := range metaPlan.Items {
		tileX, tileY := gc.Tile(item.X), gc.Tile(item.Y)
		_, err := worldhelper.SpawnFieldItem(world, item.Name, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("アイテム生成エラー (%d, %d): %w", item.X, item.Y, err)
		}
	}

	// Propsを生成する
	for _, prop := range metaPlan.Props {
		tileX, tileY := gc.Tile(prop.X), gc.Tile(prop.Y)

		propRaw, err := metaPlan.RawMaster.GetProp(prop.Name)
		if err != nil {
			return resources.Level{}, fmt.Errorf("props取得エラー (%s): %w", prop.Name, err)
		}

		propEntity, err := worldhelper.SpawnProp(world, prop.Name, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("props生成エラー (%d, %d): %w", prop.X, prop.Y, err)
		}

		// Door componentがあれば向きを設定して閉じた状態で初期化
		if propRaw.Door != nil {
			orientation := detectDoorOrientation(metaPlan, prop.X, prop.Y)
			doorComp := world.Components.Door.Get(propEntity).(*gc.Door)
			doorComp.Orientation = orientation
			if err := worldhelper.CloseDoor(world, propEntity); err != nil {
				return resources.Level{}, fmt.Errorf("ドア初期化エラー (%d, %d): %w", prop.X, prop.Y, err)
			}
		}
	}

	// 出口エンティティを生成する
	for _, exit := range metaPlan.Exits {
		tileX, tileY := gc.Tile(exit.X), gc.Tile(exit.Y)

		_, err := worldhelper.SpawnBridge(world, exit.ExitID, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("出口エンティティ生成エラー (%d, %d): %w", exit.X, exit.Y, err)
		}
	}

	// 橋ヒントエンティティを生成する
	for _, hint := range metaPlan.BridgeHints {
		tileX, tileY := gc.Tile(hint.X), gc.Tile(hint.Y)
		_, err := worldhelper.SpawnBridgeHint(world, hint.ExitID, tileX, tileY)
		if err != nil {
			return resources.Level{}, fmt.Errorf("橋ヒントエンティティ生成エラー (%d, %d): %w", hint.X, hint.Y, err)
		}
	}

	return level, nil
}

// detectDoorOrientation は隣接タイルからドアの向きを判定する
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
