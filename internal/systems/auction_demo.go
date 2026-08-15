package systems

import (
	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// auctionHouseRawID はオークションハウスの raw prop の id。テンプレートで配置されたこの prop を
// システムが見つけてオークションハウスに仕立てる。
const auctionHouseRawID = "auction_house"

// AuctionDemoSystem はオークションハウスに収納された品を競売にかける。品を入れると入札が始まり、
// ターン経過で決着する。落札されれば手取りを財布へ入れ、実績を履歴に残して品を消す。
// 落札されなければ再入札する。ハウスから取り出された品は競売を解く。
// オークションハウスが無い通常プレイでは即座に何もしない。
type AuctionDemoSystem struct{}

// String はシステム名を返す
func (sys AuctionDemoSystem) String() string {
	return "AuctionDemoSystem"
}

// Update はオークションハウスの中身をターン経過で競売する
func (sys *AuctionDemoSystem) Update(world w.World) error {
	// テンプレートで配置された auction_house prop をオークションハウスに仕立てる
	markAuctionHouses(world)

	houses := auctionHouses(world)
	if len(houses) == 0 {
		return nil
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	now := int(query.GetGameTime(world).TotalTurns)

	// ハウスの外へ出た品は競売を解く。プレイヤーが取り出した品は競売を止める
	releaseRetrievedListings(world, houses)

	// 各ハウスの品を競売する。GetStorageItems はスライスを返すので反復中に削除してよい
	for _, house := range houses {
		for _, item := range query.GetStorageItems(world, house) {
			processAuctionItem(world, player, house, item, now)
		}
	}
	return nil
}

// processAuctionItem は1品の競売を進める。未出品なら入札を始め、決着ターンに達したら落札判定する。
func processAuctionItem(world w.World, player, house, item ecs.Entity, now int) {
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
	markStorageWeightDirty(world, house)
	logAuctionSold(world, name, bid, ship, fee, net)
}

// auctionHouses はオークションハウスのエンティティを集めて返す。
func auctionHouses(world w.World) []ecs.Entity {
	var houses []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionHouse](world.ECS).Query()
	for q.Next() {
		houses = append(houses, q.Entity())
	}
	return houses
}

// releaseRetrievedListings はオークションハウスの中に無い出品を解く。取り出された品の競売を止める。
func releaseRetrievedListings(world w.World, houses []ecs.Entity) {
	houseSet := make(map[ecs.Entity]bool, len(houses))
	for _, h := range houses {
		houseSet[h] = true
	}
	var toRelease []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		inHouse := world.Components.LocationInStorage.Has(e) && houseSet[world.Components.LocationInStorage.Get(e).Owner]
		if !inHouse {
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

// markAuctionHouses はテンプレートで配置された auction_house prop をオークションハウスに仕立てる。
// 収納由来の相互作用を発送でなく専用のオークションに差し替え、識別マーカーを付ける。
// auction_house 以外の prop には触れないので通常プレイに影響しない。
func markAuctionHouses(world w.World) {
	var toMark []ecs.Entity
	q := ecs.NewFilter1[gc.RawID](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if world.Components.RawID.Get(e).ID != auctionHouseRawID || world.Components.AuctionHouse.Has(e) {
			continue
		}
		toMark = append(toMark, e)
	}
	for _, e := range toMark {
		world.Components.AuctionHouse.Add(e, &gc.AuctionHouse{})
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
