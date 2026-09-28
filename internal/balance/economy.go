package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ExpectedLootValue は危険度 danger で itemTable から拾える1個あたりの期待売買価値を返す。該当帯エントリと
// その指すアイテムグループの中身を2段で重み付き平均する。収入側の指標。
func ExpectedLootValue(master oapi.Raws, itemTableName string, danger int) float64 {
	table, err := raw.GetItemTable(master, itemTableName)
	if err != nil {
		return 0
	}
	var wSum, vSum float64
	for _, e := range table.Entries {
		if danger < e.MinDanger || danger > e.MaxDanger {
			continue
		}
		wSum += e.Weight
		vSum += e.Weight * expectedGroupValue(master, e.Id)
	}
	if wSum == 0 {
		return 0
	}
	return vSum / wSum
}

// expectedGroupValue はアイテムグループの中身の重み付き期待売買価値を返す。グループでなく
// アイテムを直接指す場合はその価値を返す。
func expectedGroupValue(master oapi.Raws, id string) float64 {
	g, err := raw.GetItemGroup(master, id)
	if err != nil {
		if it, e := raw.FindItem(master, id); e == nil {
			return float64(it.Value)
		}
		return 0
	}
	var wSum, vSum float64
	for _, e := range g.Entries {
		it, err := raw.FindItem(master, e.Id)
		if err != nil {
			continue
		}
		wSum += e.Weight
		vSum += e.Weight * float64(it.Value)
	}
	if wSum == 0 {
		return 0
	}
	return vSum / wSum
}

// ExpectedNetLootValue は危険度 danger で拾える1個あたりの店売りの期待手取り額を返す。額面の
// ExpectedLootValue に売却倍率を掛けた実収入側の終端指標。
func ExpectedNetLootValue(master oapi.Raws, itemTableName string, danger int) float64 {
	value := ExpectedLootValue(master, itemTableName, danger)
	if value <= 0 {
		return 0
	}
	// CalculateSellPrice は Currency 整数を取るので期待価値を1個ぶんだけ丸める。丸めは1品あたり0.5未満。
	return float64(query.CalculateSellPrice(int(math.Round(value))))
}

// expectedItemsPerFloor はフロアで拾える loot 個数の期待値。floorItemBase + rand(0..floorItemRandom-1) の
// 期待値で run.go と単一出典。floorItemRandom は奇数なので (floorItemRandom-1)/2 は整数除算でも厳密。
const expectedItemsPerFloor = floorItemBase + (floorItemRandom-1)/2

// ExpectedRunLootIncome は floors 層を探索して拾う loot の期待手取り総額を返す。各層で expectedItemsPerFloor
// 個を拾い、層の深さを危険度として ExpectedNetLootValue を積む。到達層数は生存依存の scenario 入力。
func ExpectedRunLootIncome(master oapi.Raws, itemTableName string, floors int) float64 {
	total := 0.0
	for depth := 1; depth <= floors; depth++ {
		total += expectedItemsPerFloor * ExpectedNetLootValue(master, itemTableName, depth)
	}
	return total
}

// CheapestFoodCostPerNutrition は満腹度1点を回復するのに要する最小の購入価値を返す。栄養価を持つ
// アイテムのうち、価値÷栄養が最小のものを賢い調達とみなす。食料が無ければ0を返す。
func CheapestFoodCostPerNutrition(master oapi.Raws) float64 {
	best := 0.0
	items := raw.PtrSlice(master.Items)
	for i := range items {
		it := &items[i]
		if it.ProvidesNutrition == nil || *it.ProvidesNutrition <= 0 || it.Value <= 0 {
			continue
		}
		cost := float64(it.Value) / float64(*it.ProvidesNutrition)
		if best == 0 || cost < best {
			best = cost
		}
	}
	return best
}

// CostOfLivingPerDay は補給なしの1日で失う満腹度を、最安の食料で埋め戻す購入費用を返す。1日の満腹度
// 減耗は turnsPerDay/HungerDrainTurns で、これに満腹度1点あたりの最小食料費を掛ける。経済の支出側の下端。
func CostOfLivingPerDay(master oapi.Raws, p Params) float64 {
	hungerPerDay := p.TurnsPerDay / p.HungerDrainTurns
	return hungerPerDay * CheapestFoodCostPerNutrition(master)
}

// EconomyDay は経過日1日ぶんの経済の断面。危険度に応じた1個あたり手取りと、1日の食費を賄うのに要する
// loot 個数を持つ。個数が小さいほど、その日の loot は生活費に対して価値が高い。
type EconomyDay struct {
	Day            int
	Danger         int
	NetLootValue   float64 // 1個あたりの期待手取り
	LootPerDayFood float64 // 1日の食費を賄うのに要する loot 個数
}

// EconomyProgression は経過日 1..days の経済を、危険度に応じた loot 手取りと一定の生活費で表す。進むほど loot
// 価値が上がり生活費が相対的に軽くなるかを見る。乱数を使わない。
func EconomyProgression(master oapi.Raws, itemTableName string, days int, p Params) []EconomyDay {
	costPerDay := CostOfLivingPerDay(master, p)
	out := make([]EconomyDay, 0, days)
	for day := 1; day <= days; day++ {
		danger := int(query.DangerLevelForDay(day))
		net := ExpectedNetLootValue(master, itemTableName, danger)
		perFood := 0.0
		if net > 0 {
			perFood = costPerDay / net
		}
		out = append(out, EconomyDay{Day: day, Danger: danger, NetLootValue: net, LootPerDayFood: perFood})
	}
	return out
}
