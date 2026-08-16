package systems

import (
	"testing"

	gc "github.com/kijimaD/ruins/internal/components"
	"github.com/kijimaD/ruins/internal/consts"
	"github.com/kijimaD/ruins/internal/testutil"
	"github.com/kijimaD/ruins/internal/world/lifecycle"
	"github.com/kijimaD/ruins/internal/world/query"
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

func TestStartAuctionListing_連番を採番して出品する(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	_, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	first, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	second, err := lifecycle.SpawnBackpackItem(world, "shovel", 1)
	require.NoError(t, err)

	n1 := query.StartAuctionListing(world, first, 0)
	n2 := query.StartAuctionListing(world, second, 0)

	assert.Equal(t, 1, n1, "最初の出品は #1")
	assert.Equal(t, 2, n2, "次の出品は #2")
	require.True(t, world.Components.AuctionListing.Has(first), "タグを貼ると出品中になる")
	assert.Positive(t, world.Components.AuctionListing.Get(first).CurrentBid, "開始入札で現在値が付く")
}

func TestAuctionDemoSystem_出品中の品はターン経過で落札されるが入金はしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	before := query.GetCurrency(world, player)
	query.StartAuctionListing(world, item, int(query.GetGameTime(world).TotalTurns))

	sys := &AuctionDemoSystem{}
	sold := false
	for i := 0; i < 200 && !sold; i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
		sold = world.Components.AuctionSold.Has(item)
	}

	require.True(t, sold, "入札が止まったターンに落札が確定する")
	assert.True(t, world.ECS.Alive(item), "落札しても品は消えず落札済みで残る")
	assert.False(t, world.Components.AuctionListing.Has(item), "落札で出品中は外れる")
	assert.Equal(t, before, query.GetCurrency(world, player), "落札だけでは入金しない")
	assert.Empty(t, query.GetAuctionHistory(world).Records, "出荷するまで履歴には残らない")
}

func TestShipStagedItems_落札済みは入金し未落札は消えるだけ(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	station, err := lifecycle.SpawnProp(world, "shipping_station", 6, 6)
	require.NoError(t, err)
	won, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	junk, err := lifecycle.SpawnBackpackItem(world, "shovel", 1)
	require.NoError(t, err)
	query.StartAuctionListing(world, won, int(query.GetGameTime(world).TotalTurns))

	// won だけ落札まで進める。junk はタグを貼らないので未出品のまま
	sys := &AuctionDemoSystem{}
	for i := 0; i < 200 && !world.Components.AuctionSold.Has(won); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	require.True(t, world.Components.AuctionSold.Has(won), "won は落札済みになる")

	// 落札済みと未落札の両方を積荷へ積む
	require.NoError(t, lifecycle.MoveToStorage(world, won, station))
	require.NoError(t, lifecycle.MoveToStorage(world, junk, station))

	bid := world.Components.AuctionSold.Get(won).Bid
	expectedNet := bid - query.AuctionShippingCost(world, won) - query.AuctionFee(bid)
	before := query.GetCurrency(world, player)

	shipped, paid, total := query.ShipStagedItems(world, station, player, 42)

	assert.Equal(t, 2, shipped, "積んだ品はすべて出荷される")
	assert.Equal(t, 1, paid, "入金されるのは落札済みだけ")
	assert.Equal(t, expectedNet, total, "手取りは落札済みの分だけ")
	assert.Equal(t, before+expectedNet, query.GetCurrency(world, player), "落札済みの手取りが入金される")
	assert.False(t, world.ECS.Alive(won), "落札済みは出荷で手放す")
	assert.False(t, world.ECS.Alive(junk), "未落札も出荷され金にならず消える")
	records := query.GetAuctionHistory(world).Records
	require.Len(t, records, 1, "履歴に残るのは落札済みだけ")
	assert.Equal(t, 42, records[0].Turn, "出荷したターンを記録する")
	assert.Equal(t, expectedNet, records[0].Net, "手取りを記録する")
}

func TestAuctionDemoSystem_積荷はターン経過で自動出荷される(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	station, err := lifecycle.SpawnProp(world, "shipping_station", 6, 6)
	require.NoError(t, err)
	item, err := lifecycle.SpawnBackpackItem(world, "angel_sword", 1)
	require.NoError(t, err)
	query.StartAuctionListing(world, item, int(query.GetGameTime(world).TotalTurns))

	sys := &AuctionDemoSystem{}
	for i := 0; i < 200 && !world.Components.AuctionSold.Has(item); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}
	require.True(t, world.Components.AuctionSold.Has(item), "落札済みになる")

	// 落札済みを積荷へ積む。以後はプレイヤーの操作なしにターン経過で出荷される
	require.NoError(t, lifecycle.MoveToStorage(world, item, station))
	bid := world.Components.AuctionSold.Get(item).Bid
	expectedNet := bid - query.AuctionShippingCost(world, item) - query.AuctionFee(bid)
	before := query.GetCurrency(world, player)

	// 出荷は10ターンに1回の集荷なので、次の集荷まで数ターンかかる
	for i := 0; i < auctionShipInterval+1 && world.ECS.Alive(item); i++ {
		query.GetGameTime(world).Advance()
		require.NoError(t, sys.Update(world))
	}

	assert.False(t, world.ECS.Alive(item), "積荷は集荷のターンに自動出荷され手放す")
	assert.Equal(t, before+expectedNet, query.GetCurrency(world, player), "自動出荷で手取りが入金される")
	require.Len(t, query.GetAuctionHistory(world).Records, 1, "出荷実績が履歴に残る")
}

func TestShipStagedItems_積荷が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	player, err := lifecycle.SpawnPlayer(world, consts.Coord[consts.Tile]{X: 10, Y: 10}, "ash")
	require.NoError(t, err)
	station, err := lifecycle.SpawnProp(world, "shipping_station", 6, 6)
	require.NoError(t, err)
	before := query.GetCurrency(world, player)

	shipped, paid, total := query.ShipStagedItems(world, station, player, 0)

	assert.Zero(t, shipped, "積荷が無ければ0件")
	assert.Zero(t, paid, "入金も0件")
	assert.Zero(t, total, "手取りも0")
	assert.Equal(t, before, query.GetCurrency(world, player), "所持金は変わらない")
}

func TestMarkShippingStations_専用propを出荷場所にする(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	station, err := lifecycle.SpawnProp(world, "shipping_station", 5, 5)
	require.NoError(t, err)

	markShippingStations(world)

	require.True(t, world.Components.AuctionStation.Has(station), "shipping_station prop は出荷場所になる")
	it := world.Components.Interactable.Get(station)
	assert.Equal(t, []gc.InteractionKind{gc.InteractionAuction}, it.Interactions, "出荷場所メニューを開く相互作用が付く")
}

func TestMarkShippingStations_他のpropには触れない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	crate, err := lifecycle.SpawnProp(world, "wooden_crate", 5, 5)
	require.NoError(t, err)

	markShippingStations(world)

	assert.False(t, world.Components.AuctionStation.Has(crate), "shipping_station 以外の prop には触れない")
}

func TestAuctionDemoSystem_出品が無ければ何もしない(t *testing.T) {
	t.Parallel()
	world := testutil.InitTestWorld(t)

	sys := &AuctionDemoSystem{}
	assert.NoError(t, sys.Update(world), "出品中の品が無ければ即座に終わる")
}
