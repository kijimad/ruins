package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// shippingStationRawID は出荷場所の raw prop の id。テンプレートで配置されたこの prop を
// システムが見つけて出荷場所に仕立てる。
const shippingStationRawID = "shipping_station"

// auctionShipDelay は集荷タイマーの長さ。積荷が入るとこのターン数だけ待ってから集荷する。
const auctionShipDelay = 10

// auctionShipDeadline は落札から出荷までの猶予ターン数。この間に積荷へ渡さないと評判が下がる。
// 出先で乱出品すると帰ってくる前に期限が切れるので、遠出しての出品を抑える。
const auctionShipDeadline = 20

// auctionReputationPenalty は出荷期限を1件破るごとに下がる評判。
const auctionReputationPenalty = 10

// AuctionSystem はタグを貼って出品された品の競売をターン経過で進める。
// 入札が来る限り現在値を上げて延長し、入札が止まったターンに現在値で落札を確定する。
// 落札しても入金はせず、品を落札済みへ移して出荷場所での出荷を待たせる。
// 出品中の品が無い通常プレイでは即座に何もしない。
type AuctionSystem struct{}

// String はシステム名を返す
func (sys AuctionSystem) String() string {
	return "AuctionSystem"
}

// Update は出品中の品をターン経過で競売する
func (sys *AuctionSystem) Update(world w.World) error {
	markShippingStations(world)

	now := int(query.GetGameTime(world).TotalTurns)

	// 反復中はワールドがロックされ構造変更できない。出品中の品を一旦集めてからループ外で処理する
	var items []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for q.Next() {
		items = append(items, q.Entity())
	}
	for _, item := range items {
		processAuctionItem(world, item, now)
	}

	updateShippingTimers(world, now)
	penalizeOverdueSold(world, now)
	return nil
}

// penalizeOverdueSold は出荷期限を過ぎても持ち物に残る落札済みの品を罰する。
// 期限までに積荷へ渡さなかった品ごとに店の評判を下げ、二重に罰さないよう印を付ける。
// 積荷へ渡した品は運送に託したとみなし罰しない。だから出先で乱出品すると帰りが間に合わず評判を失う。
func penalizeOverdueSold(world w.World, now int) {
	var overdue []ecs.Entity
	q := ecs.NewFilter2[gc.AuctionSold, gc.LocationInBackpack](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		s := world.Components.AuctionSold.Get(e)
		// 積荷へ入れた品はステーションの収納にあり持ち物クエリに掛からない。
		// 持ち物に抱えたまま期限を過ぎた品だけを罰する
		if !s.Penalized && now > s.DueTurn {
			overdue = append(overdue, e)
		}
	}
	if len(overdue) == 0 {
		return
	}
	history := query.GetAuctionHistory(world)
	for _, item := range overdue {
		world.Components.AuctionSold.Get(item).Penalized = true
		history.Reputation -= auctionReputationPenalty
		logDeadlineMissed(world, query.GetEntityName(item, world), history.Reputation)
	}
}

func logDeadlineMissed(world w.World, name string, reputation int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s missed the shipping deadline. Reputation is now %d.", gamelog.Tag("item", name), reputation)).
		Log()
}

// updateShippingTimers は各出荷場所の集荷タイマーを進める。荷物はステーションごとなので
// タイマーもステーションごとに持つ。積荷が入るとタイマーを開始し、満了したターンにその
// ステーションの積荷をまとめて集荷する。積荷が空になったらタイマーを止める。
func updateShippingTimers(world w.World, now int) {
	var stations []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionStation](world.ECS).Query()
	for q.Next() {
		stations = append(stations, q.Entity())
	}
	for _, station := range stations {
		staged := len(query.GetStorageItems(world, station)) > 0
		s := world.Components.AuctionStation.Get(station)
		switch {
		case !staged:
			s.ShipAtTurn = 0 // 積荷が無いのでタイマー停止
		case s.ShipAtTurn == 0:
			s.ShipAtTurn = now + auctionShipDelay // 積荷が入ったのでタイマー開始
			logShipScheduled(world, auctionShipDelay)
		case now >= s.ShipAtTurn:
			// 満了。集荷する。CollectStagedItems が構造変更する前にタイマーを止める
			s.ShipAtTurn = 0
			collected, receipts := query.CollectStagedItems(world, station)
			if collected > 0 {
				logCollected(world, collected, receipts)
			}
		}
	}
}

func logShipScheduled(world w.World, turns int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Shipment scheduled. Pickup in %d turns.", turns)).
		Log()
}

func logCollected(world w.World, collected, receipts int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Collected %d items. %d receipts and a pickup bill arrived in the finance tab.", collected, receipts)).
		Log()
}

// processAuctionItem は1品の競売を進める。1ターンに1回、入札が来れば現在値を上げて延長し、
// 来なければ落札を確定してその現在値で落札済みへ移す。入金は出荷場所での出荷時に行う。
func processAuctionItem(world w.World, item ecs.Entity, now int) {
	l := world.Components.AuctionListing.Get(item)
	if now == l.LastTurn {
		return // 同じターンでは1回だけ判定する。出品した当ターンも含む
	}
	l.LastTurn = now

	// 入札が来る限り延長する。来たら現在値を上げる
	if world.Config.RNG.Float64() < query.AuctionBidChance {
		l.CurrentBid += query.AuctionRaise(world, item)
		return
	}

	// 入札が止まった。落札を確定して落札済みへ移す。入金はまだしない。
	// 落札の時点から出荷期限が始まる。期限までに積荷へ渡さないと評判が下がる
	number := l.Number
	bid := l.CurrentBid
	name := query.GetEntityName(item, world)
	world.Components.AuctionListing.Remove(item)
	world.Components.AuctionSold.Add(item, &gc.AuctionSold{Number: number, Bid: bid, DueTurn: now + auctionShipDeadline})
	logAuctionWon(world, name, bid)
}

// markShippingStations はテンプレートで配置された shipping_station prop を出荷場所に仕立てる。
// 出荷と状況確認の相互作用を付け、識別マーカーを付ける。
// shipping_station 以外の prop には触れないので通常プレイに影響しない。
func markShippingStations(world w.World) {
	var toMark []ecs.Entity
	q := ecs.NewFilter1[gc.RawID](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.RawID.Get(e).ID != shippingStationRawID || world.Components.AuctionStation.Has(e) {
			continue
		}
		toMark = append(toMark, e)
	}
	for _, e := range toMark {
		world.Components.AuctionStation.Add(e, &gc.AuctionStation{})
		interactions := []gc.InteractionKind{gc.InteractionAuction}
		if world.Components.Interactable.Has(e) {
			world.Components.Interactable.Get(e).Interactions = interactions
		} else {
			world.Components.Interactable.Add(e, &gc.Interactable{Interactions: interactions})
		}
	}
}

func logAuctionWon(world w.World, name string, bid int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s was won. Bid %s. Ready to ship.", gamelog.Tag("item", name), query.FormatCurrency(bid))).
		Log()
}
