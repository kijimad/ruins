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

// deployTileSprite は展開タイルの見た目。資材の樽で野営の広がりを表す。
const deployTileSprite = "wood_barrel"

// DeployCube はキューブを展開状態にする。必要タイルがすべて空いていれば Deployed を付けて構造物タイルを
// spawn し true を返す。1タイルでも塞がれていれば状態を変えず false を返す。展開は全か無かで部分展開はしない。
func DeployCube(world w.World, cube ecs.Entity) bool {
	if world.Components.Deployed.Has(cube) {
		return true
	}
	if !deploySpaceFree(world, cube) {
		return false
	}
	world.Components.Deployed.Add(cube, &gc.Deployed{})
	spawnDeployedTiles(world, cube)
	return true
}

// StowCube はキューブを収納状態へ戻し、展開タイルを片付ける。
func StowCube(world w.World, cube ecs.Entity) {
	if !world.Components.Deployed.Has(cube) {
		return
	}
	world.Components.Deployed.Remove(cube)
	DespawnDeployedTiles(world)
}

// DespawnDeployedTiles は展開タイルをすべて消す。収納時とロード後の掃除で使う。
func DespawnDeployedTiles(world w.World) {
	q := ecs.NewFilter1[gc.DeployedTile](world.ECS).Query()
	var tiles []ecs.Entity
	for q.Next() {
		tiles = append(tiles, q.Entity())
	}
	for _, e := range tiles {
		world.ECS.RemoveEntity(e)
	}
}

// spawnDeployedTiles は展開タイルをキューブの周りへ出す。キューブと同じステージ・オーバーワールドに置く。
func spawnDeployedTiles(world w.World, cube ecs.Entity) {
	base := world.Components.GridElement.Get(cube).Coord
	stage := *world.Components.StageBound.Get(cube)
	for _, off := range deployOffsets {
		world.Components.AddEntity(world.ECS, &gc.EntitySpec{
			GridElement:     &gc.GridElement{Coord: base.Add(off)},
			SpriteRender:    &gc.SpriteRender{SpriteSheetName: fieldSpriteSheet, SpriteKey: deployTileSprite, Depth: gc.DepthNumTaller},
			DeployedTile:    &gc.DeployedTile{},
			LocationOnField: &gc.LocationOnField{},
			StageBound:      &stage,
		})
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
