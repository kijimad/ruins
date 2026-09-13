package balance

import (
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
)

// DriveRangeTiles は所持燃料 fuel と積載重量 load でキューブが運転できるタイル数を返す。
// query.DriveFuelCost をそのまま使い、燃費式の変更に自動追従する。積荷が重いほど1タイルの
// コストが上がり航続が縮む。これが重量↔移動のトレード。
func DriveRangeTiles(fuel consts.Heat, load consts.Milligram) float64 {
	cost := query.DriveFuelCost(load)
	if cost <= 0 {
		return 0
	}
	return float64(fuel) / float64(cost)
}

// DriveRangeAllFuel は容量いっぱいを material の燃料で満たしたときの航続タイルを返す。
// 燃料自身が積載重量になるので、積むほど燃費が悪化する自己ブレーキを含む。
func DriveRangeAllFuel(material oapi.Material, capacityKg int) float64 {
	weight := consts.Milligram(capacityKg) * consts.MilligramPerKg
	fuel := query.HeatOf(material, weight)
	return DriveRangeTiles(fuel, weight)
}
