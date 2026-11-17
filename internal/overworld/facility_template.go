package overworld

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/mapplanner"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

var (
	// errFacilityOutOfChunk は施設の絵が置く原点からチャンクに収まらないときに返す
	errFacilityOutOfChunk = errors.New("facility template does not fit in the chunk")
	// errFacilityGroundNotEarth は下地を残すマスの下が土の地面でないときに返す
	errFacilityGroundNotEarth = errors.New("facility ground cell is not on an earth tile")
)

// planFacility は施設 fac の絵を1枚選んで展開する。絵は入口を北に向けて描いてあり、rng で半々に左へ90度回して入口を西の街路へ向ける。左右の反転も半々で選ぶ
func planFacility(loader *maptemplate.TemplateLoader, fac string, seed uint64, rng *rand.Rand) ([][]maptemplate.MapCell, error) {
	_, cells, err := loader.FacilityChunk(fac, seed)
	if err != nil {
		return nil, err
	}
	turns := 0
	if rng.IntN(2) == 0 {
		turns = 1
	}
	return maptemplate.TransformCells(cells, turns, rng.IntN(2) == 0), nil
}

// spawnFacility は施設の絵のマスを、チャンク内の区画の左上 (ox, oy) から実体として置く。
// 扉は左右か上下の壁から向きを決める。
// 壁判定の関数と占有したマスを返し、後段の街路の敵が建物の上に湧かないようにする
func spawnFacility(world w.World, g chunkGeom, cells [][]maptemplate.MapCell, ox, oy consts.Tile, fac string, danger consts.Danger, seed uint64) (func(lx, ly consts.Tile) bool, map[consts.Coord[consts.Tile]]bool, error) {
	if int(oy)+len(cells) > int(g.chunkH) || int(ox)+len(cells[0]) > int(g.chunkW) {
		return nil, nil, fmt.Errorf("facility %q %dx%d at %d,%d, chunk %dx%d: %w", fac, len(cells[0]), len(cells), ox, oy, g.chunkW, g.chunkH, errFacilityOutOfChunk)
	}

	tileRows := map[string]rawschema.Tile{}
	tileOf := func(id string) (rawschema.Tile, error) {
		if t, ok := tileRows[id]; ok {
			return t, nil
		}
		t, err := raw.GetTile(world.Resources.RawMaster, id)
		if err != nil {
			return rawschema.Tile{}, err
		}
		tileRows[id] = t
		return t, nil
	}
	inBounds := func(x, y int) bool { return y >= 0 && y < len(cells) && x >= 0 && x < len(cells[y]) }
	wallAt := func(x, y int) bool {
		if !inBounds(x, y) || cells[y][x].Terrain == maptemplate.GroundTerrain {
			return false
		}
		t, err := tileOf(cells[y][x].Terrain)
		return err == nil && t.BlockPass
	}

	propRows := map[string]rawschema.Prop{}
	propOf := func(id string) (rawschema.Prop, error) {
		if p, ok := propRows[id]; ok {
			return p, nil
		}
		p, err := raw.GetProp(world.Resources.RawMaster, id)
		if err != nil {
			return rawschema.Prop{}, err
		}
		propRows[id] = p
		return p, nil
	}
	propBlocksPass := func(id string) (bool, error) {
		if id == "" {
			return false, nil
		}
		prop, err := propOf(id)
		if err != nil {
			return false, err
		}
		return prop.BlockPass, nil
	}

	// 扉の隣には扉や窓が並ぶので、通行を塞ぐ置物も壁と同じく向きを決める側に数える
	blocks := func(x, y int) (bool, error) {
		if !inBounds(x, y) {
			return false, nil
		}
		if wallAt(x, y) {
			return true, nil
		}
		b, err := propBlocksPass(cells[y][x].Prop)
		if err != nil {
			return false, fmt.Errorf("facility %q prop at %d,%d: %w", fac, x, y, err)
		}
		return b, nil
	}

	facings, err := cellPoses(cells, wallAt, propOf)
	if err != nil {
		return nil, nil, fmt.Errorf("facility %q: %w", fac, err)
	}

	tiles := g.tiles.get()
	occupied := make(map[consts.Coord[consts.Tile]]bool)
	lootRNG := rand.New(rand.NewPCG(seed, 0x4))
	beingRNG := rand.New(rand.NewPCG(seed, 0x5))
	for y, row := range cells {
		for x, cell := range row {
			pos := consts.Coord[consts.Tile]{X: g.offsetX + ox + consts.Tile(x), Y: g.offsetY + oy + consts.Tile(y)}
			occupied[pos] = true

			if cell.Terrain == maptemplate.GroundTerrain {
				if err := checkFacilityGround(world, tiles, pos, tileOf); err != nil {
					return nil, nil, fmt.Errorf("facility %q at %d,%d: %w", fac, x, y, err)
				}
			} else {
				t, err := tileOf(cell.Terrain)
				if err != nil {
					return nil, nil, fmt.Errorf("facility %q tile at %d,%d: %w", fac, x, y, err)
				}
				if err := spawnFacilityTile(world, tiles, pos, t); err != nil {
					return nil, nil, err
				}
			}
			orientation := gc.DoorOrientationHorizontal
			left, err := blocks(x-1, y)
			if err != nil {
				return nil, nil, err
			}
			right, err := blocks(x+1, y)
			if err != nil {
				return nil, nil, err
			}
			if left && right {
				orientation = gc.DoorOrientationVertical
			}
			if err := spawnFacilityCell(world, cell, pos, orientation, facings[consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}], fac, danger, lootRNG, beingRNG); err != nil {
				return nil, nil, err
			}
		}
	}

	isWall := func(lx, ly consts.Tile) bool { return wallAt(int(lx-ox), int(ly-oy)) }
	return isWall, occupied, nil
}

// checkFacilityGround は下地を残すマス pos の下に、土の地面のタイルが置かれていることを確かめる
func checkFacilityGround(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], tileOf func(string) (rawschema.Tile, error)) error {
	e, ok := tiles[gc.GridElement{Coord: pos}]
	if !ok || !world.ECS.Alive(e) || !world.Components.RawID.Has(e) {
		return errFacilityGroundNotEarth
	}
	t, err := tileOf(world.Components.RawID.Get(e).ID)
	if err != nil {
		return err
	}
	if t.Earth == nil || !*t.Earth {
		return errFacilityGroundNotEarth
	}
	return nil
}

// spawnFacilityTile はタイルの行 t を pos へ置く。行の生成の仕様が別のタイルを指せばそれを置き、オートタイルしないタイルは添字を付けない
func spawnFacilityTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], t rawschema.Tile) error {
	name := t.Id
	if t.Spawn != nil && t.Spawn.Tile != nil {
		name = *t.Spawn.Tile
	}
	var index *int
	if t.Spawn != nil && t.Spawn.Autotile {
		index = new(0)
	}
	if _, err := replaceTileIndexed(world, tiles, pos, name, index); err != nil {
		return fmt.Errorf("failed to place facility tile (x=%d, y=%d): %w", pos.X, pos.Y, err)
	}
	return nil
}

// spawnFacilityCell はマスの扉、家具と収納の中身、床の戦利品、敵を pos へ置く。扉の向きは呼び出し側が壁から決める
func spawnFacilityCell(world w.World, cell maptemplate.MapCell, pos consts.Coord[consts.Tile], orientation gc.DoorOrientation, facing *gc.Pose, fac string, danger consts.Danger, lootRNG, beingRNG *rand.Rand) error {
	switch {
	case cell.Prop == "door":
		if _, err := lifecycle.SpawnDoor(world, pos, orientation); err != nil {
			return fmt.Errorf("failed to place facility door at %d,%d: %w", pos.X, pos.Y, err)
		}
	case cell.Prop != "" && !slices.ContainsFunc(query.GetEntitiesAt(world, pos.X, pos.Y), world.Components.Prop.Has):
		ent, err := lifecycle.SpawnProp(world, cell.Prop, pos.X, pos.Y)
		if err != nil {
			return fmt.Errorf("failed to place facility prop (%s at %d,%d): %w", cell.Prop, pos.X, pos.Y, err)
		}
		if facing != nil {
			world.Components.Pose.Add(ent, facing)
		}
		if err := populateStorageLoot(world, ent, cell.Prop, cell.LootTable, fac, lootRNG); err != nil {
			return fmt.Errorf("failed to populate storage loot (%s): %w", cell.Prop, err)
		}
	}
	for range cell.ItemCount {
		if err := spawnFieldLoot(world, cell.ItemGroup, pos, lootRNG); err != nil {
			return fmt.Errorf("failed to place field loot (%s at %d,%d): %w", cell.ItemGroup, pos.X, pos.Y, err)
		}
	}
	if cell.Being {
		if err := spawnFacilityBeing(world, fac, danger, pos, beingRNG); err != nil {
			return fmt.Errorf("failed to place being of facility %q at %d,%d: %w", fac, pos.X, pos.Y, err)
		}
	}
	return nil
}

// cellPoses は施設の絵の、向きを決める置物 facing のマスごとの置き方 Pose を返す。通行を塞ぐ置物と facing の置物を塞がりに数え、
// 並びごとに向きを決める。絵の外は塞がりに数えない
func cellPoses(cells [][]maptemplate.MapCell, wallAt func(x, y int) bool, propOf func(id string) (rawschema.Prop, error)) (map[consts.Coord[consts.Tile]]*gc.Pose, error) {
	facers := map[consts.Coord[consts.Tile]]bool{}
	blockers := map[consts.Coord[consts.Tile]]bool{}
	for y, row := range cells {
		for x, cell := range row {
			if cell.Prop == "" {
				continue
			}
			prop, err := propOf(cell.Prop)
			if err != nil {
				return nil, fmt.Errorf("prop at %d,%d: %w", x, y, err)
			}
			c := consts.Coord[consts.Tile]{X: consts.Tile(x), Y: consts.Tile(y)}
			facing := prop.SpriteRender.Shape != nil && *prop.SpriteRender.Shape == rawschema.Facing
			if facing {
				facers[c] = true
			}
			blockers[c] = blockers[c] || facing || prop.BlockPass
		}
	}
	facings := map[consts.Coord[consts.Tile]]*gc.Pose{}
	wall := func(c consts.Coord[consts.Tile]) bool { return wallAt(int(c.X), int(c.Y)) }
	for c, f := range mapplanner.RowPoses(facers, blockers, wall) {
		facings[c] = &f
	}
	return facings, nil
}

// voidEnclosedTall は、直交と斜めの近傍8マスに通れる地形が無い高さのある地形を虚空にする。壁のオートタイルは部屋を囲む細い輪郭を
// 前提にするので、厚い壁の塊の内側を虚空にして描き崩れを防ぐ。マスの外は outsidePassable が通れると返す座標だけを通れるものとして
// 数える。家具の乗った壁は残す
func voidEnclosedTall(cells [][]maptemplate.MapCell, blocks, isTall func(terrain string) bool, outsidePassable func(x, y int) bool) {
	passable := func(x, y int) bool {
		if y < 0 || y >= len(cells) || x < 0 || x >= len(cells[y]) {
			return outsidePassable(x, y)
		}
		return !blocks(cells[y][x].Terrain)
	}
	var enclosed [][2]int
	for y, row := range cells {
		for x, c := range row {
			if !isTall(c.Terrain) || c.Prop != "" {
				continue
			}
			open := false
			for dy := -1; dy <= 1 && !open; dy++ {
				for dx := -1; dx <= 1 && !open; dx++ {
					open = (dx != 0 || dy != 0) && passable(x+dx, y+dy)
				}
			}
			if !open {
				enclosed = append(enclosed, [2]int{x, y})
			}
		}
	}
	for _, p := range enclosed {
		cells[p[1]][p[0]] = maptemplate.MapCell{Terrain: consts.TileNameVoid}
	}
}

// tileIsTall は地形 terrain が高さのあるタイルかを raw のタイルの行から返す
func tileIsTall(raws rawschema.Raws) func(terrain string) bool {
	tall := map[string]bool{}
	tiles := raw.PtrSlice(raws.Tiles)
	for i := range tiles {
		tall[tiles[i].Id] = raw.TileIsTall(tiles[i])
	}
	return func(terrain string) bool { return tall[terrain] }
}

// tileBlocksPass は地形 terrain が通れないかを raw のタイルの行から返す。行の無い地形は通れないとする
func tileBlocksPass(raws rawschema.Raws) func(terrain string) bool {
	blocks := map[string]bool{}
	tiles := raw.PtrSlice(raws.Tiles)
	for i := range tiles {
		blocks[tiles[i].Id] = tiles[i].BlockPass
	}
	return func(terrain string) bool {
		if terrain == maptemplate.GroundTerrain {
			return false
		}
		b, ok := blocks[terrain]
		return !ok || b
	}
}
