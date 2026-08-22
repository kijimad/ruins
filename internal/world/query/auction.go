package query

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/mlange-42/ark/ecs"
)

// 通信販売の価格モデル。数値は検証用の暫定で、設計 doc の確定値ではない。
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
	AuctionPickupFee consts.Currency = 100
)

// AuctionOpeningBid は開始入札額を返す。基準価値に分散を掛けた控えめな額から競売が始まる。
func AuctionOpeningBid(world w.World, item ecs.Entity) consts.Currency {
	base := GetItemValue(world, item)
	variance := 0.8 + world.Resources.Config.RNG.Float64()*0.4
	return consts.Currency(float64(base) * auctionOpeningMult * variance)
}

// AuctionRaise は1回の入札での上げ幅を返す。入札が来るたびこの額だけ現在値が上がる。
func AuctionRaise(world w.World, item ecs.Entity) consts.Currency {
	base := GetItemValue(world, item)
	variance := 0.8 + world.Resources.Config.RNG.Float64()*0.4
	raise := max(consts.Currency(float64(base)*auctionRaiseMult*variance), 1)
	return raise
}

// AuctionShippingCost は発送料を返す。重量に比例するので、重い安物は手取りを食う。
func AuctionShippingCost(world w.World, item ecs.Entity) consts.Currency {
	weightKg := float64(GetEntityWeight(world, item)) / float64(consts.MilligramPerKg)
	return consts.Currency(weightKg * auctionShipRatePerKg)
}

// AuctionFee は手数料を返す。落札額に比例する。
func AuctionFee(bid consts.Currency) consts.Currency {
	return consts.Currency(float64(bid) * auctionFeeRate)
}

// GetAuctionHistory は出荷実績の履歴シングルトンを取得する。
func GetAuctionHistory(world w.World) *gc.AuctionHistory {
	return GetSingleton[gc.AuctionHistory](world, world.Components.AuctionHistory)
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
// 受取金の額面は落札額から配送料と手数料を引いた手取り。集荷料金は集荷1回につき定額で別立て。
// だから小分けに集荷するほど集荷料金の請求がかさみ、1回にまとめるほど得になる。
// 集荷した総件数と、受取金の明細を発生させた件数を返す。
func CollectStagedItems(world w.World, station ecs.Entity) (collected, receipts int) {
	// GetStorageItems は確定したスライスを返すので、反復中に削除してよい
	items := GetStorageItems(world, station)
	if len(items) == 0 {
		return 0, 0
	}
	// 明細を一旦ためる。履歴シングルトンへの追記は品の削除が終わってから行う。
	// エンティティ削除で Get のポインタが無効化されうるので、構造変更を跨いで history を保持しない
	var entries []gc.AuctionEntry
	for _, item := range items {
		if world.Components.AuctionSold.Has(item) {
			sold := world.Components.AuctionSold.Get(item)
			bid := sold.Bid
			ship := AuctionShippingCost(world, item)
			fee := AuctionFee(bid)
			entries = append(entries, gc.AuctionEntry{
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
	entries = append(entries, gc.AuctionEntry{
		Kind: gc.AuctionEntryInvoice, Name: pickupInvoiceName, Amount: AuctionPickupFee,
	})
	history := GetAuctionHistory(world)
	history.Entries = append(history.Entries, entries...)
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
		// 売上統計: 受取金を run 統計へ加算する
		if s := GetRunStats(world); s != nil {
			s.SalesTotal += e.Amount
		}
	case gc.AuctionEntryInvoice:
		if err := AddCurrency(world, player, -e.Amount); err != nil {
			return gc.AuctionEntry{}, false
		}
	}
	history.Entries = append(history.Entries[:index], history.Entries[index+1:]...)
	return e, true
}
