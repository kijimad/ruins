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
	auctionOpeningMult   = 0.4  // 開始入札は基準価値のこの割合
	auctionRaiseMult     = 0.15 // 1回の入札の上げ幅は基準価値のこの割合

	// AuctionBidChance は毎ターン新たな入札が来る確率。入札が来る限り競売は延長し、
	// 来なかったターンに落札が確定する。
	AuctionBidChance = 0.6
)

// AuctionOpeningBid は開始入札額を返す。基準価値に分散を掛けた控えめな額から競売が始まる。
func AuctionOpeningBid(world w.World, item ecs.Entity) int {
	base := GetItemValue(world, item)
	variance := 0.8 + world.Config.RNG.Float64()*0.4
	return int(float64(base) * auctionOpeningMult * variance)
}

// AuctionRaise は1回の入札での上げ幅を返す。入札が来るたびこの額だけ現在値が上がる。
func AuctionRaise(world w.World, item ecs.Entity) int {
	base := GetItemValue(world, item)
	variance := 0.8 + world.Config.RNG.Float64()*0.4
	raise := max(int(float64(base)*auctionRaiseMult*variance), 1)
	return raise
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
