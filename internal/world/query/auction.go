package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 通信販売デモの価格モデル。数値は検証用の暫定で、設計 doc の確定値ではない。
const (
	auctionShipRatePerKg = 25.0 // 発送料は重量に比例する
	auctionFeeRate       = 0.12 // 手数料は落札額に比例する
	auctionValueMult     = 1.0  // 基準価値から落札額への基準倍率

	// AuctionTurns は1回の入札が決着するまでのターン数。ターン経過で落札判定する。
	AuctionTurns = 3
)

// AuctionBid は落札額を計算する。基準価値に、売るまで分からない分散を掛ける。
func AuctionBid(world w.World, item ecs.Entity) int {
	base := GetItemValue(world, item)
	variance := 0.6 + world.Config.RNG.Float64()*0.8
	return int(float64(base) * auctionValueMult * variance)
}

// AuctionShippingCost は発送料を返す。重量に比例するので、重い安物は手取りを食う。
func AuctionShippingCost(world w.World, item ecs.Entity) int {
	weightKg := float64(GetEntityWeight(world, item)) / float64(consts.MilligramPerKg)
	return int(weightKg * auctionShipRatePerKg)
}

// AuctionFee は手数料を返す。落札額に比例する。
func AuctionFee(bid int) int {
	return int(float64(bid) * auctionFeeRate)
}

// AuctionSaleChance は入札決着時に落札される確率を返す。落札されるかはパラメータ次第で、
// 高価な品ほど買い手が付きにくく落札されずに在庫として残りやすい。数値は検証用の暫定。
func AuctionSaleChance(world w.World, item ecs.Entity) float64 {
	base := GetItemValue(world, item)
	chance := 0.9 - float64(base)/1200.0
	if chance < 0.25 {
		chance = 0.25
	}
	if chance > 0.9 {
		chance = 0.9
	}
	return chance
}

// GetAuctionHistory は出荷実績の履歴シングルトンを取得する。無ければ作る。
func GetAuctionHistory(world w.World) *gc.AuctionHistory {
	q := ecs.NewFilter1[gc.AuctionHistory](world.ECS).Query()
	if q.Next() {
		h := world.Components.AuctionHistory.Get(q.Entity())
		q.Close()
		return h
	}
	e := world.ECS.NewEntity()
	world.Components.AuctionHistory.Add(e, &gc.AuctionHistory{})
	return world.Components.AuctionHistory.Get(e)
}
