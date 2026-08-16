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

// AuctionDemoSystem はタグを貼って出品された品の競売をターン経過で進める。
// 入札が来る限り現在値を上げて延長し、入札が止まったターンに現在値で落札を確定する。
// 落札しても入金はせず、品を落札済みへ移して出荷場所での出荷を待たせる。
// 出品中の品が無い通常プレイでは即座に何もしない。
type AuctionDemoSystem struct{}

// String はシステム名を返す
func (sys AuctionDemoSystem) String() string {
	return "AuctionDemoSystem"
}

// Update は出品中の品をターン経過で競売する
func (sys *AuctionDemoSystem) Update(world w.World) error {
	// テンプレートで配置された shipping_station prop を出荷場所に仕立てる
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

	// 積荷はターン経過で自動出荷する。プレイヤーは落札済みを積むだけでよい
	autoShipStations(world, now)
	return nil
}

// autoShipStations は各出荷場所の積荷を自動で出荷する。落札済みだけ入金し履歴へ残す。
// 積める品は落札済みに限るので、積荷はそのまま精算される。
func autoShipStations(world w.World, now int) {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return
	}
	var stations []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionStation](world.ECS).Query()
	for q.Next() {
		stations = append(stations, q.Entity())
	}
	for _, station := range stations {
		shipped, _, total := query.ShipStagedItems(world, station, player, now)
		if shipped > 0 {
			logAutoShipped(world, shipped, total)
		}
	}
}

func logAutoShipped(world w.World, count, total int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Shipped %d items. You got %s.", count, query.FormatCurrency(total))).
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

	// 入札が止まった。落札を確定して落札済みへ移す。入金はまだしない
	number := l.Number
	bid := l.CurrentBid
	name := query.GetEntityName(item, world)
	world.Components.AuctionListing.Remove(item)
	world.Components.AuctionSold.Add(item, &gc.AuctionSold{Number: number, Bid: bid})
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
