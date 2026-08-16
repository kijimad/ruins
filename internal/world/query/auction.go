package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 通信販売デモの価格モデル。数値は検証用の暫定で、設計 doc の確定値ではない。
const (
	auctionShipRatePerKg = 25.0 // 配送料は重量に比例する。品ごとにかかる
	auctionFeeRate       = 0.12 // 手数料は落札額に比例する。品ごとにかかる
	auctionOpeningMult   = 0.4  // 開始入札は基準価値のこの割合
	auctionRaiseMult     = 0.15 // 1回の入札の上げ幅は基準価値のこの割合

	// AuctionBidChance は毎ターン新たな入札が来る確率。入札が来る限り競売は延長し、
	// 来なかったターンに落札が確定する。
	AuctionBidChance = 0.6

	// AuctionPickupFee は集荷手数料。集荷1回につき定額でかかり、品ごとの配送料や手数料とは別立て。
	// 小分けに集荷するほどかさむので、1回にまとめるほど得になる。
	AuctionPickupFee = 100

	// AuctionStartingReputation は店の初期評判。出荷期限を破ると下がる。
	AuctionStartingReputation = 100
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
	world.Components.AuctionHistory.Add(e, &gc.AuctionHistory{Reputation: AuctionStartingReputation})
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

// CollectStagedItems は出荷場所の積荷をまとめて集荷する。集荷は所持金を動かさず、金銭タブへ明細を積む。
// 落札済みの品ごとに受取金の明細を、集荷1回につき集荷料金の請求の明細を発生させる。
// 落札されていない品は明細を生まず、ただ手放される。
// 受取金の額面は落札額から配送料と手数料を引いた手取り。集荷料金は集荷1回につき定額で別立て。
// だから小分けに集荷するほど集荷料金の請求がかさみ、1回にまとめるほど得になる。
// 集荷した総件数と、受取金の明細を発生させた件数を返す。
func CollectStagedItems(world w.World, station ecs.Entity) (collected, receipts int) {
	// GetStorageItems は確定したスライスを返すので、反復中に削除してよい
	items := GetStorageItems(world, station)
	if len(items) == 0 {
		return 0, 0
	}
	history := GetAuctionHistory(world)
	for _, item := range items {
		if world.Components.AuctionSold.Has(item) {
			sold := world.Components.AuctionSold.Get(item)
			bid := sold.Bid
			ship := AuctionShippingCost(world, item)
			fee := AuctionFee(bid)
			history.Entries = append(history.Entries, gc.AuctionEntry{
				Kind: gc.AuctionEntryReceipt, Number: sold.Number, Name: GetEntityName(item, world),
				Amount: bid - ship - fee, Bid: bid, Ship: ship, Fee: fee,
			})
			receipts++
		}
		// 落札済みでない品は明細を生まず、ただ手放される
		world.ECS.RemoveEntity(item)
		collected++
	}
	// 集荷料金の請求を1件立てる。品ごとの配送料や手数料とは別立て
	history.Entries = append(history.Entries, gc.AuctionEntry{
		Kind: gc.AuctionEntryInvoice, Name: pickupInvoiceName, Amount: AuctionPickupFee,
	})
	return collected, receipts
}

// pickupInvoiceName は集荷料金の請求の表示名。query.T の msgid でなく英語リテラルを保持し、表示側で訳す。
const pickupInvoiceName = "Pickup fee"

// SettleAuctionEntry は金銭タブの明細1件を精算する。受取金は所持金へ加え、請求は所持金から引く。
// 精算した受取金は出荷実績として履歴へ移す。精算した明細と、精算できたかを返す。
func SettleAuctionEntry(world w.World, player ecs.Entity, index, now int) (gc.AuctionEntry, bool) {
	history := GetAuctionHistory(world)
	if index < 0 || index >= len(history.Entries) {
		return gc.AuctionEntry{}, false
	}
	e := history.Entries[index]
	switch e.Kind {
	case gc.AuctionEntryReceipt:
		if err := AddCurrency(world, player, e.Amount); err != nil {
			return gc.AuctionEntry{}, false
		}
		history.Records = append(history.Records, gc.AuctionRecord{
			Number: e.Number, Name: e.Name, Bid: e.Bid, Ship: e.Ship, Fee: e.Fee, Net: e.Amount, Turn: now,
		})
	case gc.AuctionEntryInvoice:
		if err := AddCurrency(world, player, -e.Amount); err != nil {
			return gc.AuctionEntry{}, false
		}
	}
	history.Entries = append(history.Entries[:index], history.Entries[index+1:]...)
	return e, true
}
