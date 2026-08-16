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

// StartAuctionListing はタグを貼って出品を始める。連番を採番し開始入札で AuctionListing を付ける。
// 採番した番号を返す。以後この番号でその出品を指す。
func StartAuctionListing(world w.World, item ecs.Entity, now int) int {
	history := GetAuctionHistory(world)
	history.NextNumber++
	number := history.NextNumber
	world.Components.AuctionListing.Add(item, &gc.AuctionListing{
		Number:     number,
		CurrentBid: AuctionOpeningBid(world, item),
		LastTurn:   now,
	})
	return number
}

// ShipSoldItems は持ち物内の落札済みの品をまとめて出荷する。手取りを入金し履歴へ記録して品を消す。
// 出荷した件数と手取り合計を返す。
func ShipSoldItems(world w.World, player ecs.Entity, now int) (int, int) {
	// 反復中はワールドがロックされ削除できない。一旦集めてからループ外で処理する
	var items []ecs.Entity
	q := ecs.NewFilter2[gc.AuctionSold, gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		if world.Components.LocationInBackpack.Get(q.Entity()).Owner == player {
			items = append(items, q.Entity())
		}
	}

	history := GetAuctionHistory(world)
	count, total := 0, 0
	for _, item := range items {
		sold := world.Components.AuctionSold.Get(item)
		bid := sold.Bid
		ship := AuctionShippingCost(world, item)
		fee := AuctionFee(bid)
		net := bid - ship - fee
		if err := AddCurrency(world, player, net); err != nil {
			continue
		}
		history.Records = append(history.Records, gc.AuctionRecord{
			Number: sold.Number, Name: GetEntityName(item, world), Bid: bid, Ship: ship, Fee: fee, Net: net, Turn: now,
		})
		world.ECS.RemoveEntity(item)
		count++
		total += net
	}
	return count, total
}
