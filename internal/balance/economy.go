package balance

import (
	"math"

	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/oapi"
	"github.com/kijimaD/ruins/internal/raw"
	"github.com/kijimaD/ruins/internal/world/query"
)

// ExpectedLootValue は危険度 danger で itemTable から拾える1個あたりの期待売買価値を返す。
// itemTable の該当帯エントリを重みで期待し、各エントリが指すアイテムグループの中身をさらに重みで
// 期待する2段の重み付き平均。収入側の指標で、探索1回の収支の loot 項に当たる。
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

// itemWeightKg はアイテムの重量を kg で返す。重量未設定やパース不能は0とみなす。
func itemWeightKg(it oapi.Item) float64 {
	if it.Weight == nil {
		return 0
	}
	mg, err := consts.ParseWeight(*it.Weight)
	if err != nil {
		return 0
	}
	return float64(mg) / float64(consts.MilligramPerKg)
}

// expectedGroupWeightKg はアイテムグループの中身の重み付き期待重量を kg で返す。グループでなく
// アイテムを直接指す場合はその重量を返す。ExpectedLootValue の価値版と同じ2段重み付け。
func expectedGroupWeightKg(master oapi.Raws, id string) float64 {
	g, err := raw.GetItemGroup(master, id)
	if err != nil {
		if it, e := raw.FindItem(master, id); e == nil {
			return itemWeightKg(it)
		}
		return 0
	}
	var wSum, kgSum float64
	for _, e := range g.Entries {
		it, err := raw.FindItem(master, e.Id)
		if err != nil {
			continue
		}
		wSum += e.Weight
		kgSum += e.Weight * itemWeightKg(it)
	}
	if wSum == 0 {
		return 0
	}
	return kgSum / wSum
}

// ExpectedLootWeightKg は危険度 danger で itemTable から拾える1個あたりの期待重量を kg で返す。
// ExpectedLootValue と同じ2段の重み付き平均で、手取りの発送料項に使う。
func ExpectedLootWeightKg(master oapi.Raws, itemTableName string, danger int) float64 {
	table, err := raw.GetItemTable(master, itemTableName)
	if err != nil {
		return 0
	}
	var wSum, kgSum float64
	for _, e := range table.Entries {
		if danger < e.MinDanger || danger > e.MaxDanger {
			continue
		}
		wSum += e.Weight
		kgSum += e.Weight * expectedGroupWeightKg(master, e.Id)
	}
	if wSum == 0 {
		return 0
	}
	return kgSum / wSum
}

// ExpectedNetLootValue は危険度 danger で拾える1個あたりの競売の期待手取り額を返す。額面の
// ExpectedLootValue から手数料と発送料を引いた実収入側の終端指標。集荷料は品単位でないため含めない。
func ExpectedNetLootValue(master oapi.Raws, itemTableName string, danger int) float64 {
	value := ExpectedLootValue(master, itemTableName, danger)
	if value <= 0 {
		return 0
	}
	weightKg := ExpectedLootWeightKg(master, itemTableName, danger)
	// AuctionNetProceeds は Currency 整数を取るので期待価値を1個ぶんだけ丸める。丸めは1品あたり0.5未満で、
	// 実ゲームの落札額も整数なので意図的。ExpectedRunLootIncome の積算でも増幅は個数×0.5未満に収まる。
	net := query.AuctionNetProceeds(consts.Currency(math.Round(value)), weightKg)
	return float64(net)
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

// EconomyProgression は経過日 1..days の経済を、危険度に応じた loot 手取りと生活費で表す。難易度と同じく
// 危険度で進行するので、進むほど loot 価値が上がり生活費が相対的に軽くなるかを見る。生活費は日に依らず
// 一定なので、変化するのは loot 側。乱数を使わない。
func EconomyProgression(master oapi.Raws, itemTableName string, days int) []EconomyDay {
	costPerDay := CostOfLivingPerDay(master, DefaultParams())
	out := make([]EconomyDay, 0, days)
	for day := 1; day <= days; day++ {
		danger := query.DangerLevelForDay(day)
		net := ExpectedNetLootValue(master, itemTableName, danger)
		perFood := 0.0
		if net > 0 {
			perFood = costPerDay / net
		}
		out = append(out, EconomyDay{Day: day, Danger: danger, NetLootValue: net, LootPerDayFood: perFood})
	}
	return out
}

// AuctionTakeHomeRate は基準価値どおり落札されたときの手取り率を返す。手取り=落札額−手数料−発送料 を
// 落札額で割る。重い安物ほど発送料に食われ率が下がる。落札額分布はモンテカルロの領域で、ここは決定論。
func AuctionTakeHomeRate(saleValue consts.Currency, weightKg float64) float64 {
	if saleValue <= 0 {
		return 0
	}
	net := query.AuctionNetProceeds(saleValue, weightKg)
	return float64(net) / float64(saleValue)
}
