package overworld

import (
	"fmt"
	"math/rand/v2"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner/interior"
	"github.com/kijimaD/ruins/internal/raw"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/mlange-42/ark/ecs"
)

// furnishBuilding は建物区画 footprint を敷地計画し、Site が返す庭・壁・部屋を描いて内装で満たす。footprint を
// そのまま埋めず、interior.FurnishBuilding が前庭を空け坪庭を作り玄関を凹ませる。庭は土、壁は壁、残りの
// 部屋の床と戸口は床として描き、入口の扉と家具を spawn する。内装の乱数は建物幾何と別ストリーム 0x3 に
// する。壁判定の関数と占有タイルを返し、後段の敵配置が壁や家具の上に湧かないよう避けさせる。
func furnishBuilding(world w.World, g chunkGeom, footprint interior.Rect, door interior.Vec, orient gc.DoorOrientation, fac facilityType, seed uint64) (func(lx, ly consts.Tile) bool, map[consts.Coord[consts.Tile]]bool, error) {
	iseed := rand.New(rand.NewPCG(seed, 0x3)).Uint64()
	site, placed := interior.FurnishBuilding(iseed, footprint, door, string(fac))

	wallSet := make(map[interior.Vec]bool)
	for _, wv := range site.Walls() {
		wallSet[wv] = true
	}

	tiles := g.tiles.get()
	occupied := make(map[consts.Coord[consts.Tile]]bool)
	// 建物区画のタイルを Site から描く。庭→土、壁→壁、残り(部屋の床・戸口)→床。footprint 全体を占有に入れ、
	// 街の敵が封鎖された建物の内側や庭・坪庭に湧かないようにする。敵は街路を歩くもので、建物の中は入口の
	// 扉を通ってのみ入るため、壁だけでなく屋内の床・庭も敵の湧き先から外す
	for y := footprint.Y; y < footprint.Y+footprint.H; y++ {
		for x := footprint.X; x < footprint.X+footprint.W; x++ {
			v := interior.Vec{X: x, Y: y}
			coord := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(x), Y: g.offsetY + consts.Tile(y)}
			name := consts.TileNameFloor
			switch {
			case site.Garden[v]:
				name = consts.TileNameDirt
			case wallSet[v]:
				name = consts.TileNameDWall
			}
			if err := replaceTile(world, tiles, coord, name); err != nil {
				return nil, nil, fmt.Errorf("内装のタイル配置に失敗 (x=%d, y=%d): %w", coord.X, coord.Y, err)
			}
			occupied[coord] = true
		}
	}

	// 入口の扉を建物辺の site.Door へ置く。前庭ぶん内寄せした建物の辺にあり、前庭が街路との間に挟まる
	dcoord := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(site.Door.X), Y: g.offsetY + consts.Tile(site.Door.Y)}
	if _, err := lifecycle.SpawnDoor(world, dcoord, orient); err != nil {
		return nil, nil, fmt.Errorf("内装の扉配置に失敗: %w", err)
	}

	// 家具と装飾を spawn する。写像できる Ref だけを建物の内側へ置く。坪庭の観葉もここで庭の土の上へ乗る。
	// 写像は interior.PropRawName が持つ単一のソースで、VRT の描画も同じ判定を共有する。収納家具には戦利品を
	// 格納するので、建物ごとに別ストリーム 0x4 の決定的 RNG で引く。グローバル乱数でなく建物ローカルで
	// 決定的にし、再訪で一致させる
	lootRNG := rand.New(rand.NewPCG(seed, 0x4))
	for _, p := range placed {
		name, ok := interior.PropRawName(p.Ref)
		if !ok {
			continue // raw の無い戦利品や装飾は置かない
		}
		pos := consts.Coord[consts.Tile]{X: g.offsetX + consts.Tile(p.Pos.X), Y: g.offsetY + consts.Tile(p.Pos.Y)}
		ent, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y)
		if err != nil {
			return nil, nil, fmt.Errorf("内装の配置に失敗 (%s at %d,%d): %w", name, pos.X, pos.Y, err)
		}
		if err := populateStorageLoot(world, ent, name, lootRNG); err != nil {
			return nil, nil, fmt.Errorf("収納の戦利品格納に失敗 (%s): %w", name, err)
		}
		occupied[pos] = true
	}

	isWall := func(lx, ly consts.Tile) bool { return wallSet[interior.Vec{X: int(lx), Y: int(ly)}] }
	return isWall, occupied, nil
}

// populateStorageLoot は収納家具に戦利品を格納する。prop の raw が Storage.LootTableName を持てば、その item
// テーブルから件数ぶん重み抽選して収納エンティティへ入れる。家具別の loot テーブルを、ruins 既存の
// DropTable/ItemTable と日本語テーブル(廃墟等)で実現する。ダンジョン生成の populateStorageLoot と同型で、
// overworld は建物ローカルの決定的 RNG を使う。地上フロアなので深度は 0。
func populateStorageLoot(world w.World, entity ecs.Entity, propName string, rng *rand.Rand) error {
	propRaw, err := raw.GetProp(world.Resources.RawMaster, propName)
	if err != nil {
		return err
	}
	if propRaw.Storage == nil || propRaw.Storage.LootTableName == nil || *propRaw.Storage.LootTableName == "" {
		return nil // 収納でない家具は何も入れない
	}
	itemTable, err := raw.GetItemTable(world.Resources.RawMaster, *propRaw.Storage.LootTableName)
	if err != nil {
		return err
	}
	countMin, countMax := 1, 1
	if propRaw.Storage.LootCountMin != nil {
		countMin = int(*propRaw.Storage.LootCountMin)
	}
	if propRaw.Storage.LootCountMax != nil {
		countMax = int(*propRaw.Storage.LootCountMax)
	}
	if countMin > countMax {
		countMin = countMax
	}
	n := countMin
	if countMax > countMin {
		n = countMin + rng.IntN(countMax-countMin+1)
	}
	for range n {
		// 深度は地上の建物なので浅い loot の1を使う。廃墟テーブルは全 entry が minDepth>=1 なので深度0では
		// 何も引けない。地上の廃屋あさりは最も浅い戦利品が出る
		itemName, err := raw.SelectItemByWeight(world.Resources.RawMaster, itemTable, rng, 1)
		if err != nil {
			return err
		}
		if itemName == "" {
			continue
		}
		if _, err := lifecycle.SpawnStorageItem(world, itemName, 1, entity); err != nil {
			return err
		}
	}
	return nil
}
