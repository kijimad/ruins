package systems

import (
	"fmt"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/gamelog"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
)

// auctionDemoItems は種まきで床に撒く品の raw id。軽くて高い品と重くて安い品を混ぜ、
// 価値がそのまま利益にならないこと、すなわち重い安物は発送料で赤字になることを体感させる。
var auctionDemoItems = []string{
	"angel_sword", "green_sword", "slender_rapier", "nunchaku",
	"fire_extinguisher", "shovel", "block_hammer", "hammer",
}

// AuctionDemoSystem は通信販売デモで出品中の品の落札を毎ターン進める。
// ターンが ResolveTurn に達した出品を落札済みにし、落札額を確定してログへ出す。
// 出品が1つも無い通常プレイでは実質何もしない。
type AuctionDemoSystem struct{}

// String はシステム名を返す
func (sys AuctionDemoSystem) String() string {
	return "AuctionDemoSystem"
}

// Update はプレイヤーが1歩動くたびに出品の残り歩数を減らし、0 になった品を落札済みにする。
// ターン制の進行に依存せず移動そのものを進行の単位にするので、どの場面でも歩けば必ず進む。
// 落札前は残り歩数をカウントダウンで告知し、進んでいることを見えるようにする。
// 出品が1つも無い通常プレイでは即座に何もしない。
func (sys *AuctionDemoSystem) Update(world w.World) error {
	if !anyAuctionListing(world) {
		return nil
	}
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	if !world.Components.GridElement.Has(player) {
		return nil
	}
	pos := world.Components.GridElement.Get(player).Coord

	// プレイヤーの前回位置と比べて、1歩動いたかを判定する
	clock := query.GetAuctionClock(world)
	moved := clock.HasLast && (int(pos.X) != clock.LastX || int(pos.Y) != clock.LastY)
	clock.LastX, clock.LastY, clock.HasLast = int(pos.X), int(pos.Y), true

	// StepsLeft・Won・Bid・Announced の書き換えは構造変更しないのでクエリ反復中に行う。
	// 落札されなかった出品の出品解除は構造変更なので、後でまとめて行う
	var unsold []ecs.Entity
	listings := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	for listings.Next() {
		item := listings.Entity()
		l := world.Components.AuctionListing.Get(item)
		if l.Won {
			continue
		}
		if moved && l.StepsLeft > 0 {
			l.StepsLeft--
		}
		if l.StepsLeft <= 0 {
			// 落札されるかはパラメータ次第。落札されなければ在庫として残る
			if world.Config.RNG.Float64() < query.AuctionSaleChance(world, item) {
				l.Won = true
				l.Bid = query.AuctionBid(world, item, l.Slow)
				logAuctionWon(world, l.ID, query.GetEntityName(item, world), l.Bid)
			} else {
				logAuctionUnsold(world, l.ID, query.GetEntityName(item, world))
				unsold = append(unsold, item)
			}
			continue
		}
		if l.StepsLeft != l.Announced {
			l.Announced = l.StepsLeft
			logAuctionCountdown(world, l.ID, query.GetEntityName(item, world), l.StepsLeft)
		}
	}
	// 落札されなかった品は出品を解いて在庫に戻す。再出品すると識別子は採番し直しになる
	for _, item := range unsold {
		world.Components.AuctionListing.Remove(item)
	}
	return nil
}

// anyAuctionListing は出品中の品が1つでもあるかを返す。
func anyAuctionListing(world w.World) bool {
	q := ecs.NewFilter1[gc.AuctionListing](world.ECS).Query()
	if q.Next() {
		q.Close()
		return true
	}
	return false
}

// SeedAuctionDemo はプレイヤーのいる床に通信販売デモを仕込む。品を床に散らして出品できるようにし、
// 発送台を隣に置く。品は出品してターン経過で落札し、落札品を発送台へ運ぶと精算される。
func SeedAuctionDemo(world w.World) error {
	player, err := query.GetPlayerEntity(world)
	if err != nil {
		return err
	}
	if !world.Components.GridElement.Has(player) {
		return fmt.Errorf("cannot seed auction demo because the player has no position")
	}
	base := world.Components.GridElement.Get(player).Coord

	// プレイヤーの周囲の開けたタイルに散らす。既存の木箱と NPC、東隣に並べる発券機とポータルを避ける。
	// 近い品と遠い品を混ぜ、置き場所と動線が取り出し速度に効くようにする
	offsets := []consts.Coord[consts.Tile]{
		{X: 3, Y: 0}, {X: -3, Y: 0}, {X: 0, Y: 2}, {X: 0, Y: -2},
		{X: 3, Y: 2}, {X: -3, Y: 2}, {X: 2, Y: -3}, {X: -2, Y: -3},
	}
	for i, id := range auctionDemoItems {
		p := base.Add(offsets[i%len(offsets)])
		item, serr := lifecycle.SpawnFieldItem(world, id, p.X, p.Y, 1)
		if serr != nil {
			return serr
		}
		// 既定の拾得に加えて出品タグを貼れるようにする
		addInteraction(world, item, gc.InteractionApplyListingTag)
	}

	// 発券機とポータルをプレイヤーの東隣に並べて置く。木箱と紛れない専用プロップにする。
	// 東へ歩けば発券機、その先にポータル、と順に見つかるようにする
	if serr := spawnAuctionProp(world, "control_panel", base.X+1, base.Y, gc.InteractionDispenseListingTag); serr != nil {
		return serr
	}
	if serr := spawnAuctionProp(world, "warp_next", base.X+2, base.Y, gc.InteractionShip); serr != nil {
		return serr
	}

	logAuctionIntro(world)
	return nil
}

// addInteraction は既存の Interactable に相互作用を1つ足す。無ければ作る。
func addInteraction(world w.World, e ecs.Entity, kind gc.InteractionKind) {
	if !world.Components.Interactable.Has(e) {
		world.Components.Interactable.Add(e, &gc.Interactable{Interactions: []gc.InteractionKind{kind}})
		return
	}
	it := world.Components.Interactable.Get(e)
	it.Interactions = append(it.Interactions, kind)
}

// spawnAuctionProp はデモ用のプロップを1つ置く。既存プロップを流用し、相互作用を指定のものだけに差し替える。
func spawnAuctionProp(world w.World, propName string, x, y consts.Tile, kinds ...gc.InteractionKind) error {
	prop, err := lifecycle.SpawnProp(world, propName, x, y)
	if err != nil {
		return err
	}
	if world.Components.Interactable.Has(prop) {
		world.Components.Interactable.Get(prop).Interactions = kinds
	} else {
		world.Components.Interactable.Add(prop, &gc.Interactable{Interactions: kinds})
	}
	return nil
}

func logAuctionIntro(world w.World) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "Mail-order demo. The tag dispenser and the shipping portal stand to the east. Issue a listing tag, apply it to an item, walk to run its auction, then ship at the portal.")).
		Log()
}

func logAuctionCountdown(world w.World, id int, name string, remaining int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "L-%d %s: %d steps until it is won.", id, gamelog.Tag("item", name), remaining)).
		Log()
}

func logAuctionUnsold(world w.World, id int, name string) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "L-%d %s did not sell. It stays as stock. Apply a new tag to re-list it.",
			id, gamelog.Tag("item", name))).
		Log()
}

func logAuctionWon(world w.World, id int, name string, bid int) {
	gamelog.New(query.GetGameLog(world)).
		Markup(query.T(world, "L-%d %s was won at bid %d. Pick it up and place it in the portal to ship.",
			id, gamelog.Tag("item", name), bid)).
		Log()
}
