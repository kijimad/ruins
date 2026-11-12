package overworld

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"slices"
	"sync"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/maptemplate"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

// facilityLot は施設の絵の1辺の長さ。チャンク 24×24 から北と西の街路を引いた区画の大きさ
const facilityLot consts.Tile = 20

// errFacilityLotSizeMismatch は施設の絵の大きさが区画と一致しないときに返す
var errFacilityLotSizeMismatch = errors.New("facility template size does not match the lot")

// facilityTemplates は施設の絵とパレットを1度だけ読み込んで返す。絵は資材に埋め込まれていて実行中に変わらない
var facilityTemplates = sync.OnceValues(func() (*maptemplate.TemplateLoader, error) {
	l := maptemplate.NewTemplateLoader()
	if err := l.RegisterAllChunks([]string{"levels/chunks", "levels/facilities", "levels/layouts"}); err != nil {
		return nil, err
	}
	if err := l.RegisterAllPalettes([]string{"levels/palettes"}); err != nil {
		return nil, err
	}
	return l, nil
})

// planFacility は施設 fac の絵を1枚選んで展開する。入口が西の街路に面する区画では左へ90度回し、左右の反転は rng で半々に選ぶ
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
			name := t.Id
			if t.Spawn != nil && t.Spawn.Tile != nil {
				name = *t.Spawn.Tile
			}
			e, err := replaceTile(world, tiles, pos, name)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to place facility tile (x=%d, y=%d): %w", pos.X, pos.Y, err)
			}
			if name == consts.TileNameFloor && world.Components.TileEnvironment.Has(e) {
				world.Components.TileEnvironment.Get(e).Shelter = rawschema.Indoor
			}

			switch {
			case cell.Prop == "door":
				orientation := gc.DoorOrientationHorizontal
				if wallAt(x-1, y) && wallAt(x+1, y) {
					orientation = gc.DoorOrientationVertical
				}
				if _, err := lifecycle.SpawnDoor(world, pos, orientation); err != nil {
					return nil, nil, fmt.Errorf("failed to place facility door at %d,%d: %w", pos.X, pos.Y, err)
				}
			case cell.Prop != "":
				if slices.ContainsFunc(query.GetEntitiesAt(world, pos.X, pos.Y), world.Components.Prop.Has) {
					continue
				}
				ent, err := lifecycle.SpawnProp(world, cell.Prop, pos.X, pos.Y)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to place facility prop (%s at %d,%d): %w", cell.Prop, pos.X, pos.Y, err)
				}
				if err := populateStorageLoot(world, ent, cell.Prop, cell.LootTable, fac, lootRNG); err != nil {
					return nil, nil, fmt.Errorf("failed to populate storage loot (%s): %w", cell.Prop, err)
				}
			}
			if cell.ItemGroup != "" {
				if err := spawnFieldLoot(world, cell.ItemGroup, pos, lootRNG); err != nil {
					return nil, nil, fmt.Errorf("failed to place field loot (%s at %d,%d): %w", cell.ItemGroup, pos.X, pos.Y, err)
				}
			}
			if cell.Being {
				if err := spawnInteriorBeing(world, fac, danger, pos, beingRNG); err != nil {
					return nil, nil, fmt.Errorf("failed to place being of facility %q at %d,%d: %w", fac, pos.X, pos.Y, err)
				}
			}
		}
	}

	isWall := func(lx, ly consts.Tile) bool { return wallAt(int(lx-ox), int(ly-oy)) }
	return isWall, occupied, nil
}
