package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// auctionBoxRawID はオークション箱の raw prop の id。テンプレートで配置されたこの prop を
// システムが見つけてオークション箱に仕立てる。
const auctionBoxRawID = "auction_box"

// AuctionDemoSystem はオークション箱に収納された品を競売にかける。品を入れると入札が始まり、
// ターン経過で決着する。落札されれば手取りを財布へ入れ、実績を履歴に残して品を消す。
// 落札されなければ再入札する。箱から取り出された品は競売を解く。
// オークション箱が無い通常プレイでは即座に何もしない。
type AuctionDemoSystem struct{}

// String はシステム名を返す
func (sys AuctionDemoSystem) String() string {
	return "AuctionDemoSystem"
}

// Update はオークション箱の中身をターン経過で競売する
func (sys *AuctionDemoSystem) Update(world w.World) error {
	// テンプレートで配置された auction_box prop をオークション箱に仕立てる
	markAuctionBoxes(world)

	boxes := auctionBoxes(world)
	if len(boxes) == 0 {
		return nil
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	now := int(query.GetGameTime(world).TotalTurns)

	// 箱の外へ出た品は競売を解く。プレイヤーが取り出した品は競売を止める
	releaseRetrievedListings(world, boxes)

	// 各箱の品を競売する。GetStorageItems はスライスを返すので反復中に削除してよい
	for _, box := range boxes {
		for _, item := range query.GetStorageItems(world, box) {
			processAuctionItem(world, player, box, item, now)
		}
	}
	return nil
}

// processAuctionItem は1品の競売を進める。未出品なら開始入札で競売を始める。以後は1ターンに1回、
// 入札が来れば現在値を上げて延長し、来なければ落札を確定してその現在値で売る。
func processAuctionItem(world w.World, player, box, item ecs.Entity, now int) {
	if !world.Components.AuctionListing.Has(item) {
		bid := query.AuctionOpeningBid(world, item)
		world.Components.AuctionListing.Add(item, &gc.AuctionListing{CurrentBid: bid, LastTurn: now})
		logAuctionListed(world, query.GetEntityName(item, world), bid)
		return
	}
	l := world.Components.AuctionListing.Get(item)
	if now == l.LastTurn {
		return // 同じターンでは1回だけ判定する
	}
	l.LastTurn = now

	// 入札が来る限り延長する。来たら現在値を上げる
	if world.Config.RNG.Float64() < query.AuctionBidChance {
		l.CurrentBid += query.AuctionRaise(world, item)
		return
	}

	// 入札が止まった。落札を確定してその現在値で売る
	name := query.GetEntityName(item, world)
	bid := l.CurrentBid
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	net := bid - ship - fee
	if cerr := query.AddCurrency(world, player, net); cerr != nil {
		return
	}
	appendAuctionRecord(world, name, bid, ship, fee, net, now)
	world.ECS.RemoveEntity(item)
	markStorageWeightDirty(world, box)
	logAuctionSold(world, name, net)
}

// auctionBoxes はオークション箱のエンティティを集めて返す。
func auctionBoxes(world w.World) []ecs.Entity {
	var boxes []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionBox](world.ECS).Query()
	for q.Next() {
		boxes = append(boxes, q.Entity())
	}
	return boxes
}

// releaseRetrievedListings はオークション箱の中に無い出品を解く。取り出された品の競売を止める。
func releaseRetrievedListings(world w.World, boxes []ecs.Entity) {
	boxSet := make(map[ecs.Entity]bool, len(boxes))
	for _, h := range boxes {
		boxSet[h] = true
	}
	var toRelease []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		inBox := world.Components.LocationInStorage.Has(e) && boxSet[world.Components.LocationInStorage.Get(e).Owner]
		if !inBox {
			toRelease = append(toRelease, e)
		}
	}
	for _, e := range toRelease {
		world.Components.AuctionListing.Remove(e)
	}
}

// appendAuctionRecord は出荷実績を履歴に1件足す。
func appendAuctionRecord(world w.World, name string, bid, ship, fee, net, turn int) {
	history := query.GetAuctionHistory(world)
	history.Records = append(history.Records, gc.AuctionRecord{
		Name: name, Bid: bid, Ship: ship, Fee: fee, Net: net, Turn: turn,
	})
}

// markStorageWeightDirty は収納の重量再計算を要求する。中身が減ったので次フレームで引き直す。
func markStorageWeightDirty(world w.World, storage ecs.Entity) {
	if !world.Components.WeightDirty.Has(storage) {
		world.Components.WeightDirty.Add(storage, &gc.WeightDirty{})
	}
}

// markAuctionBoxes はテンプレートで配置された auction_box prop をオークション箱に仕立てる。
// 収納由来の相互作用を発送でなく専用のオークションに差し替え、識別マーカーを付ける。
// auction_box 以外の prop には触れないので通常プレイに影響しない。
func markAuctionBoxes(world w.World) {
	var toMark []ecs.Entity
	q := ecs.NewFilter1[gc.RawID](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.RawID.Get(e).ID != auctionBoxRawID || world.Components.AuctionBox.Has(e) {
			continue
		}
		toMark = append(toMark, e)
	}
	for _, e := range toMark {
		world.Components.AuctionBox.Add(e, &gc.AuctionBox{})
		if world.Components.Interactable.Has(e) {
			world.Components.Interactable.Get(e).Interactions = []gc.InteractionKind{gc.InteractionAuction}
		} else {
			world.Components.Interactable.Add(e, &gc.Interactable{Interactions: []gc.InteractionKind{gc.InteractionAuction}})
		}
	}
}

func logAuctionListed(world w.World, name string, opening int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s went up for auction. Opening bid %s.", gamelog.Tag("item", name), query.FormatCurrency(opening))).
		Log()
}

func logAuctionSold(world w.World, name string, net int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s was won. You got %s.", gamelog.Tag("item", name), query.FormatCurrency(net))).
		Log()
}
