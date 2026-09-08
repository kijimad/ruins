package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// DriveCubeTiles は運転できるキューブのタイル座標を集めて返す。マクロ地図のマーカー用。
// 位置は GridElement から常に導けるので保持せず、読み取り時に集める。
func DriveCubeTiles(world w.World) []consts.Coord[consts.Tile] {
	var out []consts.Coord[consts.Tile]
	q := ActiveFilter2[gc.GridElement, gc.Drivable](world).Query()
	for q.Next() {
		out = append(out, world.Components.GridElement.Get(q.Entity()).Coord)
	}
	return out
}

// PlayerBandTile はプレイヤーの帯ローカルなタイル座標を返す。プレイヤーが居なければ ok=false。
// マクロ地図のプレイヤーマーカーを置くのに使う。
func PlayerBandTile(world w.World) (consts.Coord[consts.Tile], bool) {
	player, err := GetPlayerEntity(world)
	if err != nil || !world.Components.GridElement.Has(player) {
		return consts.Coord[consts.Tile]{}, false
	}
	return world.Components.GridElement.Get(player).Coord, true
}

// CubeWeight はキューブ収納にある物の総重量を返す。運転1タイルの燃料コスト算出に使う。
// 収納の中身から常に導けるので値を保持せず、読み取り時に合算する。
func CubeWeight(world w.World, cube ecs.Entity) consts.Milligram {
	var total consts.Milligram
	for _, item := range GetStorageItems(world, cube) {
		total += GetEntityWeight(world, item)
	}
	return total
}

// DriveFuelCost は総重量から1タイル運転するのに要する燃料量を返す。空でも基準量がかかり、
// 総重量に比例して増える。重いほど燃費が悪化する。
func DriveFuelCost(total consts.Milligram) consts.Heat {
	kg := int(total / consts.MilligramPerKg)
	return consts.Heat(consts.DriveFuelBase + consts.DriveFuelPerKg*kg)
}

// CubeFuelTotal はキューブ収納にある物の燃焼熱量の総和を返す。運転の燃料源。
// 火への給油と同じ HeatContent を使い燃料値の定義を二重化しない。
func CubeFuelTotal(world w.World, cube ecs.Entity) consts.Heat {
	var total consts.Heat
	for _, item := range GetStorageItems(world, cube) {
		total += HeatContent(world, item)
	}
	return total
}
