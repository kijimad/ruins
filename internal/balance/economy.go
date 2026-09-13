package balance

import (
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
