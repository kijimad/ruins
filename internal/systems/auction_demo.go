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

// processAuctionItem は1品の競売を進める。未出品なら入札を始め、決着ターンに達したら落札判定する。
func processAuctionItem(world w.World, player, box, item ecs.Entity, now int) {
	if !world.Components.AuctionListing.Has(item) {
		world.Components.AuctionListing.Add(item, &gc.AuctionListing{ResolveTurn: now + query.AuctionTurns})
		logAuctionListed(world, query.GetEntityName(item, world), query.AuctionTurns)
		return
	}
	l := world.Components.AuctionListing.Get(item)
	if now < l.ResolveTurn {
		return
	}

	// 落札されるかはパラメータ次第
	if world.Config.RNG.Float64() >= query.AuctionSaleChance(world, item) {
		l.ResolveTurn = now + query.AuctionTurns
		logAuctionReauction(world, query.GetEntityName(item, world))
		return
	}

	name := query.GetEntityName(item, world)
	bid := query.AuctionBid(world, item)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	net := bid - ship - fee
	if cerr := query.AddCurrency(world, player, net); cerr != nil {
		return
	}
	appendAuctionRecord(world, name, bid, ship, fee, net, now)
	world.ECS.RemoveEntity(item)
	markStorageWeightDirty(world, box)
	logAuctionSold(world, name, bid, ship, fee, net)
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

func logAuctionListed(world w.World, name string, turns int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s went up for auction. It resolves in %d turns.", gamelog.Tag("item", name), turns)).
		Log()
}

func logAuctionReauction(world w.World, name string) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "%s did not sell. Re-auction started.", gamelog.Tag("item", name))).
		Log()
}

func logAuctionSold(world w.World, name string, bid, ship, fee, net int) {
	store := query.GetGameLog(world)
	if net < 0 {
		gamelog.New(store).Markup(query.T(world, "Sold %s at a LOSS. net %d, bid %d, ship %d, fee %d",
			gamelog.Tag("item", name), net, bid, ship, fee)).Log()
		return
	}
	gamelog.New(store).Markup(query.T(world, "Sold %s. net %d, bid %d, ship %d, fee %d",
		gamelog.Tag("item", name), net, bid, ship, fee)).Log()
}
