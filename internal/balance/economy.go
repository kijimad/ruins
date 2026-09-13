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

// ExpectedNetLootValue は危険度 danger で拾える1個あたりの、競売に出して手元に残る期待手取り額を返す。
// 手取りは落札額と重量の線形式なので、期待価値と期待重量を query.AuctionNetProceeds に入れれば
// 期待手取りになる。額面の ExpectedLootValue に対し、手数料と発送料を引いた実収入側の終端指標。
// 集荷料は集荷1回ごとで品単位でないためここには含めない。
//
// 探索1回の総収支は、これに拾える個数と移動燃料コストを掛け合わせて出るが、1回の移動タイル数は
// コードに定数が無い設計値なので閉形式では導けない。総収支はモンテカルロと設計値の領域に残す。
func ExpectedNetLootValue(master oapi.Raws, itemTableName string, danger int) float64 {
	value := ExpectedLootValue(master, itemTableName, danger)
	if value <= 0 {
		return 0
	}
	weightKg := ExpectedLootWeightKg(master, itemTableName, danger)
	net := query.AuctionNetProceeds(consts.Currency(math.Round(value)), weightKg)
	return float64(net)
}

// AuctionTakeHomeRate は基準価値どおりに落札されたときの競売の手取り率を返す。
// 手取り = 落札額 − 手数料 − 発送料を落札額で割る。集荷料は集荷1回ごとで品単位でないためここには含めない。
// 重い安物ほど発送料が手取りを食い、率が下がる。query.AuctionNetProceeds を単一出典で参照する。
//
// 落札額の分散や入札の伸びは確率過程で、実際の落札額分布はモンテカルロで測る領域。ここは基準価値で
// 売れた場合の決定論的な手取り率だけを見る。
func AuctionTakeHomeRate(saleValue consts.Currency, weightKg float64) float64 {
	if saleValue <= 0 {
		return 0
	}
	net := query.AuctionNetProceeds(saleValue, weightKg)
	return float64(net) / float64(saleValue)
}
