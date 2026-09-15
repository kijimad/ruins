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

// FuelBurnTurns は material を weightKg だけ地面直の火にくべたとき増える燃焼ターン数を返す。
// 熱量を地面の燃焼効率で割り引く。query.HeatOf と consts.Heat.BurnTurns をそのまま使い、
// 熱量式や効率の変更に自動追従する。移動燃料と同じ熱量が、火では暖と燃焼時間へ変換される。
func FuelBurnTurns(material oapi.Material, weightKg int) int {
	weight := consts.Milligram(weightKg) * consts.MilligramPerKg
	return int(query.HeatOf(material, weight).BurnTurns(query.GroundBurnEfficiency))
}
