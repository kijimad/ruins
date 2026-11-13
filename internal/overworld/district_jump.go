package overworld

import (
	"errors"
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/rawschema"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
)

var (
	// errDistrictNotFound は raw に無い区域の種類を指定したときに返す
	errDistrictNotFound = errors.New("district not found")
	// errNoDistrictNorth は北に区域が見つからないときに返す
	errNoDistrictNorth = errors.New("no district to the north")
)

// districtJumpSearch は北の区域を探すリージョンの数の上限
const districtJumpSearch = 100

// JumpToDistrict はプレイヤーの北で一番近い種類 id の区域まで帯を進め、プレイヤーを区域の南の端の
// 真ん中の列のすぐ外へ、キューブをその隣へ置く。デバッグで区域を次々に確かめるため。
// 帯は1行ずつ北へシフトし、シフトのたびにプレイヤーとキューブを1行北へ動かして中央の行に留める。
// シフトは南端の行を捨てるので、キューブを連れていかないと失う
func (dr *Driver) JumpToDistrict(world w.World, id string) error {
	raws := world.Resources.RawMaster
	var target *district
	for _, d := range districtsFrom(raws) {
		if d.row.Id == id {
			target = &d
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: %q", errDistrictNotFound, id)
	}

	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	sb := query.GetSeamlessBand(world)
	chunkH := sb.ChunkH
	playerRow := sb.AbsChunkRow(world.Components.GridElement.Get(player).Y)

	r, ok := nearestDistrictNorth(raws, *target, sb.RunSeed, playerRow, sb.Cols)
	if !ok {
		return fmt.Errorf("%w: %q", errNoDistrictNorth, id)
	}

	for range playerRow - (r.anchor.Y + r.h) {
		world.Components.GridElement.Get(player).Y -= chunkH
		cubes := query.ActiveFilter2[gc.GridElement, gc.Cube](world).Query()
		for cubes.Next() {
			world.Components.GridElement.Get(cubes.Entity()).Y -= chunkH
		}
		if err := dr.band.ShiftNorth(world, dr.gen); err != nil {
			return err
		}
	}
	sb.NorthIndex = dr.band.NorthIndex()
	query.InvalidateSpatialIndex(world)

	rowTop := world.Components.GridElement.Get(player).Y / chunkH * chunkH
	gate := consts.Coord[consts.Tile]{X: (r.anchor.X + r.w/2).Tiles(sb.ChunkW) + urbanStreetW/2, Y: rowTop}
	spawn := walkableSpawnNear(world, gate)
	if err := lifecycle.MovePlayerToPosition(world, spawn); err != nil {
		return err
	}
	cubes := query.ActiveFilter2[gc.GridElement, gc.Cube](world).Query()
	for cubes.Next() {
		world.Components.GridElement.Get(cubes.Entity()).Coord = walkableSpawnNear(world, spawn.Add(consts.Coord[consts.Tile]{X: 2}))
	}
	query.InvalidateSpatialIndex(world)
	query.GetVisionState(world).RequestUpdate()
	return nil
}

// nearestDistrictNorth は絶対チャンク行 playerRow より北にある種類 d の区域のうち、一番近いものを返す。
// 種類の違う区域との重なりで出ない区域は飛ばす
func nearestDistrictNorth(raws rawschema.Raws, d district, runSeed uint64, playerRow, cols consts.Chunk) (districtRegion, bool) {
	ry := consts.FloorDiv(playerRow, consts.Chunk(d.row.Spacing))
	for range districtJumpSearch {
		r := d.regionIn(runSeed, ry, cols)
		ry--
		if r.anchor.Y+r.h >= playerRow {
			continue
		}
		if got, ok := districtAt(raws, runSeed, r.anchor, cols); ok && got == r {
			return r, true
		}
	}
	return districtRegion{}, false
}
