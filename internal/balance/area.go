package balance

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// nightTurns は夜と深夜の2つの時間帯を合わせた、1晩のターン数
const nightTurns = float64(gc.TurnsPerDay) * 2 / 6

// AreaSupply は拠点を置いたエリアの収納と地面にある、食べ物の栄養と燃やせる物の熱の合計
type AreaSupply struct {
	Nutrition int
	Heat      consts.Heat
}

// MeasureAreaSupply は world にある全てのアイテムの栄養と熱を足す。world には測るエリアだけを生成しておく
func MeasureAreaSupply(world w.World) AreaSupply {
	var s AreaSupply
	q := ecs.NewFilter1[gc.Item](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.ProvidesNutrition.Has(e) {
			s.Nutrition += world.Components.ProvidesNutrition.Get(e).Amount
		}
		s.Heat += query.HeatContent(world, e)
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
