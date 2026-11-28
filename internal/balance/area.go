package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// nightTurns は夜から深夜までの時間帯を合わせた、1晩のターン数
const nightTurns = float64(gc.TimeMidnight-gc.TimeNight+1) * float64(gc.TurnsPerTimeOfDay)

// stayDays は1つのエリアに留まる日数の目安。鮮度の1段階がこれより長い食べ物は、留まる間は新鮮なまま食べられる
const stayDays = 7

// AreaSupply は拠点を置いたエリアの収納と地面にある、食べ物の栄養と燃やせる物の熱の合計
type AreaSupply struct {
	// Nutrition は全ての食べ物の栄養
	Nutrition int
	// KeepingNutrition は鮮度の1段階が stayDays 日以上ある食べ物の栄養
	KeepingNutrition int
	// Heat は食べ物でない燃やせる物の熱。食べた物は燃やせないので、食べ物は数えない
	Heat consts.Heat
}

// MeasureAreaSupply は world にある全てのアイテムの栄養と熱を足す。world には測るエリアだけを生成しておく
func MeasureAreaSupply(world w.World) AreaSupply {
	var s AreaSupply
	q := ecs.NewFilter1[gc.Item](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !world.Components.ProvidesNutrition.Has(e) {
			s.Heat += query.HeatContent(world, e)
			continue
		}
		amount := world.Components.ProvidesNutrition.Get(e).Amount
		s.Nutrition += amount
		if !world.Components.Perishable.Has(e) || world.Components.Perishable.Get(e).StageLength >= stayDays*gc.TurnsPerDay {
			s.KeepingNutrition += amount
		}
	}
	return s
}

// FoodDays は栄養 nutrition で、1日に減る満腹度を何日まかなえるかを返す
func FoodDays(p Params, nutrition int) float64 {
	return float64(nutrition) / (p.TurnsPerDay * hungerDrainPerTurn(p))
}

// FuelNights は熱 heat で、地面に直に起こした火を何晩絶やさずにいられるかを返す
func FuelNights(heat consts.Heat) float64 {
	return float64(heat.BurnTurns(query.GroundBurnEfficiency)) / nightTurns
}

// AreaSupplySample は1つの種と北の深さで測ったエリアの量
type AreaSupplySample struct {
	Seed   uint64
	Depth  consts.Chunk
	Supply AreaSupply
}
