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
	auctionSlowMult      = 1.25 // 粘り出品は高値が付く
	auctionFastMult      = 0.85 // 速い出品は安値に落ち着く

	// AuctionFastSteps と AuctionSlowSteps は出品から落札までのプレイヤーの移動数。換金速度と値段の綱引き。
	// ターン制の進行に依存せず移動そのものを単位にするので、どの場面でも歩けば必ず落札へ近づく。
	AuctionFastSteps = 1
	AuctionSlowSteps = 3
)

// AuctionBid は落札額を計算する。基準価値に出品速度の倍率と、売るまで分からない分散を掛ける。
func AuctionBid(world w.World, item ecs.Entity, slow bool) int {
	base := GetItemValue(world, item)
	mult := auctionFastMult
	if slow {
		mult = auctionSlowMult
	}
	variance := 0.6 + world.Config.RNG.Float64()*0.8
	return int(float64(base) * auctionValueMult * mult * variance)
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

// AuctionSaleChance は出品期間の満了時に落札される確率を返す。落札されるかはパラメータ次第で、
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

// PlayerHoldsListingTag はプレイヤーが出品タグを1枚でも持ち物に持つかを返す。
func PlayerHoldsListingTag(world w.World) bool {
	player, err := GetPlayerEntity(world)
	if err != nil {
		return false
	}
	q := ecs.NewFilter1[gc.AuctionTag](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.LocationInBackpack.Has(e) && world.Components.LocationInBackpack.Get(e).Owner == player {
			q.Close()
			return true
		}
	}
	return false
}

// GetAuctionClock は通信販売デモの進行状態シングルトンを取得する。無ければ作る。
func GetAuctionClock(world w.World) *gc.AuctionClock {
	q := ecs.NewFilter1[gc.AuctionClock](world.ECS).Query()
	if q.Next() {
		c := world.Components.AuctionClock.Get(q.Entity())
		q.Close()
		return c
	}
	e := world.ECS.NewEntity()
	world.Components.AuctionClock.Add(e, &gc.AuctionClock{NextTagID: 1})
	return world.Components.AuctionClock.Get(e)
}
