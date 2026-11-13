package overworld

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// facilityLot は施設の絵の1辺の長さ。チャンク 24×24 から北と西の街路を引いた区画の大きさ
const facilityLot consts.Tile = 20

// errFacilityLotSizeMismatch は施設の絵の大きさが区画と一致しないときに返す
var errFacilityLotSizeMismatch = errors.New("facility template size does not match the lot")

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
// 屋内の床は屋内として扱い、扉は左右か上下の壁から向きを決める。収納の中身と敵は今の施設と同じ処理で引く。
// 壁判定の関数と占有したマスを返し、後段の街路の敵が建物の上に湧かないようにする
func spawnFacility(world w.World, g chunkGeom, cells [][]maptemplate.MapCell, ox, oy consts.Tile, fac string, danger consts.Danger, seed uint64) (func(lx, ly consts.Tile) bool, map[consts.Coord[consts.Tile]]bool, error) {
	if len(cells) != int(g.chunkH-oy) || len(cells[0]) != int(g.chunkW-ox) {
		return nil, nil, fmt.Errorf("facility %q %dx%d, lot %dx%d: %w", fac, len(cells[0]), len(cells), g.chunkW-ox, g.chunkH-oy, errFacilityLotSizeMismatch)
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
	wallAt := func(x, y int) bool {
		if y < 0 || y >= len(cells) || x < 0 || x >= len(cells[y]) {
			return false
		}
		t, err := tileOf(cells[y][x].Terrain)
		return err == nil && t.BlockPass
	}

	indoor := enclosedCells(cells, wallAt)
	tiles := g.tiles.get()
	occupied := make(map[consts.Coord[consts.Tile]]bool)
	lootRNG := rand.New(rand.NewPCG(seed, 0x4))
	beingRNG := rand.New(rand.NewPCG(seed, 0x5))
	for y, row := range cells {
		for x, cell := range row {
			pos := consts.Coord[consts.Tile]{X: g.offsetX + ox + consts.Tile(x), Y: g.offsetY + oy + consts.Tile(y)}
			occupied[pos] = true

			t, err := tileOf(cell.Terrain)
			if err != nil {
				return nil, nil, fmt.Errorf("facility %q tile at %d,%d: %w", fac, x, y, err)
			}
			if err := spawnFacilityTile(world, tiles, pos, t, indoor[[2]int{x, y}]); err != nil {
				return nil, nil, err
			}
			orientation := gc.DoorOrientationHorizontal
			if wallAt(x-1, y) && wallAt(x+1, y) {
				orientation = gc.DoorOrientationVertical
			}
			if err := spawnFacilityCell(world, cell, pos, orientation, fac, danger, lootRNG, beingRNG); err != nil {
				return nil, nil, err
			}
		}
	}

	isWall := func(lx, ly consts.Tile) bool { return wallAt(int(lx-ox), int(ly-oy)) }
	return isWall, occupied, nil
}

// enclosedCells は区画の縁からたどれない、壁と扉で囲まれたマスを屋内として返す。塀は家具なので囲いに数えず、塀の内の庭は屋外のままにする
func enclosedCells(cells [][]maptemplate.MapCell, wallAt func(x, y int) bool) map[[2]int]bool {
	h, w := len(cells), len(cells[0])
	blocks := func(x, y int) bool { return wallAt(x, y) || cells[y][x].Prop == "door" }
	outside := map[[2]int]bool{}
	var queue [][2]int
	for y := range h {
		for x := range w {
			if (x == 0 || y == 0 || x == w-1 || y == h-1) && !blocks(x, y) {
				outside[[2]int{x, y}] = true
				queue = append(queue, [2]int{x, y})
			}
		}
	}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if n[0] < 0 || n[1] < 0 || n[0] >= w || n[1] >= h || outside[n] || blocks(n[0], n[1]) {
				continue
			}
			outside[n] = true
			queue = append(queue, n)
		}
	}
	indoor := map[[2]int]bool{}
	for y := range h {
		for x := range w {
			if !outside[[2]int{x, y}] && !blocks(x, y) {
				indoor[[2]int{x, y}] = true
			}
		}
	}
	return indoor
}

// spawnFacilityTile はタイルの行 t を pos へ置く。行の生成の仕様が別のタイルを指せばそれを置き、オートタイルしないタイルは添字を付けない
func spawnFacilityTile(world w.World, tiles map[gc.GridElement]ecs.Entity, pos consts.Coord[consts.Tile], t rawschema.Tile, indoor bool) error {
	name := t.Id
	if t.Spawn != nil && t.Spawn.Tile != nil {
		name = *t.Spawn.Tile
	}
	var index *int
	if t.Spawn != nil && t.Spawn.Autotile {
		index = new(0)
	}
	e, err := replaceTileIndexed(world, tiles, pos, name, index)
	if err != nil {
		return fmt.Errorf("failed to place facility tile (x=%d, y=%d): %w", pos.X, pos.Y, err)
	}
	if indoor && world.Components.TileEnvironment.Has(e) {
		world.Components.TileEnvironment.Get(e).Shelter = rawschema.Indoor
	}
	return nil
}

// spawnFacilityCell はマスの扉、家具と収納の中身、床の戦利品、敵を pos へ置く。扉の向きは呼び出し側が壁から決める
func spawnFacilityCell(world w.World, cell maptemplate.MapCell, pos consts.Coord[consts.Tile], orientation gc.DoorOrientation, fac string, danger consts.Danger, lootRNG, beingRNG *rand.Rand) error {
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
