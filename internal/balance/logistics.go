package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/world/query"
)

// FollowRangeTiles は燃料ポイント fuel と積載重量 load でキューブが随伴できる歩数を返す。query.FollowFuelCost を
// そのまま使い燃費式に自動追従する。積荷が重いほど1歩のコストが上がり航続が縮む重量↔移動のトレード。
func FollowRangeTiles(fuel consts.Heat, load consts.Milligram) float64 {
	cost := query.FollowFuelCost(load)
	if cost <= 0 {
		return 0
	}
	return float64(fuel) / float64(cost)
}

// FuelBurnTurns は material を weightKg だけ地面直の火にくべたとき増える燃焼ターン数を返す。熱量を地面の燃焼効率で
// 割り引く。MaterialDefs.HeatOf と consts.Heat.BurnTurns に自動追従する。移動燃料と同じ熱量が火では燃焼時間へ変わる。
func FuelBurnTurns(defs gc.MaterialDefs, material oapi.Material, weightKg int) int {
	weight := consts.Milligram(weightKg) * consts.MilligramPerKg
	return int(defs.HeatOf(material, weight).BurnTurns(query.GroundBurnEfficiency))
}
