package activity

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// executeDispenseListingTag は発券機で出品タグを1枚、持ち物に発行する。タグは同一の消耗品で
// 個体の区別を持たず、貼るときに識別子が決まる。
func executeDispenseListingTag(actor ecs.Entity, world w.World) (*ActionResult, error) {
	spawnListingTag(world, actor)
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Issued a listing tag. Apply it to an item to start its auction.")).
		Log()
	return &ActionResult{Success: true, Message: "issued listing tag"}, nil
}

// executeApplyListingTag は品に出品タグを貼る。持っている出品タグを1枚消費し、そのとき一意識別子を
// 採番して入札を始める。既に出品済みか、出品タグを持っていなければ何もしない。
func executeApplyListingTag(actor ecs.Entity, target ecs.Entity, world w.World) (*ActionResult, error) {
	if world.Components.AuctionListing.Has(target) {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "%s is already listed.", gamelog.Tag("item", query.GetEntityName(target, world)))).
			Log()
		return &ActionResult{Success: false, Message: "already listed"}, nil
	}
	tagEntity, ok := anyHeldTag(world, actor)
	if !ok {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "No listing tag in hand. Issue one at the dispenser.")).
			Log()
		return &ActionResult{Success: false, Message: "no listing tag"}, nil
	}

	// 貼るときに一意識別子を採番する。タグ自体は区別を持たない
	clock := query.GetAuctionClock(world)
	id := clock.NextTagID
	clock.NextTagID++

	world.Components.AuctionListing.Add(target, &gc.AuctionListing{ID: id, StepsLeft: query.AuctionSlowSteps, Slow: true, Announced: -1})
	world.ECS.RemoveEntity(tagEntity)

	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Applied a listing tag to %s. Its auction started as %s.",
			gamelog.Tag("item", query.GetEntityName(target, world)), auctionTagLabel(id))).
		Log()
	return &ActionResult{Success: true, Message: "listed for auction"}, nil
}

// executeShip はポータルでの発送。持ち物の落札済みの品を精算し、財布へ手取りを入れる。
// 発送料は重量に、手数料は落札額に比例する。まだ落札していない品には何もしない。
func executeShip(actor ecs.Entity, world w.World) (*ActionResult, error) {
	// collect-then-mutate: 落札済みの持ち物を先に集めてから精算する
	var wonItems []ecs.Entity
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if !world.Components.AuctionListing.Get(e).Won {
			continue
		}
		if !isInBackpackOf(world, e, actor) {
			continue
		}
		wonItems = append(wonItems, e)
	}
	if len(wonItems) == 0 {
		gamelog.New(query.GetGameLog(world)).
			Markup(query.T(world, "No won item to place. Carry a won item here.")).
			Log()
		return &ActionResult{Success: false, Message: "nothing to ship"}, nil
	}

	for _, item := range wonItems {
		listing := world.Components.AuctionListing.Get(item)
		id := listing.ID
		bid := listing.Bid
		name := query.GetEntityName(item, world)
		ship := query.AuctionShippingCost(world, item)
		fee := query.AuctionFee(bid)
		net := bid - ship - fee
		if err := query.AddCurrency(world, actor, net); err != nil {
			return nil, err
		}
		world.ECS.RemoveEntity(item)
		logAuctionShip(world, id, name, bid, ship, fee, net)
	}
	return &ActionResult{Success: true, Message: "shipped"}, nil
}

// spawnListingTag は出品タグを1枚、持ち物として発行する。タグは同一の消耗品で識別子を持たない。
func spawnListingTag(world w.World, owner ecs.Entity) {
	e := world.ECS.NewEntity()
	world.Components.AuctionTag.Add(e, &gc.AuctionTag{})
	world.Components.Weight.Add(e, &gc.Weight{})
	world.Components.Value.Add(e, &gc.Value{})
	world.Components.Name.Add(e, &gc.Name{Name: "Listing tag"})
	world.Components.LocationInBackpack.Add(e, &gc.LocationInBackpack{Owner: owner})
}

// anyHeldTag は持ち物の出品タグを1枚返す。タグは同一なのでどれでもよい。
func anyHeldTag(world w.World, actor ecs.Entity) (ecs.Entity, bool) {
	q := ecs.NewFilter1[gc.AuctionTag](world.ECS).Query()
	for q.Next() {
		e := q.Entity()
		if isInBackpackOf(world, e, actor) {
			q.Close()
			return e, true
		}
	}
	return gc.InvalidEntity, false
}

func isInBackpackOf(world w.World, entity ecs.Entity, owner ecs.Entity) bool {
	return world.Components.LocationInBackpack.Has(entity) &&
		world.Components.LocationInBackpack.Get(entity).Owner == owner
}

func auctionTagLabel(id int) string {
	return fmt.Sprintf("L-%d", id)
}

func logAuctionShip(world w.World, id int, name string, bid, ship, fee, net int) {
	store := query.GetGameLog(world)
	if net < 0 {
		gamelog.New(store).Markup(query.T(world, "Shipped %s %s at a LOSS. net %d, bid %d, ship %d, fee %d",
			auctionTagLabel(id), gamelog.Tag("item", name), net, bid, ship, fee)).Log()
		return
	}
	gamelog.New(store).Markup(query.T(world, "Shipped %s %s. net %d, bid %d, ship %d, fee %d",
		auctionTagLabel(id), gamelog.Tag("item", name), net, bid, ship, fee)).Log()
}
