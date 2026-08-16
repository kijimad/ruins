package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	w "github.com/kijimaD/ruins/internal/world"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
	"github.com/mlange-42/ark/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuctionPrice_重い安物は発送料で赤字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// fire_extinguisher は価値40で重量5kg。発送料が落札額を食って手取りが負になる
	item, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionOpeningBid(world, item)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, ship, "重量に応じた発送料がかかる")
	assert.Negative(t, bid-ship-fee, "基準価値が低く重い品は開始入札の時点で発送料に負ける")
}

func TestAuctionPrice_軽い高額品は黒字になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// angel_sword は価値500で重量2kg。軽く高価なので手取りが出る
	item, err := lifecycle.SpawnFieldItem(world, "angel_sword", 5, 5, 1)
	require.NoError(t, err)

	bid := query.AuctionOpeningBid(world, item)
	ship := query.AuctionShippingCost(world, item)
	fee := query.AuctionFee(bid)
	assert.Positive(t, bid-ship-fee, "軽く価値が高い品は開始入札の時点で手取りが出る")
}

func TestAuctionRaise_入札の上げ幅は必ず1以上になる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	// 入札が来たのに現在値が動かない事態を防ぐため、上げ幅は最低1を保証する
	item, err := lifecycle.SpawnFieldItem(world, "fire_extinguisher", 5, 5, 1)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, query.AuctionRaise(world, item), 1, "上げ幅は必ず1以上")
}

func TestAuctionDemoSystem_収納した品は競売にかかる(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	house := newTestAuctionBox(t, world, 11, 10)
	item, err := lifecycle.SpawnStorageItem(world, "angel_sword", 1, house)
	require.NoError(t, err)

	sys := &AuctionDemoSystem{}
	require.NoError(t, sys.Update(world))

	require.True(t, world.Components.AuctionListing.Has(item), "収納した品は競売にかかる")
	assert.Positive(t, world.Components.AuctionListing.Get(item).CurrentBid, "開始入札で現在値が付く")
}

func TestAuctionDemoSystem_ターン経過で決着し実績が履歴に残る(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	house := newTestAuctionBox(t, world, 11, 10)
	// shovel は安く落札されやすいので、再入札を繰り返せば必ず売れる
	item, err := lifecycle.SpawnStorageItem(world, "shovel", 1, house)
	require.NoError(t, err)

	sys := &AuctionDemoSystem{}
	sold := false
	for i := 0; i < 80 && !sold; i++ {
		require.NoError(t, sys.Update(world))
		query.GetGameTime(world).Advance()
		sold = !world.ECS.Alive(item)
	}

	assert.True(t, sold, "ターン経過で競売が決着し品が売れる")
	assert.NotEmpty(t, query.GetAuctionHistory(world).Records, "出荷実績が履歴に残る")
}

func TestAuctionDemoSystem_箱から取り出すと競売を解く(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	house := newTestAuctionBox(t, world, 11, 10)
	item, err := lifecycle.SpawnStorageItem(world, "angel_sword", 1, house)
	require.NoError(t, err)

	sys := &AuctionDemoSystem{}
	require.NoError(t, sys.Update(world))
	require.True(t, world.Components.AuctionListing.Has(item), "収納中は競売中")

	// 取り出す。持ち物へ移すと競売が解かれる
	require.NoError(t, lifecycle.MoveToBackpack(world, item, player))
	require.NoError(t, sys.Update(world))

	assert.False(t, world.Components.AuctionListing.Has(item), "取り出した品は競売が解かれる")
}

func TestMarkAuctionBoxes_専用propをオークション箱にする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	house, err := lifecycle.SpawnProp(world, "auction_box", 5, 5)
	require.NoError(t, err)

	markAuctionBoxes(world)

	require.True(t, world.Components.AuctionBox.Has(house), "auction_box prop はオークション箱になる")
	it := world.Components.Interactable.Get(house)
	require.Len(t, it.Interactions, 1, "相互作用はオークションだけになる")
	assert.Equal(t, gc.InteractionAuction, it.Interactions[0], "専用のオークション相互作用が付く")
}

func TestMarkAuctionBoxes_他のpropには触れない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 5, 5)
	require.NoError(t, err)

	markAuctionBoxes(world)

	assert.False(t, world.Components.AuctionBox.Has(crate), "auction_box 以外の prop には触れない")
}

func TestAuctionDemoSystem_箱が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sys := &AuctionDemoSystem{}
	assert.NoError(t, sys.Update(world), "オークション箱が無ければ即座に終わる")
}

// newTestAuctionBox はテスト用にオークション箱を1つ作る。
func newTestAuctionBox(t *testing.T, world w.World, x, y consts.Tile) ecs.Entity {
	t.Helper()
	house, err := lifecycle.SpawnProp(world, "wooden_crate", x, y)
	require.NoError(t, err)
	world.Components.AuctionBox.Add(house, &gc.AuctionBox{})
	return house
}
