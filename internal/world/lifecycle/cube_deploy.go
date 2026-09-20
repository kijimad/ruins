package lifecycle

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// deployOffsets は展開時にキューブの周りで占有する相対タイル。今は四方の隣接に固定する。
var deployOffsets = []consts.Coord[consts.Tile]{
	{X: 0, Y: -1}, {X: 1, Y: 0}, {X: 0, Y: 1}, {X: -1, Y: 0},
}

// DeployCube はキューブを展開状態にする。必要タイルがすべて空いていれば Deployed を付けて true を返す。
// 1タイルでも塞がれていれば状態を変えず false を返す。展開は全か無かで部分展開はしない。
func DeployCube(world w.World, cube ecs.Entity) bool {
	if world.Components.Deployed.Has(cube) {
		return true
	}
	if !deploySpaceFree(world, cube) {
		return false
	}
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	return true
}

// StowCube はキューブを収納状態へ戻す。
func StowCube(world w.World, cube ecs.Entity) {
	if world.Components.Deployed.Has(cube) {
		world.Components.Deployed.Remove(cube)
	}
}

// deploySpaceFree は展開に要する全タイルが空いているかを返す。壁・不可通行の物・キャラクターがいれば偽。
func deploySpaceFree(world w.World, cube ecs.Entity) bool {
	si := query.GetSpatialIndex(world)
	if si == nil {
		return false
	}
	base := world.Components.GridElement.Get(cube).Coord
	for _, off := range deployOffsets {
		t := base.Add(off)
		if si.IsBlockPass(t) {
			return false
		}
		if _, ok := si.CharacterAt(t); ok {
			return false
		}
	}
	return true
}
