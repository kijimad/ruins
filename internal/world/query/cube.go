package query

import (
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

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
