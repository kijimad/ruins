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
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// furnishBuilding は建物区画 footprint を敷地計画し、Site が返す庭・壁・部屋を描いて内装で満たす。footprint を
// そのまま埋めず、interior.FurnishBuilding が前庭を空け坪庭を作り玄関を凹ませる。庭は土、壁は壁、残りの
// 部屋の床と戸口は床として描き、入口の扉と家具を spawn する。内装の乱数は建物幾何と別ストリーム 0x3 に
// する。壁判定の関数と占有タイルを返し、後段の敵配置が壁や家具の上に湧かないよう避けさせる。
func furnishBuilding(world w.World, g chunkGeom, footprint interior.Rect, door interior.Vec, fac facilityType, seed uint64) (func(lx, ly consts.Tile) bool, map[consts.Coord[consts.Tile]]bool, error) {
	iseed := rand.New(rand.NewPCG(seed, 0x3)).Uint64()
	site, placed := interior.FurnishBuilding(iseed, footprint, door, interior.FacilityKind(fac))

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
			coord := consts.Coord[consts.Tile]{X: g.offsetX + x, Y: g.offsetY + y}
			name := consts.TileNameFloor
			switch {
			case site.Garden[v]:
				name = consts.TileNameDirt
			case wallSet[v]:
				name = consts.TileNameDWall
			}
			if err := replaceTile(world, tiles, coord, name); err != nil {
				return nil, nil, fmt.Errorf("failed to place interior tile (x=%d, y=%d): %w", coord.X, coord.Y, err)
			}
			occupied[coord] = true
		}
	}

	// 入口の扉を建物辺の site.Door へ置く。前庭ぶん内寄せした建物の辺にあり、前庭が街路との間に挟まる。
	// 向きは入口も部屋間の戸口も同じ doorOrientation で壁の走る方向から決め、規約を1箇所に集約する
	dcoord := consts.Coord[consts.Tile]{X: g.offsetX + site.Door.X, Y: g.offsetY + site.Door.Y}
	if _, err := lifecycle.SpawnDoor(world, dcoord, doorOrientation(wallSet, site.Door)); err != nil {
		return nil, nil, fmt.Errorf("failed to place interior door: %w", err)
	}

	// 部屋間の戸口にも扉を置く。interior は戸口を壁の切れ目として持つが、扉エンティティは overworld が立てる。
	// 入口と同じく閉状態で通行と視界を遮り、ぶつかると開く。同じ戸口は隣接2部屋が共有するので座標で重複排除し、
	// 入口の扉とも重ねない
	doorSeen := map[interior.Vec]bool{site.Door: true}
	for _, hr := range site.Rooms {
		for _, dw := range hr.Room.Doorways {
			dv := dw
			if doorSeen[dv] {
				continue
			}
			doorSeen[dv] = true
			ic := consts.Coord[consts.Tile]{X: g.offsetX + dv.X, Y: g.offsetY + dv.Y}
			if _, err := lifecycle.SpawnDoor(world, ic, doorOrientation(wallSet, dv)); err != nil {
				return nil, nil, fmt.Errorf("failed to place interior partition door: %w", err)
			}
		}
	}

	// 配置指示を spawn する。建物ごとに別ストリーム 0x4 の決定的 RNG で loot を引く。グローバル乱数でなく
	// 建物ローカルで決定的にし、再訪で一致させる。KindLoot は床 loot として実アイテムに実体化し、それ以外の
	// 家具と装飾は prop として置く。写像は interior が持つ単一のソースで、VRT の描画も同じ判定を共有する
	lootRNG := rand.New(rand.NewPCG(seed, 0x4))
	for _, p := range placed {
		pos := consts.Coord[consts.Tile]{X: g.offsetX + p.Pos.X, Y: g.offsetY + p.Pos.Y}
		switch p.Kind {
		case interior.KindLoot:
			// 抽象 loot Ref を item group へ写し、1山を抽選して床へ置く。写像を持たない Ref は置かない
			groupID, ok := interior.LootGroupName(p.Ref)
			if !ok {
				continue
			}
			if err := spawnFieldLoot(world, groupID, pos, lootRNG); err != nil {
				return nil, nil, fmt.Errorf("failed to place field loot (%s at %d,%d): %w", groupID, pos.X, pos.Y, err)
			}
			occupied[pos] = true
		default:
			// 家具と装飾。写像できる Ref だけを建物の内側へ置く。坪庭の観葉もここで庭の土の上へ乗る。
			// 収納家具には戦利品を格納する。raw の無い装飾や、敵・罠など prop でない指示は置かない
			name, ok := interior.PropRawName(p.Ref)
			if !ok {
				continue
			}
			ent, err := lifecycle.SpawnProp(world, name, pos.X, pos.Y)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to place interior prop (%s at %d,%d): %w", name, pos.X, pos.Y, err)
			}
			if err := populateStorageLoot(world, ent, name, lootRNG); err != nil {
				return nil, nil, fmt.Errorf("failed to populate storage loot (%s): %w", name, err)
			}
			occupied[pos] = true
		}
	}

	isWall := func(lx, ly consts.Tile) bool { return wallSet[interior.Vec{X: lx, Y: ly}] }
	return isWall, occupied, nil
}

// doorOrientation は扉の向きを、扉が乗る壁の走る方向から決める。左右が壁の東西に走る壁の切れ目は Vertical、
// 上下が壁の南北に走る壁は Horizontal。入口も部屋間の戸口も同じこの規約で向きを揃え、規約を1箇所に集約する。
func doorOrientation(wallSet map[interior.Vec]bool, pos interior.Vec) gc.DoorOrientation {
	if wallSet[interior.Vec{X: pos.X - 1, Y: pos.Y}] && wallSet[interior.Vec{X: pos.X + 1, Y: pos.Y}] {
		return gc.DoorOrientationVertical
	}
	return gc.DoorOrientationHorizontal
}

// spawnFieldLoot は loot の item group から1山を抽選し、床へアイテムを spawn する。抽選は
// raw.SelectFromItemGroup が distribution/collection と pack を解釈する。地上フロアなので深度は扱わず
// group を直接引く。収納 loot の populateStorageLoot と同型で、建物ローカルの決定的 RNG を使い再訪で一致させる。
func spawnFieldLoot(world w.World, groupID string, pos consts.Coord[consts.Tile], rng *rand.Rand) error {
	draws, err := raw.SelectFromItemGroup(world.Resources.RawMaster, groupID, rng)
	if err != nil {
		return err
	}
	for _, d := range draws {
		if d.Name == "" {
			continue
		}
		if _, err := lifecycle.SpawnFieldItem(world, d.Name, pos.X, pos.Y, d.Count); err != nil {
			return err
		}
	}
	return nil
}

// populateStorageLoot は収納家具に戦利品を格納する。prop の raw が Storage.LootTableId を持てば、その item
// テーブルから件数ぶん重み抽選して収納エンティティへ入れる。家具別の loot テーブルを、ruins 既存の
// DropTable/ItemTable と日本語テーブル(廃墟等)で実現する。ダンジョン生成の populateStorageLoot と同型で、
// overworld は建物ローカルの決定的 RNG を使う。地上フロアなので深度は 0。
func populateStorageLoot(world w.World, entity ecs.Entity, propName string, rng *rand.Rand) error {
	propRaw, err := raw.GetProp(world.Resources.RawMaster, propName)
	if err != nil {
		return err
	}
	if propRaw.Storage == nil || propRaw.Storage.LootTableId == nil || *propRaw.Storage.LootTableId == "" {
		return nil // 収納でない家具は何も入れない
	}
	itemTable, err := raw.GetItemTable(world.Resources.RawMaster, *propRaw.Storage.LootTableId)
	if err != nil {
		return err
	}
	// ルート数はダイス表記で決める。省略時は1個
	lootDice := consts.Dice{Base: 1, Sides: 1}
	if propRaw.Storage.LootCount != nil {
		d, err := consts.ParseDice(*propRaw.Storage.LootCount)
		if err != nil {
			return fmt.Errorf("invalid lootCount notation for storage '%s': %w", propName, err)
		}
		lootDice = d
	}
	// 危険度は経過日数で決める。日が進むほど希少な loot が出る。廃墟テーブルは全 entry が
	// minDanger>=1 で危険度0では何も引けないため、序盤でも最も浅い戦利品が出るよう1で下限を張る。
	danger := max(1, query.DangerLevelAt(world))
	n := lootDice.Roll(rng)
	for range n {
		itemName, err := raw.SelectItemByWeight(world.Resources.RawMaster, itemTable, rng, danger)
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
